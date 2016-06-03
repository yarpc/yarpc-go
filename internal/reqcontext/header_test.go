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

package reqcontext

import (
	"fmt"
	"testing"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// Actions must have only one attribute set
type action struct {
	add    *transport.Headers      // ctx = AddHeaders(ctx, add)
	get    *transport.Headers      // GetHeaders(ctx) == get
	modify func(transport.Headers) // modify(GetHeaders(ctx))
}

func (a action) String() string {
	switch {
	case a.add != nil:
		return fmt.Sprintf("Add(%v)", *a.add)
	case a.get != nil:
		return fmt.Sprintf("Get(%v)", *a.get)
	case a.modify != nil:
		return fmt.Sprintf("Modify(%v)", a.modify)
	default:
		return fmt.Sprintf("Noop")
	}
}

func TestHeaders(t *testing.T) {
	var nilHeaders transport.Headers

	// Each test is a series of actions.
	tests := [][]action{
		{
			{get: &nilHeaders},
			{add: &transport.Headers{"foo": "bar"}},
			{get: &transport.Headers{"foo": "bar"}},
		},
		{
			{get: &nilHeaders},
			{add: &transport.Headers{"foo": "bar"}},
			{add: &transport.Headers{"baz": "qux"}},
			{get: &transport.Headers{"foo": "bar", "baz": "qux"}},
		},
		{
			{get: &nilHeaders},
			{add: &transport.Headers{"foo": "bar"}},
			{get: &transport.Headers{"foo": "bar"}},
			{add: &transport.Headers{"foo": ""}},
			{get: &transport.Headers{"foo": ""}},
		},
		{
			{get: &nilHeaders},
			{add: &transport.Headers{}},
			{get: &transport.Headers{}},
		},
		{
			{get: &nilHeaders},
			{add: &transport.Headers{"foo": "bar"}},
			{modify: func(h transport.Headers) {
				h.Set("foo", "baz")
			}},
			{get: &transport.Headers{"foo": "baz"}},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		var ops []action
		for _, action := range tt {
			ops = append(ops, action)
			switch {
			case action.add != nil:
				ctx = AddHeaders(ctx, *action.add)
			case action.get != nil:
				assert.Equal(
					t, *action.get, GetHeaders(ctx), "failed on: %v", ops)
			case action.modify != nil:
				action.modify(GetHeaders(ctx))
			}
		}
	}
}
