// Copyright (c) 2021 Uber Technologies, Inc.
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

package firstoutboundmiddleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/firstoutboundmiddleware"
	"go.uber.org/yarpc/yarpctest"
)

func TestFirstOutboundMidleware(t *testing.T) {
	out := yarpctest.NewFakeTransport().NewOutbound(nil,
		yarpctest.OutboundCallOverride(
			func(context.Context, *transport.Request) (*transport.Response, error) { return nil, nil },
		),
		yarpctest.OutboundCallStreamOverride(
			func(context.Context, *transport.StreamRequest) (*transport.ClientStream, error) { return nil, nil },
		),
		yarpctest.OutboundCallOnewayOverride(
			func(context.Context, *transport.Request) (transport.Ack, error) { return nil, nil },
		),
	)

	t.Run("unary", func(t *testing.T) {
		req := &transport.Request{Transport: "", CallerProcedure: "" /* not set */}

		outWithMiddleware := middleware.ApplyUnaryOutbound(out, firstoutboundmiddleware.New())
		ctx := yarpctest.ContextWithCall(context.Background(), &yarpctest.Call{Transport: "", Procedure: "ABC"})
		_, err := outWithMiddleware.Call(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, "fake", string(req.Transport))
		assert.Equal(t, "", string(req.CallerProcedure))
	})

	t.Run("oneway", func(t *testing.T) {
		req := &transport.Request{Transport: "" /* not set */}

		outWithMiddleware := middleware.ApplyOnewayOutbound(out, firstoutboundmiddleware.New())
		ctx := yarpctest.ContextWithCall(context.Background(), &yarpctest.Call{Transport: "", Procedure: "ABC"})
		_, err := outWithMiddleware.CallOneway(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, "fake", string(req.Transport))
		assert.Equal(t, "", string(req.CallerProcedure))
	})

	t.Run("stream", func(t *testing.T) {
		streamReq := &transport.StreamRequest{Meta: &transport.RequestMeta{Transport: "" /* not set */}}

		outWithMiddleware := middleware.ApplyStreamOutbound(out, firstoutboundmiddleware.New())
		ctx := yarpctest.ContextWithCall(context.Background(), &yarpctest.Call{Transport: "", Procedure: "ABC"})
		_, err := outWithMiddleware.CallStream(ctx, streamReq)
		require.NoError(t, err)

		assert.Equal(t, "fake", string(streamReq.Meta.Transport))
		assert.Equal(t, "", string(streamReq.Meta.CallerProcedure))
	})
}
