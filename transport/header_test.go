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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaders(t *testing.T) {
	tests := []struct {
		headers  map[string]string
		matches  map[string]string
		failures map[string]struct{}
	}{
		{
			nil,
			map[string]string{},
			map[string]struct{}{"foo": struct{}{}},
		},
		{
			map[string]string{
				"Foo": "Bar",
				"Baz": "qux",
			},
			map[string]string{
				"foo": "Bar",
				"Foo": "Bar",
				"FOO": "Bar",
				"baz": "qux",
				"Baz": "qux",
				"BaZ": "qux",
			},
			map[string]struct{}{"bar": struct{}{}},
		},
		{
			map[string]string{
				"foo": "bar",
				"baz": "",
			},
			map[string]string{
				"foo": "bar",
			},
			map[string]struct{}{
				"baz": struct{}{},
				"qux": struct{}{},
			},
		},
	}

	for _, tt := range tests {
		headers := NewHeaders(tt.headers)
		for k, v := range tt.matches {
			assert.Equal(t, v, headers.Get(k))
		}
		for k := range tt.failures {
			assert.Equal(t, "", headers.Get(k))
		}
	}
}
