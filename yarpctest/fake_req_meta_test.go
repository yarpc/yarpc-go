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

package yarpctest

import (
	"testing"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
)

func TestReqMeta(t *testing.T) {
	tests := []struct {
		build func(*ReqMetaBuilder) *ReqMetaBuilder

		wantEncoding  transport.Encoding
		wantHeaders   yarpc.Headers
		wantCaller    string
		wantProcedure string
		wantService   string
	}{
		{
			build: func(r *ReqMetaBuilder) *ReqMetaBuilder {
				return r
			},
			wantEncoding:  "",
			wantHeaders:   yarpc.NewHeaders(),
			wantCaller:    "",
			wantProcedure: "",
			wantService:   "",
		},
		{
			build: func(r *ReqMetaBuilder) *ReqMetaBuilder {
				return r.
					Encoding(transport.Encoding("myencoding")).
					Headers(yarpc.NewHeaders().With("foo", "bar")).
					Caller("caller").
					Service("service").
					Procedure("procedure")
			},
			wantEncoding:  "myencoding",
			wantHeaders:   yarpc.NewHeaders().With("foo", "bar"),
			wantCaller:    "caller",
			wantService:   "service",
			wantProcedure: "procedure",
		},
	}

	for _, tt := range tests {
		reqMeta := tt.build(NewReqMetaBuilder()).Build()
		assert.Equal(t, tt.wantEncoding, reqMeta.Encoding())
		assert.Equal(t, tt.wantHeaders, reqMeta.Headers())
		assert.Equal(t, tt.wantCaller, reqMeta.Caller())
		assert.Equal(t, tt.wantProcedure, reqMeta.Procedure())
		assert.Equal(t, tt.wantService, reqMeta.Service())
	}
}
