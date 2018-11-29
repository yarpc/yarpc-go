// Copyright (c) 2018 Uber Technologies, Inc.
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

package internalyarpcobservability

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaldigester"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestMiddlewareLogging(t *testing.T) {
	defer stubTime()()
	req := &yarpc.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "raw",
		Procedure:       "procedure",
		Headers:         yarpc.NewHeaders().With("password", "super-secret"),
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
	}
	reqBuf := yarpc.NewBufferString("body")
	failed := errors.New("fail")

	baseFields := func() []zapcore.Field {
		return []zapcore.Field{
			zap.String("source", req.Caller),
			zap.String("dest", req.Service),
			zap.String("transport", unknownIfEmpty(req.Transport)),
			zap.String("procedure", req.Procedure),
			zap.String("encoding", string(req.Encoding)),
			zap.String("routingKey", req.RoutingKey),
			zap.String("routingDelegate", req.RoutingDelegate),
		}
	}

	type test struct {
		desc            string
		err             error // downstream error
		applicationErr  error // downstream application error
		wantErrLevel    zapcore.Level
		wantInboundMsg  string
		wantOutboundMsg string
		wantFields      []zapcore.Field
	}

	tests := []test{
		{
			desc:            "no downstream errors",
			wantErrLevel:    zapcore.DebugLevel,
			wantInboundMsg:  "Handled inbound request.",
			wantOutboundMsg: "Made outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(),
				zap.Skip(),
			},
		},
		{
			desc:            "downstream transport error",
			err:             failed,
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(failed),
			},
		},
		{
			desc:            "no downstream error but with application error",
			applicationErr:  yarpcerror.New(yarpcerror.CodeUnknown, "error", yarpcerror.WithName("hello")),
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.: hello: error",
			wantOutboundMsg: "Error making outbound call.: hello: error",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.String("error", "hello"),
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{err: t.err, applicationErr: t.applicationErr}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{err: t.err, applicationErr: t.applicationErr}
	}

	for _, tt := range tests {
		core, logs := observer.New(zapcore.DebugLevel)
		mw := NewMiddleware(zap.New(core), metrics.New().Scope(), NewNopContextExtractor())

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
			res, _, err := mw.Handle(
				context.Background(),
				req,
				reqBuf,
				newHandler(tt),
			)

			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Unary"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantInboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else if tt.applicationErr != nil {
				assert.NotNil(t, res.ApplicationErrorInfo)
			} else {
				require.NotNil(t, res)
				assert.Nil(t, res.ApplicationErrorInfo)
			}
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			res, _, err := mw.Call(context.Background(), req, reqBuf, newOutbound(tt))
			checkErr(err)
			if tt.err == nil {
				assert.NotNil(t, res, "Expected non-nil response if call is successful.")
				if tt.applicationErr != nil {
					assert.NotNil(t, res.ApplicationErrorInfo)
				} else {
					assert.Nil(t, res.ApplicationErrorInfo)
				}
			}
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Unary"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantOutboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
			assert.Equal(t, tt.err, err)
		})

		// Application errors aren't applicable to streaming
		if tt.applicationErr != nil {
			continue
		}

		t.Run(tt.desc+", stream inbound", func(t *testing.T) {
			stream, err := yarpc.NewServerStream(&fakeStream{ctx: context.Background(), request: req})
			require.NoError(t, err)
			err = mw.HandleStream(stream, newHandler(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Streaming"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantInboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", stream outbound", func(t *testing.T) {
			clientStream, err := mw.CallStream(context.Background(), req, newOutbound(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Streaming"),
			)
			logContext = append(logContext, tt.wantFields...)
			if tt.err == nil {
				assert.NotNil(t, clientStream, "Expected non-nil clientStream if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantOutboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
	}
}

func TestMiddlewareMetrics(t *testing.T) {
	defer stubTime()()
	req := &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Transport: "",
		Encoding:  "raw",
		Procedure: "procedure",
	}
	reqBuf := yarpc.NewBufferString("body")

	type test struct {
		desc               string
		err                error // downstream error
		wantCalls          int
		wantSuccesses      int
		wantCallerFailures map[string]int
		wantServerFailures map[string]int
	}

	tests := []test{
		{
			desc:          "no downstream errors",
			wantCalls:     1,
			wantSuccesses: 1,
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerror.New(yarpcerror.CodeInvalidArgument, "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantCallerFailures: map[string]int{
				yarpcerror.CodeInvalidArgument.String(): 1,
			},
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerror.New(yarpcerror.CodeInternal, "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				yarpcerror.CodeInternal.String(): 1,
			},
		},
		{
			desc:          "unknown (unwrapped) error",
			err:           errors.New("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"unknown_internal_yarpc": 1,
			},
		},
		{
			desc:          "custom error code error",
			err:           yarpcerror.New(yarpcerror.Code(1000), "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"1000": 1,
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{err: t.err}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{err: t.err}
	}

	for _, tt := range tests {
		validate := func(mw *Middleware, direction string) {
			key, free := getKey(req, direction)
			edge := mw.graph.getEdge(key)
			free()
			assert.Equal(t, int64(tt.wantCalls), edge.calls.Load())
			assert.Equal(t, int64(tt.wantSuccesses), edge.successes.Load())
			for tagName, val := range tt.wantCallerFailures {
				assert.Equal(t, int64(val), edge.callerFailures.MustGet(_error, tagName).Load())
			}
			for tagName, val := range tt.wantServerFailures {
				assert.Equal(t, int64(val), edge.serverFailures.MustGet(_error, tagName).Load())
			}
		}
		t.Run(tt.desc+", unary inbound", func(t *testing.T) {
			mw := NewMiddleware(zap.NewNop(), metrics.New().Scope(), NewNopContextExtractor())
			mw.Handle(
				context.Background(),
				req,
				reqBuf,
				newHandler(tt),
			)
			validate(mw, string(_directionInbound))
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			mw := NewMiddleware(zap.NewNop(), metrics.New().Scope(), NewNopContextExtractor())
			mw.Call(context.Background(), req, reqBuf, newOutbound(tt))
			validate(mw, string(_directionOutbound))
		})
	}
}

// getKey gets the "key" that we will use to get an edge in the graph.  We use
// a separate function to recreate the logic because extracting it out in the
// main code could have performance implications.
func getKey(req *yarpc.Request, direction string) (key []byte, free func()) {
	d := internaldigester.New()
	d.Add(req.Caller)
	d.Add(req.Service)
	d.Add(req.Transport)
	d.Add(string(req.Encoding))
	d.Add(req.Procedure)
	d.Add(req.RoutingKey)
	d.Add(req.RoutingDelegate)
	d.Add(direction)
	return d.Digest(), d.Free
}

func TestUnaryInboundApplicationErrors(t *testing.T) {
	defer stubTime()()
	req := &yarpc.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
	}
	reqBuf := yarpc.NewBufferString("body")

	expectedFields := []zapcore.Field{
		zap.String("source", req.Caller),
		zap.String("dest", req.Service),
		zap.String("transport", "unknown"),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("routingKey", req.RoutingKey),
		zap.String("routingDelegate", req.RoutingDelegate),
		zap.String("direction", string(_directionInbound)),
		zap.String("rpcType", "Unary"),
		zap.Duration("latency", 0),
		zap.Bool("successful", false),
		zap.Skip(),
		zap.String("error", "hello"),
	}

	core, logs := observer.New(zap.DebugLevel)
	mw := NewMiddleware(zap.New(core), metrics.New().Scope(), NewNopContextExtractor())

	res, _, err := mw.Handle(
		context.Background(),
		req,
		reqBuf,
		fakeHandler{err: nil, applicationErr: yarpcerror.New(yarpcerror.CodeUnknown, "error", yarpcerror.WithName("hello"))},
	)

	require.NoError(t, err, "Unexpected transport error.")
	require.NotNil(t, res)
	require.NotNil(t, res.ApplicationErrorInfo)

	expected := observer.LoggedEntry{
		Entry: zapcore.Entry{
			Level:   zapcore.ErrorLevel,
			Message: "Error handling inbound request.: hello: error",
		},
		Context: expectedFields,
	}
	entries := logs.TakeAll()
	require.Equal(t, 1, len(entries), "Unexpected number of log entries written.")
	entry := entries[0]
	entry.Time = time.Time{}
	assert.Equal(t, expected, entry, "Unexpected log entry written.")
}

func TestMiddlewareSuccessSnapshot(t *testing.T) {
	defer stubTime()()
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(zap.NewNop(), meter, NewNopContextExtractor())

	res, _, err := mw.Handle(
		context.Background(),
		&yarpc.Request{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
		},
		yarpc.NewBufferString("body"),
		fakeHandler{nil, nil},
	)
	require.NoError(t, err, "Unexpected transport error.")
	require.NotNil(t, res)
	require.Nil(t, res.ApplicationErrorInfo)

	snap := root.Snapshot()
	tags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"transport":        "unknown",
		"encoding":         "raw",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"source":           "caller",
	}
	want := &metrics.RootSnapshot{
		Counters: []metrics.Snapshot{
			{Name: "calls", Tags: tags, Value: 1},
			{Name: "successes", Tags: tags, Value: 1},
		},
		Histograms: []metrics.HistogramSnapshot{
			{
				Name: "caller_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name: "server_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name:   "success_latency_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{1},
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}

func TestMiddlewareFailureSnapshot(t *testing.T) {
	defer stubTime()()
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(zap.NewNop(), meter, NewNopContextExtractor())

	res, _, err := mw.Handle(
		context.Background(),
		&yarpc.Request{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
		},
		yarpc.NewBufferString("body"),
		fakeHandler{fmt.Errorf("yuno"), nil},
	)
	require.Error(t, err, "Expected transport error.")
	require.Nil(t, res)

	snap := root.Snapshot()
	tags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"transport":        "unknown",
		"encoding":         "raw",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"source":           "caller",
	}
	errorTags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"transport":        "unknown",
		"encoding":         "raw",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"source":           "caller",
		"error":            "unknown_internal_yarpc",
	}
	want := &metrics.RootSnapshot{
		Counters: []metrics.Snapshot{
			{Name: "calls", Tags: tags, Value: 1},
			{Name: "server_failures", Tags: errorTags, Value: 1},
			{Name: "successes", Tags: tags, Value: 0},
		},
		Histograms: []metrics.HistogramSnapshot{
			{
				Name: "caller_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name:   "server_failure_latency_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{1},
			},
			{
				Name: "success_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}
