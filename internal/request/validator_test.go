// Copyright (c) 2016 Uber Technologies, Inc.
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

package request

import (
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	tests := []struct {
		req           *transport.Request
		transportType transport.Type

		ttl       time.Duration
		ttlString string // set to try parseTTL

		wantErr     error
		wantMessage string
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
			wantErr: missingParametersError{
				Parameters: []string{"encoding"},
			},
			wantMessage: "missing encoding",
		},
		{
			req: &transport.Request{
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantErr: missingParametersError{
				Parameters: []string{"caller name"},
			},
			wantMessage: "missing caller name",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantErr: missingParametersError{
				Parameters: []string{"service name"},
			},
			wantMessage: "missing service name",
		},
		{
			req: &transport.Request{
				Caller:   "caller",
				Service:  "service",
				Encoding: "raw",
			},
			wantErr: missingParametersError{
				Parameters: []string{"procedure"},
			},
			wantMessage: "missing procedure",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			transportType: transport.Unary,
			wantErr: missingParametersError{
				Parameters: []string{"TTL"},
			},
			wantMessage: "missing TTL",
		},
		{
			req: &transport.Request{},
			wantErr: missingParametersError{
				Parameters: []string{
					"service name", "procedure", "caller name", "encoding",
				},
			},
			wantMessage: "missing service name, procedure, caller name, and encoding",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				Procedure: "hello",
			},
			transportType: transport.Unary,
			ttlString:     "-1000",
			wantErr: invalidTTLError{
				Service:   "service",
				Procedure: "hello",
				TTL:       "-1000",
			},
			wantMessage: `invalid TTL "-1000" for procedure "hello" of service "service": must be positive integer`,
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				Procedure: "hello",
			},
			transportType: transport.Unary,
			ttlString:     "not an integer",
			wantErr: invalidTTLError{
				Service:   "service",
				Procedure: "hello",
				TTL:       "not an integer",
			},
			wantMessage: `invalid TTL "not an integer" for procedure "hello" of service "service": must be positive integer`,
		},
	}

	for _, tt := range tests {
		v := Validator{Request: tt.req}

		ctx := context.Background()
		_, err := v.Validate(ctx)

		if err == nil && tt.transportType == transport.Oneway {
			_, err = v.ValidateOneway(ctx)
		} else if err == nil { // default to unary
			var cancel func()

			if tt.ttl != 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.ttl)
				defer cancel()
			}

			if tt.ttlString != "" {
				ctx, cancel = v.ParseTTL(ctx, tt.ttlString)
				defer cancel()
			}

			_, err = v.ValidateUnary(ctx)
		}

		if tt.wantErr != nil {
			assert.Equal(t, tt.wantErr, err)
			if tt.wantMessage != "" && err != nil {
				assert.Equal(t, tt.wantMessage, err.Error())
			}
		} else {
			assert.NoError(t, err)
		}
	}
}
