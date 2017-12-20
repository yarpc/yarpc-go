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

package transport_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap/zapcore"
)

func TestValidator(t *testing.T) {
	tests := []struct {
		req           *transport.Request
		transportType transport.Type
		ttl           time.Duration

		wantMissingParams []string
	}{
		{
			// No error
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				Procedure: "hello",
			},
			ttl: time.Second,
		},
		{
			// encoding is not required
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
			},
			wantMissingParams: []string{"encoding"},
		},
		{
			req: &transport.Request{
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantMissingParams: []string{"caller"},
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantMissingParams: []string{"service"},
		},
		{
			req: &transport.Request{
				Caller:   "caller",
				Service:  "service",
				Encoding: "raw",
			},
			wantMissingParams: []string{"procedure"},
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			transportType:     transport.Unary,
			wantMissingParams: []string{"TTL"},
		},
		{
			req:               &transport.Request{},
			wantMissingParams: []string{"encoding", "caller", "service", "procedure"},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		err := transport.ValidateRequest(tt.req)

		if err == nil && tt.transportType == transport.Unary {
			var cancel func()

			if tt.ttl != 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.ttl)
				defer cancel()
			}

			err = transport.ValidateRequestContext(ctx)
		}

		if len(tt.wantMissingParams) > 0 {
			if assert.Error(t, err) {
				for _, wantMissingParam := range tt.wantMissingParams {
					assert.Contains(t, err.Error(), wantMissingParam)
				}
			}
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestRequestLogMarshaling(t *testing.T) {
	r := &transport.Request{
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
	enc := zapcore.NewMapObjectEncoder()
	assert.NoError(t, r.MarshalLogObject(enc), "Unexpected error marshaling request.")
	assert.Equal(t, map[string]interface{}{
		"caller":          "caller",
		"service":         "service",
		"encoding":        "raw",
		"procedure":       "procedure",
		"shardKey":        "shard01",
		"routingKey":      "routing-key",
		"routingDelegate": "routing-delegate",
	}, enc.Fields, "Unexpected output after marshaling request.")
}

func TestRequestMetaToRequestConversionAndBack(t *testing.T) {
	reqMeta := &transport.RequestMeta{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "raw",
		Procedure:       "hello",
		Headers:         transport.NewHeaders().With("key", "val"),
		ShardKey:        "shard",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
	}

	req := reqMeta.ToRequest()

	assert.Equal(t, reqMeta, req.ToRequestMeta())
}

func TestNilRequestMetaToRequestConversion(t *testing.T) {
	var reqMeta *transport.RequestMeta

	assert.Equal(t, &transport.Request{}, reqMeta.ToRequest())
}
