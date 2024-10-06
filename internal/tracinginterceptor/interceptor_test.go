package tracinginterceptor

import (
	"context"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"testing"
)

// Define UnaryHandlerFunc to adapt a function into a UnaryHandler.
type UnaryHandlerFunc func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error

func (f UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return f(ctx, req, resw)
}

// Define OnewayHandlerFunc to adapt a function into a OnewayHandler.
type OnewayHandlerFunc func(ctx context.Context, req *transport.Request) error

func (f OnewayHandlerFunc) HandleOneway(ctx context.Context, req *transport.Request) error {
	return f(ctx, req)
}

// Define OnewayOutboundFunc to adapt a function into an OnewayOutbound.
type OnewayOutboundFunc func(ctx context.Context, req *transport.Request) (transport.Ack, error)

func (f OnewayOutboundFunc) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return f(ctx, req)
}

// Define StreamHandlerFunc to adapt a function into a StreamHandler.
type StreamHandlerFunc func(s *transport.ServerStream) error

func (f StreamHandlerFunc) HandleStream(s *transport.ServerStream) error {
	return f(s)
}

// Setup mock tracer
func setupMockTracer() *mocktracer.MockTracer {
	return mocktracer.New()
}

// TestUnaryInboundHandle tests the Handle method for Unary Inbound
func TestUnaryInboundHandle(t *testing.T) {
	tracer := setupMockTracer()
	interceptor := tracinginterceptor.New(tracinginterceptor.Params{
		Tracer:    tracer,
		Transport: "http",
	})

	handlerCalled := false
	handler := UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		handlerCalled = true
		return nil
	})

	ctx := context.Background()
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "procedure",
		Headers:   transport.Headers{},
	}

	err := interceptor.Handle(ctx, req, nil, handler)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)

	finishedSpans := tracer.FinishedSpans()
	assert.Len(t, finishedSpans, 1)
	span := finishedSpans[0]
	assert.Equal(t, "procedure", span.OperationName)
	assert.False(t, span.Tag("error").(bool))
}

// TestUnaryOutboundCall tests the Call method for Unary Outbound
func TestUnaryOutboundCall(t *testing.T) {
	tracer := setupMockTracer()
	interceptor := tracinginterceptor.New(tracinginterceptor.Params{
		Tracer:    tracer,
		Transport: "http",
	})

	outboundCalled := false
	outbound := transport.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request) (*transport.Response, error) {
		outboundCalled = true
		return &transport.Response{}, nil
	})

	ctx := context.Background()
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "procedure",
		Headers:   transport.Headers{},
	}

	res, err := interceptor.Call(ctx, req, outbound)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, outboundCalled)

	finishedSpans := tracer.FinishedSpans()
	assert.Len(t, finishedSpans, 1)
	span := finishedSpans[0]
	assert.Equal(t, "procedure", span.OperationName)
	assert.False(t, span.Tag("error").(bool))
}
