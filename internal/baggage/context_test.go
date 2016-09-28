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

package baggage

import (
	"fmt"
	"testing"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"context"
)

type pair struct{ Key, Value string }

func (p pair) String() string {
	return fmt.Sprintf("%q=%q", p.Key, p.Value)
}

type headers map[string]string

// Actions must have only one attribute set
type action struct {
	add  *pair    // ctx = NewContext(ctx, key, value)
	many *headers // NewContextWithHeaders(ctx, many)
	get  *pair    // value == Get(ctx, key)
	from *headers // FromContext(ctx) == from
}

func (a action) String() string {
	switch {
	case a.add != nil:
		return fmt.Sprintf("Add(%v)", *a.add)
	case a.many != nil:
		return fmt.Sprintf("Many(%v)", *a.many)
	case a.get != nil:
		return fmt.Sprintf("Get(%v)", *a.get)
	case a.from != nil:
		return fmt.Sprintf("From(%v)", *a.from)
	default:
		return fmt.Sprintf("Noop")
	}
}

func TestHeaders(t *testing.T) {
	var nilHeaders headers

	// Each test is a series of actions.
	tests := [][]action{
		{
			{from: &nilHeaders},
			{add: &pair{"foo", "bar"}},
			{from: &headers{"foo": "bar"}},
			{get: &pair{"foo", "bar"}},
		},
		{
			{from: &nilHeaders},
			{many: &headers{"foo": "bar", "baz": "qux"}},
			{from: &headers{"foo": "bar", "baz": "qux"}},
			{get: &pair{"foo", "bar"}},
			{get: &pair{"baz", "qux"}},
		},
		{
			{from: &nilHeaders},
			{add: &pair{"foo", "bar"}},
			{add: &pair{"baz", "qux"}},
			{from: &headers{"foo": "bar", "baz": "qux"}},
			{get: &pair{"foo", "bar"}},
			{get: &pair{"baz", "qux"}},
		},
		{
			{from: &nilHeaders},
			{add: &pair{"foo", "bar"}},
			{from: &headers{"foo": "bar"}},
			{add: &pair{"foo", ""}},
			{from: &headers{"foo": ""}},
			{get: &pair{"foo", ""}},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		var ops []action
		for _, action := range tt {
			ops = append(ops, action)
			switch {
			case action.add != nil:
				ctx = NewContext(ctx, action.add.Key, action.add.Value)
			case action.many != nil:
				ctx = NewContextWithHeaders(ctx, *action.many)
			case action.get != nil:
				value, ok := Get(ctx, action.get.Key)
				if assert.True(t, ok, "expected success: %v", ops) {
					assert.Equal(t, action.get.Value, value, "failed on: %v", ops)
				}
			case action.from != nil:
				from := transport.HeadersFromMap(*action.from)
				assert.Equal(t, from, FromContext(ctx), "failed on: %v", ops)
			}
		}
	}
}

func TestContextIsNotModified(t *testing.T) {
	var emptyHeaders transport.Headers

	root := context.Background()
	assert.Equal(t, emptyHeaders, FromContext(root),
		"empty context must have no headers")

	ctx1 := NewContext(root, "foo", "bar")
	assert.Equal(t, emptyHeaders, FromContext(root),
		"empty context must have no headers")
	assert.Equal(t, transport.NewHeaders().With("foo", "bar"), FromContext(ctx1))

	ctx2 := NewContext(ctx1, "baz", "qux")
	assert.Equal(t, emptyHeaders, FromContext(root),
		"empty context must have no headers")
	assert.Equal(t, transport.NewHeaders().With("foo", "bar"), FromContext(ctx1))
	assert.Equal(t,
		transport.NewHeaders().
			With("foo", "bar").
			With("baz", "qux"), FromContext(ctx2))

	ctx3 := NewContextWithHeaders(ctx2,
		map[string]string{"hello": "world", "foo": "not-bar"})
	assert.Equal(t, emptyHeaders, FromContext(root),
		"empty context must have no headers")
	assert.Equal(t, transport.NewHeaders().With("foo", "bar"), FromContext(ctx1))
	assert.Equal(t, transport.NewHeaders().With("foo", "bar").With("baz", "qux"), FromContext(ctx2))
	assert.Equal(t,
		transport.NewHeaders().
			With("foo", "not-bar").
			With("baz", "qux").
			With("hello", "world"), FromContext(ctx3))
}
