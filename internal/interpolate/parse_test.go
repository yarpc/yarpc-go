// Copyright (c) 2020 Uber Technologies, Inc.
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

package interpolate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSuccess(t *testing.T) {
	tests := []struct {
		give string
		want String
	}{
		{
			give: "foo",
			want: String{literal("foo")},
		},
		{
			give: "foo ${bar} baz",
			want: String{
				literal("foo "),
				variable{Name: "bar"},
				literal(" baz"),
			},
		},
		{
			give: "foo $ {bar} baz",
			want: String{literal("foo "), literal("$ "), literal("{bar} baz")},
		},
		{
			give: "foo ${bar:}",
			want: String{
				literal("foo "),
				variable{Name: "bar", HasDefault: true},
			},
		},
		{
			give: "${foo:bar}",
			want: String{
				variable{
					Name:       "foo",
					Default:    "bar",
					HasDefault: true,
				},
			},
		},
		{
			give: `foo \${bar:42} baz`,
			want: String{
				literal("foo "),
				literal("$"),
				literal("{bar:42} baz"),
			},
		},
		{
			give: "$foo${bar}",
			want: String{
				literal("$f"),
				literal("oo"),
				variable{Name: "bar"},
			},
		},
		{
			give: "foo${b-a-r}",
			want: String{
				literal("foo"),
				variable{Name: "b-a-r"},
			},
		},
		{
			give: "foo ${bar::baz} qux",
			want: String{
				literal("foo "),
				variable{Name: "bar", HasDefault: true, Default: ":baz"},
				literal(" qux"),
			},
		},
		{
			give: "a ${b:hello world} c",
			want: String{
				literal("a "),
				variable{Name: "b", HasDefault: true, Default: "hello world"},
				literal(" c"),
			},
		},
		{
			give: "foo $${bar}",
			want: String{literal("foo "), literal("$$"), literal("{bar}")},
		},
	}

	for _, tt := range tests {
		out, err := Parse(tt.give)
		assert.NoError(t, err)
		assert.Equal(t, tt.want, out)
	}
}

func TestParseFailures(t *testing.T) {
	tests := []string{
		"${foo",
		"${foo.}",
		"${foo-}",
		"${foo--bar}",
	}

	for _, tt := range tests {
		_, err := Parse(tt)
		assert.Error(t, err, tt)
	}
}
