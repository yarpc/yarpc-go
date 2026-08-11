// Copyright (c) 2026 Uber Technologies, Inc.
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

package tracinginterceptor

import (
	"context"
	"errors"
	"io"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
)

var (
	_ transport.StreamCloser        = (*tracedOTelClientStream)(nil)
	_ transport.StreamHeadersReader = (*tracedOTelClientStream)(nil)
	_ transport.StreamHeadersSender = (*tracedOTelServerStream)(nil)
)

// spanErrorFor returns the error that should be recorded on a stream's span.
// io.EOF is the normal termination signal for a stream, not a failure.
func spanErrorFor(err error) error {
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// tracedOTelClientStream wraps a transport.ClientStream to add tracing. It owns
// the lifetime of its span, because the span outlives the CallStream call that
// created it.
type tracedOTelClientStream struct {
	clientStream *transport.ClientStream
	span         trace.Span
	closed       atomic.Bool
}

// Context returns the context associated with the client stream.
func (t *tracedOTelClientStream) Context() context.Context {
	return t.clientStream.Context()
}

// Request returns the initial StreamRequest metadata for the client stream.
func (t *tracedOTelClientStream) Request() *transport.StreamRequest {
	return t.clientStream.Request()
}

// SendMessage delegates to the underlying stream and ends the span on error.
func (t *tracedOTelClientStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	if err := t.clientStream.SendMessage(ctx, msg); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// ReceiveMessage delegates to the underlying stream and ends the span on error
// or end of stream.
func (t *tracedOTelClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.clientStream.ReceiveMessage(ctx)
	if err != nil {
		return nil, t.closeWithErr(err)
	}
	return msg, nil
}

// Close closes the client stream and ends the span with any final error.
func (t *tracedOTelClientStream) Close(ctx context.Context) error {
	return t.closeWithErr(t.clientStream.Close(ctx))
}

// Headers implements transport.StreamHeadersReader. It reads the initial stream
// response headers and ends the span on error.
func (t *tracedOTelClientStream) Headers() (transport.Headers, error) {
	headers, err := t.clientStream.Headers()
	if err != nil {
		return headers, t.closeWithErr(err)
	}
	return headers, nil
}

// closeWithErr records the outcome and ends the span, at most once.
func (t *tracedOTelClientStream) closeWithErr(err error) error {
	if !t.closed.Swap(true) && t.span != nil {
		// Attributes must be recorded before End; afterwards they are dropped.
		_ = otelUpdateSpanWithErrorDetails(t.span, false, nil, spanErrorFor(err))
		t.span.End()
	}
	return err
}

// wrapOTelClientStream wraps the traced stream for return to the caller. The
// wrap only fails for a nil stream, in which case the span is ended and the
// untraced stream is returned rather than failing the call.
func wrapOTelClientStream(s *tracedOTelClientStream) *transport.ClientStream {
	wrapped, err := transport.NewClientStream(s)
	if err != nil {
		s.span.End()
		return s.clientStream
	}
	return wrapped
}

// tracedOTelServerStream wraps a transport.ServerStream to add tracing. Both
// this and HandleStream end the span, matching the OpenTracing server stream:
// whichever happens first wins, and end of stream on receive is the common
// case.
type tracedOTelServerStream struct {
	serverStream *transport.ServerStream
	span         trace.Span
	ctx          context.Context // enriched context carrying the span and baggage
	closed       atomic.Bool
}

// Context returns the enriched context if one was supplied, otherwise the
// underlying stream's context.
func (t *tracedOTelServerStream) Context() context.Context {
	if t.ctx != nil {
		return t.ctx
	}
	return t.serverStream.Context()
}

// Request returns the initial StreamRequest metadata for the server stream.
func (t *tracedOTelServerStream) Request() *transport.StreamRequest {
	return t.serverStream.Request()
}

// SendMessage delegates to the underlying stream and ends the span on error.
func (t *tracedOTelServerStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	if err := t.serverStream.SendMessage(ctx, msg); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// ReceiveMessage delegates to the underlying stream and ends the span on error
// or end of stream.
func (t *tracedOTelServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.serverStream.ReceiveMessage(ctx)
	if err != nil {
		return nil, t.closeWithErr(err)
	}
	return msg, nil
}

// SendHeaders implements transport.StreamHeadersSender.
func (t *tracedOTelServerStream) SendHeaders(h transport.Headers) error {
	if err := t.serverStream.SendHeaders(h); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// closeWithErr records the outcome and ends the span, at most once.
func (t *tracedOTelServerStream) closeWithErr(err error) error {
	if !t.closed.Swap(true) && t.span != nil {
		// Attributes must be recorded before End; afterwards they are dropped.
		_ = otelUpdateSpanWithErrorDetails(t.span, false, nil, spanErrorFor(err))
		t.span.End()
	}
	return err
}
