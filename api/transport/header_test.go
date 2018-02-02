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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaders(t *testing.T) {
	tests := []struct {
		headers  map[string]string
		matches  map[string]string
		failures []string
	}{
		{
			nil,
			map[string]string{},
			[]string{"foo"},
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
			[]string{"bar"},
		},
		{
			map[string]string{
				"foo": "bar",
				"baz": "",
			},
			map[string]string{
				"foo": "bar",
				"baz": "",
				"Baz": "",
			},
			[]string{"qux"},
		},
	}

	for _, tt := range tests {
		headers := HeadersFromMap(tt.headers)
		for k, v := range tt.matches {
			vg, ok := headers.Get(k)
			assert.True(t, ok, "expected true for %q", k)
			assert.Equal(t, v, vg, "value mismatch for %q", k)
		}
		for _, k := range tt.failures {
			v, ok := headers.Get(k)
			assert.False(t, ok, "expected false for %q", k)
			assert.Equal(t, "", v, "expected empty string for %q", k)
		}
	}
}

func TestItemsAndRawItems(t *testing.T) {
	const (
		headerKey          = "foo-BAR-BaZ"
		headerKeyLowercase = "foo-bar-baz"
		headerVal          = "FooBarBaz"
	)
	header := NewHeaders().With(headerKey, headerVal)
	assert.Equal(t, map[string]string{headerKey: headerVal}, header.RawItems())
	assert.Equal(t, map[string]string{headerKeyLowercase: headerVal}, header.Items())
}
