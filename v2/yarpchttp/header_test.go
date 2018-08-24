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

package yarpchttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/v2/yarpctransport"
)

func TestHTTPHeaders(t *testing.T) {
	tests := []struct {
		prefix        string
		toTransport   yarpctransport.Headers
		fromTransport yarpctransport.Headers
		http          http.Header
	}{
		{
			ApplicationHeaderPrefix,
			yarpctransport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "hello",
			}),
			yarpctransport.HeadersFromMap(map[string]string{
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
		assert.Equal(t, tt.fromTransport, m.FromHTTPHeaders(tt.http, yarpctransport.Headers{}))
		assert.Equal(t, tt.http, m.ToHTTPHeaders(tt.toTransport, nil))
	}
}

// TODO(abg): Test handling of duplicate HTTP headers when
// https://github.com/yarpc/yarpc/issues/21 is resolved.
