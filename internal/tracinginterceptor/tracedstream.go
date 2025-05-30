// Copyright (c) 2025 Uber Technologies, Inc.
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

	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
)

var (
	_ transport.StreamCloser        = (*tracedClientStream)(nil)
	_ transport.StreamHeadersReader = (*tracedClientStream)(nil)
	_ transport.StreamHeadersSender = (*tracedServerStream)(nil)
)

// tracedClientStream wraps the transport.ClientStream to add tracing.
type tracedClientStream struct {
	clientStream *transport.ClientStream
	span         opentracing.Span
	closed       atomic.Bool
}

// Context returns the context associated with the client stream.
func (t *tracedClientStream) Context() context.Context {
	return t.clientStream.Context()
}

// Request returns the initial StreamRequest metadata for the client stream.
func (t *tracedClientStream) Request() *transport.StreamRequest {
	return t.clientStream.Request()
}

// SendMessage delegates to the underlying stream's SendMessage and updates the span on error.
func (t *tracedClientStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	if err := t.clientStream.SendMessage(ctx, msg); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// ReceiveMessage delegates to the underlying stream's ReceiveMessage and updates the span on error or EOF.
func (t *tracedClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.clientStream.ReceiveMessage(ctx)
	if err != nil {
		return nil, t.closeWithErr(err)
	}
	return msg, nil
}

// Close closes the client stream and updates the span with any final error.
func (t *tracedClientStream) Close(ctx context.Context) error {
	return t.closeWithErr(t.clientStream.Close(ctx))
}

// Headers implements transport.StreamHeadersReader.
// It reads the initial stream response headers and updates the span on error.
func (t *tracedClientStream) Headers() (transport.Headers, error) {
	headers, err := t.clientStream.Headers()
	if err != nil {
		return headers, t.closeWithErr(err)
	}
	return headers, nil
}

// closeWithErr finishes the span once and tags it if there was an error.
func (t *tracedClientStream) closeWithErr(err error) error {
	if !t.closed.Swap(true) && t.span != nil {
		t.span.Finish()
		// treat EOF as non-error; everything else is an error
		isErr := err != nil && !errors.Is(err, io.EOF)
		// updateSpanWithErrorDetails will tag the span appropriately and return the original error
		_ = updateSpanWithErrorDetails(t.span, isErr, nil, err)
	}
	return err
}

// tracedServerStream wraps a transport.ServerStream to add tracing.
type tracedServerStream struct {
	serverStream *transport.ServerStream
	span         opentracing.Span
	closed       atomic.Bool
}

// Context returns the context associated with the server stream.
func (t *tracedServerStream) Context() context.Context {
	return t.serverStream.Context()
}

// Request returns the initial StreamRequest metadata for the server stream.
func (t *tracedServerStream) Request() *transport.StreamRequest {
	return t.serverStream.Request()
}

// SendMessage delegates to the underlying stream's SendMessage and updates the span on error.
func (t *tracedServerStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	if err := t.serverStream.SendMessage(ctx, msg); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// ReceiveMessage delegates to the underlying stream's ReceiveMessage and updates the span on error or EOF.
func (t *tracedServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.serverStream.ReceiveMessage(ctx)
	if err != nil {
		return nil, t.closeWithErr(err)
	}
	return msg, nil
}

// SendHeaders implements transport.StreamHeadersSender.
func (t *tracedServerStream) SendHeaders(h transport.Headers) error {
	if err := t.serverStream.SendHeaders(h); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// closeWithErr finishes the span once and tags it if there was an error.
func (t *tracedServerStream) closeWithErr(err error) error {
	if !t.closed.Swap(true) && t.span != nil {
		t.span.Finish()
		// treat EOF as non-error; everything else is an error
		isErr := err != nil && !errors.Is(err, io.EOF)
		// updateSpanWithErrorDetails will tag the span appropriately and return the original error
		_ = updateSpanWithErrorDetails(t.span, isErr, nil, err)
	}
	return err
}
