// Copyright (c) 2025 Uber Technologies, Inc.
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

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

func TestHTTPHeaders(t *testing.T) {
	tests := []struct {
		name         string
		transHeaders transport.Headers
		expectedHTTP http.Header
	}{
		{
			name: "application only",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "hello",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"hello"},
			},
		},
		{
			name: "tracing only",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"uber-trace-id": "tid",
				"uberctx-foo":   "ctxval",
			}),
			expectedHTTP: http.Header{
				"Uber-Trace-Id": []string{"tid"},
				"Uberctx-Foo":   []string{"ctxval"},
			},
		},
		{
			name: "mixed headers",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo":           "bar",
				"uber-trace-id": "tid",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
				"Uber-Trace-Id":  []string{"tid"},
			},
		},
	}

	for _, tt := range tests {
		hm := headerMapper{ApplicationHeaderPrefix}
		gotHTTP := hm.ToHTTPHeaders(tt.transHeaders, nil)
		assert.Equal(t, tt.expectedHTTP, gotHTTP, "%s: ToHTTPHeaders", tt.name)
		gotTrans := hm.FromHTTPHeaders(gotHTTP, transport.Headers{})
		assert.Equal(t, tt.transHeaders.Items(), gotTrans.Items(), "%s: FromHTTPHeaders", tt.name)
	}
}

// TODO(abg): Test handling of duplicate HTTP headers when
// https://github.com/yarpc/yarpc/issues/21 is resolved.
