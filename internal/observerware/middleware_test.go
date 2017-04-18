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

package observerware

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestUnaryInboundMiddleware(t *testing.T) {
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

	tests := []struct {
		desc       string
		err        error // downstream error
		wantFields []zapcore.Field
	}{
		{
			desc: "no downstream errors",
			wantFields: []zapcore.Field{
				zap.Object("request", req),
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(),
				zap.Skip(),
			},
		},
		{
			desc: "downstream errors",
			err:  failed,
			wantFields: []zapcore.Field{
				zap.Object("request", req),
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Error(failed),
				zap.Skip(),
			},
		},
	}

	for _, tt := range tests {
		core, logs := observer.New(zapcore.DebugLevel)
		mw := New(zap.New(core), NewNopContextExtractor())

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
			err := mw.Handle(context.Background(), req, nil /* response writer */, fakeHandler{tt.err})
			checkErr(err)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Handled inbound request.",
				},
				Context: append([]zapcore.Field{zap.String("rpcType", "unary")}, tt.wantFields...),
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			res, err := mw.Call(context.Background(), req, fakeOutbound{err: tt.err})
			checkErr(err)
			if tt.err == nil {
				assert.NotNil(t, res, "Expected non-nil response if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Made outbound call.",
				},
				Context: append([]zapcore.Field{zap.String("rpcType", "unary")}, tt.wantFields...),
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", oneway inbound", func(t *testing.T) {
			err := mw.HandleOneway(context.Background(), req, fakeHandler{tt.err})
			checkErr(err)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Handled inbound request.",
				},
				Context: append([]zapcore.Field{zap.String("rpcType", "oneway")}, tt.wantFields...),
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", oneway outbound", func(t *testing.T) {
			ack, err := mw.CallOneway(context.Background(), req, fakeOutbound{err: tt.err})
			checkErr(err)
			if tt.err == nil {
				assert.NotNil(t, ack, "Expected non-nil ack if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Made outbound call.",
				},
				Context: append([]zapcore.Field{zap.String("rpcType", "oneway")}, tt.wantFields...),
			}
			assert.Equal(t, expected, getLog(), "Unexpected log entry written.")
		})
	}
}
