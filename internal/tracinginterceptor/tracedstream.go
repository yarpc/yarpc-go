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
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

// tracedServerStream implements the transport.ServerStream and additional tracing capabilities.
type tracedServerStream struct {
	transport.ServerStream
	span             opentracing.Span
	isApplicationErr bool
	appErrorMeta     *transport.ApplicationErrorMeta
	log              *zap.Logger // Adding logger for error handling
}

// newTracedServerStream creates a new tracedServerStream with embedded ServerStream.
func newTracedServerStream(s transport.ServerStream, span opentracing.Span, logger *zap.Logger) *tracedServerStream {
	return &tracedServerStream{
		ServerStream: s,
		span:         span,
		log:          logger,
	}
}

func (t *tracedServerStream) IsApplicationError() bool {
	return t.isApplicationErr
}

func (t *tracedServerStream) ApplicationErrorMeta() *transport.ApplicationErrorMeta {
	return t.appErrorMeta
}

func (t *tracedServerStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	err := t.ServerStream.SendMessage(ctx, msg)
	if updateErr := updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err); updateErr != nil {
		t.log.Error("Failed to update span with error details", zap.Error(updateErr))
	}
	return err
}

func (t *tracedServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.ServerStream.ReceiveMessage(ctx)
	if updateErr := updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err); updateErr != nil {
		t.log.Error("Failed to update span with error details", zap.Error(updateErr))
	}
	return msg, err
}

// tracedClientStream wraps ClientStream with tracing and error metadata.
type tracedClientStream struct {
	transport.ClientStream
	span             opentracing.Span
	isApplicationErr bool
	appErrorMeta     *transport.ApplicationErrorMeta
	log              *zap.Logger // Adding logger for error handling
}

func newTracedClientStream(cs *transport.ClientStream, span opentracing.Span, logger *zap.Logger) *tracedClientStream {
	return &tracedClientStream{
		ClientStream: *cs,
		span:         span,
		log:          logger,
	}
}

func (t *tracedClientStream) IsApplicationError() bool {
	return t.isApplicationErr
}

func (t *tracedClientStream) ApplicationErrorMeta() *transport.ApplicationErrorMeta {
	return t.appErrorMeta
}

func (t *tracedClientStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	err := t.ClientStream.SendMessage(ctx, msg)
	if updateErr := updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err); updateErr != nil {
		t.log.Error("Failed to update span with error details", zap.Error(updateErr))
	}
	return err
}

func (t *tracedClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.ClientStream.ReceiveMessage(ctx)
	if updateErr := updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err); updateErr != nil {
		t.log.Error("Failed to update span with error details", zap.Error(updateErr))
	}
	return msg, err
}

func (t *tracedClientStream) Close(ctx context.Context) error {
	err := t.ClientStream.Close(ctx)
	if updateErr := updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err); updateErr != nil {
		t.log.Error("Failed to update span with error details", zap.Error(updateErr))
	}
	t.span.Finish()
	return err
}
