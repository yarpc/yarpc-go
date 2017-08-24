// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package observability

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/yarpc/internal/pally/pallytest"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestDigester(t *testing.T) {
	const (
		goroutines = 10
		iterations = 100
	)

	expected := []byte{'f', 'o', 'o', 0, 'b', 'a', 'r'}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				d := newDigester()
				defer d.free()

				assert.Equal(t, 0, len(d.digest()), "Expected fresh digester to have no internal state.")
				assert.True(t, cap(d.digest()) > 0, "Expected fresh digester to have available capacity.")

				d.add("foo")
				d.add("bar")
				assert.Equal(
					t,
					string(expected),
					string(d.digest()),
					"Expected digest to be null-separated concatenation of inputs.",
				)
			}
		}()
	}

	wg.Wait()
}

func TestMiddlewareLogging(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "raw",
		Procedure:       "procedure",
		Headers:         transport.NewHeaders().With("password", "super-secret"),
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
		Body:            strings.NewReader("body"),
	}
	failed := errors.New("fail")

	baseFields := func() []zapcore.Field {
		return []zapcore.Field{
			zap.String("source", req.Caller),
			zap.String("dest", req.Service),
			zap.String("procedure", req.Procedure),
			zap.String("encoding", string(req.Encoding)),
			zap.String("routingKey", req.RoutingKey),
			zap.String("routingDelegate", req.RoutingDelegate),
		}
	}

	tests := []struct {
		desc           string
		err            error // downstream error
		applicationErr bool  // downstream application error
		wantFields     []zapcore.Field
	}{
		{
			desc: "no downstream errors",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(),
				zap.Skip(),
			},
		},
		{
			desc: "downstream transport error",
			err:  failed,
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(failed),
			},
		},
	}

	for _, tt := range tests {
		core, logs := observer.New(zapcore.DebugLevel)
		mw := NewMiddleware(zap.New(core), pally.NewRegistry(), NewNopContextExtractor())

		getLog := func() observer.LoggedEntry {
			entries := logs.TakeAll()
			require.Equal(t, 1, len(entries), "Unexpected number of logs written.")
			e := entries[0]
			e.Entry.Time = time.Time{}
			return e
		}

		checkErr := func(err error) {
			if tt.err != nil {
				assert.Error(t, err, "Expected an error from middleware.")
			} else {
				assert.NoError(t, err, "Unexpected error from middleware.")
			}
		}

		t.Run(tt.desc+", unary inbound", func(t *testing.T) {
			err := mw.Handle(
				context.Background(),
				req,
				&transporttest.FakeResponseWriter{},
				fakeHandler{tt.err, tt.applicationErr},
			)
			checkErr(err)
			logContext := append(baseFields(), zap.String("rpcType", "Unary"))
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Handled inbound request.",
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			res, err := mw.Call(context.Background(), req, fakeOutbound{err: tt.err})
			checkErr(err)
			if tt.err == nil {
				assert.NotNil(t, res, "Expected non-nil response if call is successful.")
			}
			logContext := append(baseFields(), zap.String("rpcType", "Unary"))
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Made outbound call.",
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", oneway inbound", func(t *testing.T) {
			err := mw.HandleOneway(context.Background(), req, fakeHandler{tt.err, false})
			checkErr(err)
			logContext := append(baseFields(), zap.String("rpcType", "Oneway"))
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Handled inbound request.",
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", oneway outbound", func(t *testing.T) {
			ack, err := mw.CallOneway(context.Background(), req, fakeOutbound{err: tt.err})
			checkErr(err)
			logContext := append(baseFields(), zap.String("rpcType", "Oneway"))
			logContext = append(logContext, tt.wantFields...)
			if tt.err == nil {
				assert.NotNil(t, ack, "Expected non-nil ack if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Made outbound call.",
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
	}
}

func TestMiddlewareMetrics(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  "raw",
		Procedure: "procedure",
		Body:      strings.NewReader("body"),
	}

	tests := []struct {
		desc               string
		err                error // downstream error
		applicationErr     bool  // downstream application error
		wantCalls          int
		wantSuccesses      int
		wantCallerFailures map[string]int
		wantServerFailures map[string]int
	}{
		{
			desc:          "no downstream errors",
			wantCalls:     1,
			wantSuccesses: 1,
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerrors.InvalidArgumentErrorf("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantCallerFailures: map[string]int{
				yarpcerrors.CodeInvalidArgument.String(): 1,
			},
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerrors.InternalErrorf("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				yarpcerrors.CodeInternal.String(): 1,
			},
		},
		{
			desc:          "unknown (unwrapped) error",
			err:           errors.New("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"unknown_strange": 1,
			},
		},
		{
			desc:          "custom error code error",
			err:           yarpcerrors.FromHeaders(yarpcerrors.Code(1000), "", "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"1000": 1,
			},
		},
	}

	for _, tt := range tests {
		validate := func(mw *Middleware) {
			key, free := getKey(req)
			edge := mw.graph.getEdge(key)
			free()
			assert.Equal(t, int64(tt.wantCalls), edge.calls.Load())
			assert.Equal(t, int64(tt.wantSuccesses), edge.successes.Load())
			for tagName, val := range tt.wantCallerFailures {
				assert.Equal(t, int64(val), edge.callerFailures.MustGet(tagName).Load())
			}
			for tagName, val := range tt.wantServerFailures {
				assert.Equal(t, int64(val), edge.serverFailures.MustGet(tagName).Load())
			}
		}
		t.Run(tt.desc+", unary inbound", func(t *testing.T) {
			mw := NewMiddleware(zap.NewNop(), pally.NewRegistry(), NewNopContextExtractor())
			mw.Handle(
				context.Background(),
				req,
				&transporttest.FakeResponseWriter{},
				fakeHandler{tt.err, tt.applicationErr},
			)
			validate(mw)
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			mw := NewMiddleware(zap.NewNop(), pally.NewRegistry(), NewNopContextExtractor())
			mw.Call(context.Background(), req, fakeOutbound{err: tt.err})
			validate(mw)
		})
	}
}

func TestUnaryInboundApplicationErrors(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
		Body:            strings.NewReader("body"),
	}

	expectedFields := []zapcore.Field{
		zap.String("source", req.Caller),
		zap.String("dest", req.Service),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("routingKey", req.RoutingKey),
		zap.String("routingDelegate", req.RoutingDelegate),
		zap.String("rpcType", "Unary"),
		zap.Duration("latency", 0),
		zap.Bool("successful", false),
		zap.Skip(),
		zap.String("error", "application_error"),
	}

	core, logs := observer.New(zap.DebugLevel)
	mw := NewMiddleware(zap.New(core), pally.NewRegistry(), NewNopContextExtractor())

	assert.NoError(t, mw.Handle(
		context.Background(),
		req,
		&transporttest.FakeResponseWriter{},
		fakeHandler{err: nil, applicationErr: true},
	), "Unexpected transport error.")

	expected := observer.LoggedEntry{
		Entry: zapcore.Entry{
			Level:   zapcore.DebugLevel,
			Message: "Handled inbound request.",
		},
		Context: expectedFields,
	}
	entries := logs.TakeAll()
	require.Equal(t, 1, len(entries), "Unexpected number of log entries written.")
	entry := entries[0]
	entry.Time = time.Time{}
	assert.Equal(t, expected, entry, "Unexpected log entry written.")
}

func TestMiddlewareStats(t *testing.T) {
	defer stubTime()()
	reg := pally.NewRegistry()
	mw := NewMiddleware(zap.NewNop(), reg, NewNopContextExtractor())

	err := mw.Handle(
		context.Background(),
		&transport.Request{
			Caller:          "caller",
			Service:         "service",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
			Body:            strings.NewReader("body"),
		},
		&transporttest.FakeResponseWriter{},
		fakeHandler{nil, false},
	)
	assert.NoError(t, err, "Unexpected transport error.")

	expected, err := ioutil.ReadFile("testdata/prom.txt")
	assert.NoError(t, err, "Unexpected error reading testdata.")

	pallytest.AssertPrometheus(t, reg, strings.TrimSpace(string(expected)))
}
