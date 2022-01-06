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

package protobuf

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestCastError(t *testing.T) {
	assert.Equal(t, yarpcerrors.CodeInternal, yarpcerrors.FromError(CastError(nil, nil)).Code())
}

func TestClientBuilderOptions(t *testing.T) {
	assert.Nil(t, ClientBuilderOptions(nil, reflect.StructField{Tag: `service:"keyvalue"`}))
	assert.Equal(t, []ClientOption{UseJSON}, ClientBuilderOptions(nil, reflect.StructField{Tag: `service:"keyvalue" proto:"json"`}))
}

func TestUniqueLowercaseStrings(t *testing.T) {
	tests := []struct {
		give []string
		want []string
	}{
		{
			give: []string{"foo", "bar", "baz"},
			want: []string{"foo", "bar", "baz"},
		},
		{
			give: []string{"foo", "BAR", "bAz"},
			want: []string{"foo", "bar", "baz"},
		},
		{
			give: []string{"foo", "BAR", "bAz", "bar"},
			want: []string{"foo", "bar", "baz"},
		},
	}
	for _, tt := range tests {
		got := uniqueLowercaseStrings(tt.give)
		sort.Strings(tt.want)
		sort.Strings(got)
		assert.Equal(t, tt.want, got)
	}
}
