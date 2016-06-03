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

	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestContextHeaders(t *testing.T) {
	tests := []struct {
		ctx  context.Context
		want transport.Headers
	}{
		{
			context.Background(),
			transport.Headers{},
		},
		{
			WithHeaders(context.Background(), transport.Headers{"foo": "bar"}),
			transport.Headers{"foo": "bar"},
		},
		{
			WithHeaders(
				WithHeaders(
					context.Background(),
					transport.Headers{"foo": "bar"},
				),
				transport.Headers{"baz": "qux", "xy": "z"},
			),
			transport.Headers{"foo": "bar", "baz": "qux", "xy": "z"},
		},
		{
			WithHeaders(
				WithHeaders(
					context.Background(),
					transport.Headers{"foo": "bar"},
				),
				transport.Headers{"foo": "qux"},
			),
			transport.Headers{"foo": "qux"},
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, HeadersFromContext(tt.ctx))
	}
}

func TestContextHeadersAreCopies(t *testing.T) {
	ctx := WithHeaders(context.Background(), transport.Headers{"foo": "bar"})
	assert.Equal(t, transport.Headers{"foo": "bar"}, HeadersFromContext(ctx))

	HeadersFromContext(ctx)["foo"] = "not-bar"
	assert.Equal(t, transport.Headers{"foo": "bar"}, HeadersFromContext(ctx))
}

func TestContextHeadersOriginalMapIsCopied(t *testing.T) {
	hs := transport.Headers{"foo": "bar"}
	ctx := WithHeaders(context.Background(), hs)
	assert.Equal(t, transport.Headers{"foo": "bar"}, HeadersFromContext(ctx))

	hs["foo"] = "not-bar"
	assert.Equal(t, transport.Headers{"foo": "bar"}, HeadersFromContext(ctx))
}
