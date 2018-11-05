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

package internalyarpcobservability

import (
	"context"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/v2"
	"go.uber.org/zap"
)

var (
	_ yarpc.UnaryInboundTransportMiddleware  = (*Middleware)(nil)
	_ yarpc.UnaryOutboundTransportMiddleware = (*Middleware)(nil)

	_ yarpc.StreamInboundTransportMiddleware  = (*Middleware)(nil)
	_ yarpc.StreamOutboundTransportMiddleware = (*Middleware)(nil)
)

// Middleware is logging and metrics middleware for all RPC types.
type Middleware struct {
	graph graph
}

// NewMiddleware constructs a Middleware.
func NewMiddleware(logger *zap.Logger, scope *metrics.Scope, extract ContextExtractor) *Middleware {
	return &Middleware{newGraph(scope, logger, extract)}
}

// Handle implements yarpc.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer, h yarpc.UnaryTransportHandler) (*yarpc.Response, *yarpc.Buffer, error) {
	call := m.graph.begin(ctx, yarpc.Unary, _directionInbound, req)
	res, resBuf, err := h.Handle(ctx, req, reqBuf)

	isApplicationError := false
	if res != nil {
		isApplicationError = res.ApplicationError != nil
	}
	call.EndWithAppError(err, isApplicationError)
	return res, resBuf, err
}

// Call implements yarpc.UnaryOutbound.
func (m *Middleware) Call(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer, out yarpc.UnaryOutbound) (*yarpc.Response, *yarpc.Buffer, error) {
	call := m.graph.begin(ctx, yarpc.Unary, _directionOutbound, req)
	res, resBuf, err := out.Call(ctx, req, reqBuf)

	isApplicationError := false
	if res != nil {
		isApplicationError = res.ApplicationError != nil
	}
	call.EndWithAppError(err, isApplicationError)
	return res, resBuf, err
}

// HandleStream implements yarpc.StreamInbound.
func (m *Middleware) HandleStream(serverStream *yarpc.ServerStream, h yarpc.StreamTransportHandler) error {
	call := m.graph.begin(serverStream.Context(), yarpc.Streaming, _directionInbound, serverStream.Request())
	err := h.HandleStream(serverStream)
	// TODO(pedge): wrap the *yarpc.ServerStream?
	call.End(err)
	return err
}

// CallStream implements yarpc.StreamOutbound.
func (m *Middleware) CallStream(ctx context.Context, request *yarpc.Request, out yarpc.StreamOutbound) (*yarpc.ClientStream, error) {
	call := m.graph.begin(ctx, yarpc.Streaming, _directionOutbound, request)
	clientStream, err := out.CallStream(ctx, request)
	// TODO(pedge): wrap the *yarpc.ClientStream?
	call.End(err)
	return clientStream, err
}
