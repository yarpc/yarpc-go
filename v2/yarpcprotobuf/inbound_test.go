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

package yarpcprotobuf

import (
	"context"
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
)

func TestResponseHeaders(t *testing.T) {
	h := NewUnaryHandler(UnaryHandlerParams{
		Handle: func(ctx context.Context, _ proto.Message) (proto.Message, error) {
			err := yarpc.CallFromContext(ctx).WriteResponseHeader("foo-key", "bar-val")
			require.NoError(t, err)

			// currently, we still expect to get response headers when a handler returns an error
			return nil, errors.New("")
		},
	})

	ctx := context.Background()
	_, err := h.Handle(ctx, &yarpc.Buffer{})
	require.Error(t, err)
}
