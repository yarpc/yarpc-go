// Copyright (c) 2022 Uber Technologies, Inc.
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
		prefix        string
		toTransport   transport.Headers
		fromTransport transport.Headers
		http          http.Header
	}{
		{
			ApplicationHeaderPrefix,
			transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "hello",
			}),
			transport.HeadersFromMap(map[string]string{
				"Foo":     "bar",
				"Foo-Bar": "hello",
			}),
			http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"hello"},
			},
		},
		{
			ApplicationHeaderPrefix,
			transport.HeadersFromMap(map[string]string{
				"Rpc-Header-Foo":     "bar",
				"Rpc-Header-Foo-Bar": "hello",
			}),
			transport.HeadersFromMap(map[string]string{
				"Foo":     "bar",
				"Foo-Bar": "hello",
			}),
			http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"hello"},
			},
		},
	}

	for _, tt := range tests {
		m := headerMapper{tt.prefix}
		assert.Equal(t, tt.fromTransport, m.FromHTTPHeaders(tt.http, transport.Headers{}))
		assert.Equal(t, tt.http, m.ToHTTPHeaders(tt.toTransport, nil))

		m = headerMapper{tt.prefix}
		var res http.Header
		for k, v := range tt.toTransport.OriginalItems() {
			res = m.ToHTTPHeader(res, k, v)
		}
		assert.Equal(t, tt.http, res)
	}
}

// TODO(abg): Test handling of duplicate HTTP headers when
// https://github.com/yarpc/yarpc/issues/21 is resolved.
