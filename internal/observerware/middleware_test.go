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
		desc    string
		handler transport.UnaryHandler
		extract ContextExtractor

		wantErr    bool
		wantFields []zapcore.Field
	}{
		{
			desc:    "downstream errors",
			handler: fakeHandler{},
			extract: NewNopContextExtractor(),
			wantFields: []zapcore.Field{
				zap.String("rpcType", "unary inbound"),
				zap.Skip(),
				zap.Object("request", req),
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(),
			},
		},
		{
			desc:    "no downstream errors",
			extract: NewNopContextExtractor(),
			handler: fakeHandler{failed},
			wantErr: true,
			wantFields: []zapcore.Field{
				zap.String("rpcType", "unary inbound"),
				zap.Skip(),
				zap.Object("request", req),
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Error(failed),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			mw := NewUnaryInbound(zap.New(core), tt.extract)
			err := mw.Handle(context.Background(), req, nil /* response writer */, tt.handler)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error from middleware.")
			} else {
				assert.NoError(t, err, "Unexpected error from middleware.")
			}
			require.Equal(t, 1, logs.Len(), "Unexpected number of logs written.")
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   zapcore.DebugLevel,
					Message: "Handled request.",
				},
				Context: tt.wantFields,
			}
			assert.Equal(t, expected, logs.AllUntimed()[0], "Unexpected log entry written.")
		})
	}
}
