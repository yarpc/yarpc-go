// Copyright (c) 2021 Uber Technologies, Inc.
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

func TestFromHTTP2ConnectRequest(t *testing.T) {
	tests := []struct {
		desc      string
		treq      *transport.Request
		wantError string
	}{
		{
			desc: "malformed CONNECT request: :scheme header set",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":scheme": "http2"}),
			},
			wantError: `HTTP2 CONNECT request must not contain pseudo header ":scheme"`,
		},
		{
			desc: "malformed CONNECT request: :path header set",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":path": "foo/path"}),
			},
			wantError: `HTTP2 CONNECT request must not contain pseudo header ":path"`,
		},
		{
			desc:      "malformed CONNECT request: :authority header missing",
			treq:      &transport.Request{},
			wantError: `HTTP2 CONNECT request must contain pseudo header ":authority"`,
		},
		{
			desc: "malformed CONNECT request: :authority header missing",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":authority": "127.0.0.1:1234"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			req, err := fromHTTP2ConnectRequest(tt.treq)
			if tt.wantError != "" {
				assert.EqualError(t, err, tt.wantError)
				return
			}
			assert.Equal(t, http.MethodConnect, req.Method)
		})
	}
}
