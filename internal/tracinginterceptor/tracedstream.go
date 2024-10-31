// Copyright (c) 2024 Uber Technologies, Inc.
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
	"io"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
)

var (
	_ transport.StreamCloser = (*tracedClientStream)(nil)
)

type tracedClientStream struct {
	clientStream *transport.ClientStream
	span         opentracing.Span
	closed       atomic.Bool
}

// Context returns the context for the traced client stream.
// This method delegates to the underlying clientStream's Context method,
// allowing access to the context associated with the stream.
func (t *tracedClientStream) Context() context.Context {
	return t.clientStream.Context()
}

// Request returns the metadata about the request.
func (t *tracedClientStream) Request() *transport.StreamRequest {
	return t.clientStream.Request()
}

// SendMessage sends a message over the stream. If an error occurs, it closes the span with error details.
func (t *tracedClientStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	if err := t.clientStream.SendMessage(ctx, msg); err != nil {
		return t.closeWithErr(err)
	}
	return nil
}

// ReceiveMessage receives a message from the stream. If an error occurs or EOF is reached, it closes the span.
func (t *tracedClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.clientStream.ReceiveMessage(ctx)
	if err != nil {
		if err == io.EOF {
			return msg, t.closeWithErr(nil)
		}
		return msg, t.closeWithErr(err)
	}
	return msg, nil
}

// Close closes the stream and updates the span with any final error details.
func (t *tracedClientStream) Close(ctx context.Context) error {
	return t.closeWithErr(t.clientStream.Close(ctx))
}

// closeWithErr closes the span with error details, ensuring it is only closed once.
func (t *tracedClientStream) closeWithErr(err error) error {
	if !t.closed.Swap(true) {
		t.span.Finish()
		return updateSpanWithErrorDetails(t.span, err != nil, nil, err)
	}
	return err
}
