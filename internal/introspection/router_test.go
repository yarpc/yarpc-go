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

package introspection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcedureName(t *testing.T) {
	tests := []struct {
		msg  string
		p    Procedure
		want string
	}{
		{
			msg:  "proto encoding",
			p:    Procedure{Encoding: "proto", Name: "full.path.to.service::method"},
			want: "/full.path.to.service/method",
		},
		{
			msg:  "json encoding",
			p:    Procedure{Encoding: "json", Name: "full.path.to.service::method"},
			want: "full.path.to.service::method",
		},
		{
			msg:  "thrift encoding",
			p:    Procedure{Encoding: "thrift", Name: "full.path.to.service::method"},
			want: "full.path.to.service::method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.p.ProcedureName(), "procedure name mismatch")
		})
	}
}
