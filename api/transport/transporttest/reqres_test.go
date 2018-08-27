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

package transporttest

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

func TestRequestMatcherTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		wantMatch bool
	}{
		{
			name:      "unset",
			transport: "",
			wantMatch: true,
		},
		{
			name:      "set",
			transport: "foo-transport",
			wantMatch: true,
		},
		{
			name:      "no match",
			transport: "bar-transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqMatcher := NewRequestMatcher(t, &transport.Request{
				Transport: tt.transport,
				Body:      &bytes.Buffer{},
			})

			req := &transport.Request{
				Transport: "foo-transport",
				Body:      &bytes.Buffer{},
			}

			if tt.wantMatch {
				assert.True(t, reqMatcher.Matches(req), "unexpected match")
			} else {
				assert.False(t, reqMatcher.Matches(req), "expected match")
			}
		})
	}
}
