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

package observability

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

var _writerPool = sync.Pool{New: func() interface{} {
	return &writer{}
}}

// writer wraps a transport.ResponseWriter so the observing middleware can
// detect application errors.
type writer struct {
	transport.ResponseWriter

	isApplicationError bool
}

func newWriter(rw transport.ResponseWriter) *writer {
	w := _writerPool.Get().(*writer)
	w.isApplicationError = false
	w.ResponseWriter = rw
	return w
}

func (w *writer) SetApplicationError() {
	w.isApplicationError = true
	w.ResponseWriter.SetApplicationError()
}

func (w *writer) free() {
	_writerPool.Put(w)
}

// Middleware is logging and metrics middleware for all RPC types.
type Middleware struct {
	graph graph
}

// NewMiddleware constructs a Middleware.
func NewMiddleware(logger *zap.Logger, scope *metrics.Scope, extract ContextExtractor) *Middleware {
	return &Middleware{newGraph(scope, logger, extract)}
}

// Handle implements middleware.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *transport.Request, w transport.ResponseWriter, h transport.UnaryHandler) error {
	call := m.graph.begin(ctx, transport.Unary, _directionInbound, req)
	wrappedWriter := newWriter(w)
	err := h.Handle(ctx, req, wrappedWriter)
	call.EndWithAppError(err, wrappedWriter.isApplicationError)
	wrappedWriter.free()
	return err
}

// Call implements middleware.UnaryOutbound.
func (m *Middleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	call := m.graph.begin(ctx, transport.Unary, _directionOutbound, req)
	res, err := out.Call(ctx, req)

	var appErr error
	if res != nil && res.ApplicationError {
		// This is a stub until outbound application error could be properly
		// propagated
		appErr = errors.New("application_error")
	}

	call.EndWithAppError(err, appErr)

	return res, err
}

// HandleOneway implements middleware.OnewayInbound.
func (m *Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	call := m.graph.begin(ctx, transport.Oneway, _directionInbound, req)
	err := h.HandleOneway(ctx, req)
	call.End(err)
	return err
}

// CallOneway implements middleware.OnewayOutbound.
func (m *Middleware) CallOneway(ctx context.Context, req *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	call := m.graph.begin(ctx, transport.Oneway, _directionOutbound, req)
	ack, err := out.CallOneway(ctx, req)
	call.End(err)
	return ack, err
}

// HandleStream implements middleware.StreamInbound.
func (m *Middleware) HandleStream(serverStream *transport.ServerStream, h transport.StreamHandler) error {
	call := m.graph.begin(serverStream.Context(), transport.Streaming, _directionInbound, serverStream.Request().Meta.ToRequest())
	err := h.HandleStream(serverStream)
	// TODO(pedge): wrap the *transport.ServerStream?
	call.End(err)
	return err
}

// CallStream implements middleware.StreamOutbound.
func (m *Middleware) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	call := m.graph.begin(ctx, transport.Streaming, _directionOutbound, request.Meta.ToRequest())
	clientStream, err := out.CallStream(ctx, request)
	// TODO(pedge): wrap the *transport.ClientStream?
	call.End(err)
	return clientStream, err
}
