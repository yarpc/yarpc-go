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

package yarpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thriftrw/thriftrw-go/ptr"
	"golang.org/x/net/context"
)

func TestContextHeaders(t *testing.T) {
	tests := []struct {
		ctx  context.Context
		want map[string]*string
	}{
		{
			context.Background(),
			map[string]*string{"foo": nil},
		},
		{
			WithBaggage(context.Background(), "foo", "bar"),
			map[string]*string{"foo": ptr.String("bar"), "baz": nil},
		},
		{
			WithBaggage(WithBaggage(context.Background(), "foo", "bar"), "baz", "qux"),
			map[string]*string{"foo": ptr.String("bar"), "baz": ptr.String("qux")},
		},
		{
			WithBaggage(WithBaggage(context.Background(), "foo", "bar"), "foo", "baz"),
			map[string]*string{"foo": ptr.String("baz"), "bar": nil},
		},
	}

	for _, tt := range tests {
		for k, v := range tt.want {
			got, ok := BaggageFromContext(tt.ctx, k)
			if v != nil {
				if assert.True(t, ok, "expected to find %v in baggage", k) {
					assert.Equal(t, *v, got, "value fo %v did not match", k)
				}
			} else {
				assert.False(t, ok, "did not expect to find %v in baggage", k)
			}
		}
	}
}
