// Copyright (c) 2019 Uber Technologies, Inc.
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
	"sync"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// Config configures the observability middleware.
type Config struct {
	// Logger to which messages will be logged.
	Logger *zap.Logger

	// Scope to which metrics are emitted.
	Scope *metrics.Scope

	// Extracts request-scoped information from the context for logging.
	ContextExtractor ContextExtractor

	// Log level used to log successful inbound and outbound calls.
	//
	// Defaults to DebugLevel.
	SuccessLevel *zapcore.Level

	// Log level used to log failed inbound and outbound calls. This includes
	// low-level network errors, TChannel error frames, etc.
	//
	// Defaults to ErrorLevel.
	FailureLevel *zapcore.Level

	// Log level used to log calls that failed with an application error. All
	// Thrift exceptions are considered application errors.
	//
	// Defaults to ErrorLevel.
	ApplicationErrorLevel *zapcore.Level
}

// NewMiddleware constructs an observability middleware with the provided
// configuration.
func NewMiddleware(cfg Config) *Middleware {
	m := &Middleware{newGraph(cfg.Scope, cfg.Logger, cfg.ContextExtractor)}

	if lvl := cfg.SuccessLevel; lvl != nil {
		m.graph.succLevel = *lvl
	}
	if lvl := cfg.FailureLevel; lvl != nil {
		m.graph.failLevel = *lvl
	}
	if lvl := cfg.ApplicationErrorLevel; lvl != nil {
		m.graph.appErrLevel = *lvl
	}

	return m
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

	isApplicationError := false
	if res != nil {
		isApplicationError = res.ApplicationError
	}
	call.EndWithAppError(err, isApplicationError)
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

	wrappedStream, err := transport.NewServerStream(newServerStreamWrapper(call, serverStream))
	if err != nil {
		// This will never happen since transport.NewServerStream only returns an
		// error for nil streams. In the nearly impossible situation where we do, we
		// fall back to using the original, unwrapped stream.
		m.graph.logger.Error("transport.ServerStream wrapping should never fail, streaming metrics are disabled")
		wrappedStream = serverStream
	}

	err = h.HandleStream(wrappedStream)
	// TODO: end stream
	call.End(err)
	return err
}

// CallStream implements middleware.StreamOutbound.
func (m *Middleware) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	call := m.graph.begin(ctx, transport.Streaming, _directionOutbound, request.Meta.ToRequest())
	clientStream, err := out.CallStream(ctx, request)
	call.End(err)
	if err != nil {
		return nil, err
	}

	wrappedStream, err := transport.NewClientStream(newClientStreamWrapper(call, clientStream))
	if err != nil {
		// This will never happen since transport.NewClientStream only returns an
		// error for nil streams. In the nearly impossible situation where we do, we
		// fall back to using the original, unwrapped stream.
		m.graph.logger.Error("transport.ClientStream wrapping should never fail, streaming metrics are disabled")
		wrappedStream = clientStream
	}
	return wrappedStream, nil
}
