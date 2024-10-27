package tracinginterceptor

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
)

// serverStreamWrapper is an interface that includes all methods of ServerStream.
type serverStreamWrapper interface {
	transport.ServerStream
	IsApplicationError() bool
	ApplicationErrorMeta() *transport.ApplicationErrorMeta
}

// tracedServerStream implements transport.ServerStream and additional tracing capabilities.
type tracedServerStream struct {
	transport.ServerStream
	span             opentracing.Span
	isApplicationErr bool
	appErrorMeta     *transport.ApplicationErrorMeta
}

// newTracedServerStream creates a new tracedServerStream with embedded ServerStream.
func newTracedServerStream(s transport.ServerStream, span opentracing.Span) *tracedServerStream {
	return &tracedServerStream{
		ServerStream: s,
		span:         span,
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
	updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err)
	return err
}

func (t *tracedServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.ServerStream.ReceiveMessage(ctx)
	updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err)
	return msg, err
}

// tracedClientStream wraps ClientStream with tracing and error metadata.
type tracedClientStream struct {
	transport.ClientStream
	span             opentracing.Span
	isApplicationErr bool
	appErrorMeta     *transport.ApplicationErrorMeta
}

func newTracedClientStream(cs *transport.ClientStream, span opentracing.Span) *tracedClientStream {
	return &tracedClientStream{
		ClientStream: *cs,
		span:         span,
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
	updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err)
	return err
}

func (t *tracedClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := t.ClientStream.ReceiveMessage(ctx)
	updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err)
	return msg, err
}

func (t *tracedClientStream) Close(ctx context.Context) error {
	err := t.ClientStream.Close(ctx)
	updateSpanWithErrorDetails(t.span, t.isApplicationErr, t.appErrorMeta, err)
	t.span.Finish()
	return err
}
