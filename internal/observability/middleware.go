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

	// Levels specify log levels for various classes of requests.
	Levels LevelsConfig
}

// LevelsConfig specifies log level overrides for inbound traffic, outbound
// traffic, or the defaults for either.
type LevelsConfig struct {
	Default  DirectionalLevelsConfig
	Inbound  DirectionalLevelsConfig
	Outbound DirectionalLevelsConfig
}

// DirectionalLevelsConfig may override the log levels for any combination of
// successes, failures, and application errors.
type DirectionalLevelsConfig struct {
	// Log level used to log successful calls.
	//
	// Defaults to DebugLevel.
	Success *zapcore.Level

	// Log level used to log failed calls.
	// This includes low-level network errors, TChannel error frames, etc.
	//
	// Defaults to ErrorLevel.
	Failure *zapcore.Level

	// Log level used to log calls that failed with an application error.
	// All Thrift exceptions are considered application errors.
	//
	// Defaults to ErrorLevel.
	ApplicationError *zapcore.Level
}

// NewMiddleware constructs an observability middleware with the provided
// configuration.
func NewMiddleware(cfg Config) *Middleware {
	m := &Middleware{newGraph(cfg.Scope, cfg.Logger, cfg.ContextExtractor)}

	// Apply the default levels
	applyLogLevelsConfig(&m.graph.inboundLevels, &cfg.Levels.Default)
	applyLogLevelsConfig(&m.graph.outboundLevels, &cfg.Levels.Default)
	// Override with direction-specific levels
	applyLogLevelsConfig(&m.graph.inboundLevels, &cfg.Levels.Inbound)
	applyLogLevelsConfig(&m.graph.outboundLevels, &cfg.Levels.Outbound)

	return m
}

func applyLogLevelsConfig(dst *levels, src *DirectionalLevelsConfig) {
	if level := src.Success; level != nil {
		dst.success = *src.Success
	}
	if level := src.Failure; level != nil {
		dst.failure = *src.Failure
	}
	if level := src.ApplicationError; level != nil {
		dst.applicationError = *src.ApplicationError
	}
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
	call.EndStreamHandshake()

	wrappedStream, err := transport.NewServerStream(newServerStreamWrapper(call, serverStream))
	if err != nil {
		// This will never happen since transport.NewServerStream only returns an
		// error for nil streams. In the nearly impossible situation where we do, we
		// fall back to using the original, unwrapped stream.
		m.graph.logger.DPanic("transport.ServerStream wrapping should never fail, streaming metrics are disabled")
		wrappedStream = serverStream
	}

	err = h.HandleStream(wrappedStream)
	call.EndStream(err)
	return err
}

// CallStream implements middleware.StreamOutbound.
func (m *Middleware) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	call := m.graph.begin(ctx, transport.Streaming, _directionOutbound, request.Meta.ToRequest())
	clientStream, err := out.CallStream(ctx, request)
	call.EndStreamHandshakeWithError(err)
	if err != nil {
		return nil, err
	}

	wrappedStream, err := transport.NewClientStream(newClientStreamWrapper(call, clientStream))
	if err != nil {
		// This will never happen since transport.NewClientStream only returns an
		// error for nil streams. In the nearly impossible situation where we do, we
		// fall back to using the original, unwrapped stream.
		m.graph.logger.DPanic("transport.ClientStream wrapping should never fail, streaming metrics are disabled")
		wrappedStream = clientStream
	}
	return wrappedStream, nil
}
