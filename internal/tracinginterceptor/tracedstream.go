package tracinginterceptor

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

// tracedServerStream implements transport.ServerStream and additional tracing capabilities.
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
