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
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	tests := []struct {
		req         *transport.Request
		ttl         string // set to try parseTTL
		wantErr     error
		wantMessage string
	}{
		{
			// No error
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				TTL:       time.Second,
				Procedure: "hello",
			},
		},
		{
			// encoding is not required
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				TTL:       time.Second,
				Procedure: "hello",
			},
		},
		{
			req: &transport.Request{
				Service:   "service",
				TTL:       time.Second,
				Procedure: "hello",
			},
			wantErr: transport.MissingParametersError{
				Parameters: []string{"caller name"},
			},
			wantMessage: "missing caller name",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				TTL:       time.Second,
				Procedure: "hello",
			},
			wantErr: transport.MissingParametersError{
				Parameters: []string{"service name"},
			},
			wantMessage: "missing service name",
		},
		{
			req: &transport.Request{
				Caller:  "caller",
				Service: "service",
				TTL:     time.Second,
			},
			wantErr: transport.MissingParametersError{
				Parameters: []string{"procedure"},
			},
			wantMessage: "missing procedure",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
			},
			wantErr: transport.MissingParametersError{
				Parameters: []string{"TTL"},
			},
			wantMessage: "missing TTL",
		},
		{
			req: &transport.Request{},
			wantErr: transport.MissingParametersError{
				Parameters: []string{
					"service name", "procedure", "caller name", "TTL",
				},
			},
			wantMessage: "missing service name, procedure, caller name, and TTL",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				TTL:       -1 * time.Second,
				Procedure: "hello",
			},
			wantErr: transport.InvalidTTLError{
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
			ttl: "-1000",
			wantErr: transport.InvalidTTLError{
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
			ttl: "not an integer",
			wantErr: transport.InvalidTTLError{
				Service:   "service",
				Procedure: "hello",
				TTL:       "not an integer",
			},
			wantMessage: `invalid TTL "not an integer" for procedure "hello" of service "service": must be positive integer`,
		},
	}

	for _, tt := range tests {
		v := Validator{Request: tt.req}
		if tt.ttl != "" {
			v.ParseTTL(tt.ttl)
		}

		_, err := v.Validate()
		if tt.wantErr != nil {
			assert.Equal(t, tt.wantErr, err)
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, err.Error())
			}
		} else {
			assert.NoError(t, err)
		}
	}
}
