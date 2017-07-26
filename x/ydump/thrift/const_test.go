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

package thrift

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/compile"
)

func TestConstToRequest(t *testing.T) {
	tests := []struct {
		v    compile.ConstantValue
		want interface{}
	}{
		{
			v:    compile.ConstantBool(true),
			want: true,
		},
		{
			v:    compile.ConstantBool(false),
			want: false,
		},
		{
			v:    compile.ConstantDouble(1.05),
			want: 1.05,
		},
		{
			v:    compile.ConstantInt(1),
			want: int64(1),
		},
		{
			v:    compile.ConstantString("foo"),
			want: "foo",
		},
		{
			v: compile.ConstReference{
				Target: &compile.Constant{
					Name:  "a",
					Type:  &compile.StringSpec{},
					Value: compile.ConstantString("foo"),
				},
			},
			want: "foo",
		},
		{
			v: compile.EnumItemReference{
				Item: &compile.EnumItem{
					Name:  "a",
					Value: 5,
				},
			},
			want: int32(5),
		},
		{
			v: compile.ConstantList{
				compile.ConstantInt(1),
				compile.ConstantBool(true),
				compile.ConstantString("foo"),
			},
			want: []interface{}{int64(1), true, "foo"},
		},
		{
			v: compile.ConstantSet{
				compile.ConstantString("foo"),
				compile.ConstantSet{compile.ConstantString("bar")},
			},
			want: []interface{}{"foo", []interface{}{"bar"}},
		},
		{
			v: compile.ConstantMap{
				{Key: compile.ConstantInt(1), Value: compile.ConstantString("v1")},
				{Key: compile.ConstantBool(true), Value: compile.ConstantString("v2")},
			},
			want: map[interface{}]interface{}{
				int64(1): "v1",
				true:     "v2",
			},
		},
	}

	for _, tt := range tests {
		got := constToRequest(tt.v)
		assert.Equal(t, tt.want, got, "Result mismatch for %v", tt.v)
	}
}
