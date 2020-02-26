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

package types

import (
	"context"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// GiveHeader is an API for giving headers to a Request or a Response.
type GiveHeader struct {
	api.NoopLifecycle

	Key   string
	Value string
}

// ApplyClientStreamRequest implements api.ClientStreamRequestOption.
func (h *GiveHeader) ApplyClientStreamRequest(opts *api.ClientStreamRequestOpts) {
	opts.GiveRequest.Meta.Headers = opts.GiveRequest.Meta.Headers.With(h.Key, h.Value)
}

// ApplyRequest implements RequestOption.
func (h *GiveHeader) ApplyRequest(opts *api.RequestOpts) {
	opts.GiveRequest.Headers = opts.GiveRequest.Headers.With(h.Key, h.Value)
}

// Handle implements middleware.UnaryInbound.
func (h *GiveHeader) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	err := handler.Handle(ctx, req, resw)
	resw.AddHeaders(transport.NewHeaders().With(h.Key, h.Value))
	return err
}

// WantHeader is an API for asserting headers sent on a Request or a Response.
type WantHeader struct {
	api.SafeTestingTBOnStart
	api.NoopStop

	Key   string
	Value string
}

// ApplyServerStream implements ServerStreamAction.
func (h *WantHeader) ApplyServerStream(c *transport.ServerStream) error {
	actualValue, ok := c.Request().Meta.Headers.Get(h.Key)
	require.True(h.GetTestingTB(), ok, "header %q was not set on the request", h.Key)
	require.Equal(h.GetTestingTB(), actualValue, h.Value, "headers did not match for %q", h.Key)
	return nil
}

// ApplyRequest implements RequestOption.
func (h *WantHeader) ApplyRequest(opts *api.RequestOpts) {
	opts.WantResponse.Headers = opts.WantResponse.Headers.With(h.Key, h.Value)
}

// Handle implements middleware.UnaryInbound.
func (h *WantHeader) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	actualValue, ok := req.Headers.Get(h.Key)
	require.True(h.GetTestingTB(), ok, "header %q was not set on the request", h.Key)
	require.Equal(h.GetTestingTB(), actualValue, h.Value, "headers did not match for %q", h.Key)
	return handler.Handle(ctx, req, resw)
}
