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

package tracingmiddleware

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/firstoutboundmiddleware"
	"go.uber.org/yarpc/internal/outboundmiddleware"
	"go.uber.org/yarpc/yarpcerrors"
)

// Params defines the parameters for creating the Middleware
type Params struct {
	Tracer opentracing.Tracer
}

// Middleware is the tracing middleware for all RPC types.
// It handles both observability and inter-process context propagation.
type Middleware struct {
	tracer opentracing.Tracer
}

// NewMiddleware constructs an observability middleware with the provided
// configuration.
func NewMiddleware(p Params) *Middleware {
	m := &Middleware{
		tracer: p.Tracer,
	}
	if m.tracer == nil {
		m.tracer = opentracing.GlobalTracer()
	}

	return m
}

var _ middleware.UnaryInbound = (*Middleware)(nil)
var _ middleware.OnewayInbound = (*Middleware)(nil)
var _ middleware.StreamInbound = (*Middleware)(nil)
var _ middleware.UnaryOutbound = (*Middleware)(nil)
var _ middleware.OnewayOutbound = (*Middleware)(nil)
var _ middleware.StreamOutbound = (*Middleware)(nil)

func (m Middleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	parentSpanCtx, _ := m.tracer.Extract(transport.GetPropagationFormat(req.Transport), transport.GetPropagationCarrier(req.Headers.Items(), req.Transport))
	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            m.tracer,
		TransportName:     req.Transport,
		StartTime:         time.Now(),
		// circular dependencies - we need to relocate the tracing tags
		// ExtraTags:         yarpc.OpentracingTags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	err := h.Handle(ctx, req, resw)
	return updateSpanWithError(span, err)
}

func (m Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	// TODO implement me
	panic("implement me")
}

func (m Middleware) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	// TODO implement me
	panic("implement me")
}

func (m Middleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        m.tracer,
		TransportName: req.Transport,
		StartTime:     time.Now(),
		// circular dependencies - we need to relocate the tracing tags
		//ExtraTags:     yarpc.OpentracingTags
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	tracingHeaders := make(map[string]string)
	if err := m.tracer.Inject(span.Context(), transport.GetPropagationFormat(req.Transport), transport.GetPropagationCarrier(tracingHeaders, req.Transport)); err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.String("event", "error"), log.String("message", err.Error()))
		return nil, err
	}
	for k, v := range tracingHeaders {
		req.Headers = req.Headers.With(k, v)
	}

	res, err := out.Call(ctx, req)
	return res, updateSpanWithOutboundError(span, res, err)
}

func (m Middleware) CallOneway(ctx context.Context, request *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	// TODO implement me
	panic("implement me")
}

func (m Middleware) CallStream(ctx context.Context, req *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	// TODO implement me
	// client stream is a bit more complex, as we need to intercept the clientStream.
	// We can refer yarpc its own implementation: https://github.com/yarpc/yarpc-go/blob/dev/transport/grpc/stream.go#L103
	// or opentracing contrib implementation: https://github.com/opentracing-contrib/go-grpc/blob/master/client.go#L131
	panic("implement me")
}

func updateSpanWithError(span opentracing.Span, err error) error {
	if err == nil {
		return err
	}

	ext.Error.Set(span, true)
	if yarpcerrors.IsStatus(err) {
		status := yarpcerrors.FromError(err)
		errCode := status.Code()
		span.SetTag("rpc.yarpc.status_code", errCode.String())
		span.SetTag("error.type", errCode.String())
		return err
	}

	span.SetTag("error.type", "unknown_internal_yarpc")
	return err
}

func updateSpanWithOutboundError(span opentracing.Span, res *transport.Response, err error) error {
	isApplicationError := false
	if res != nil {
		isApplicationError = res.ApplicationError
	}
	if err == nil && !isApplicationError {
		return err
	}

	ext.Error.Set(span, true)
	if yarpcerrors.IsStatus(err) {
		status := yarpcerrors.FromError(err)
		errCode := status.Code()
		span.SetTag("rpc.yarpc.status_code", errCode.String())
		span.SetTag("error.type", errCode.String())
		return err
	}

	if isApplicationError {
		span.SetTag("error.type", "application_error")
		return err
	}

	span.SetTag("error.type", "unknown_internal_yarpc")
	return err
}

// ApplyUnaryInbound apply tracing middleware to a unary inbound
// If yarpc dispatch is not used, you can use this method to apply tracing middleware to a unary inbound
func (m Middleware) ApplyUnaryInbound(h transport.UnaryHandler) transport.UnaryHandler {
	return middleware.ApplyUnaryInbound(h, m)
}

// ApplyUnaryOutbound apply tracing middleware to a unary outbound
// If yarpc dispatch is not used, you can use this method to apply tracing middleware to a unary outbound
func (m Middleware) ApplyUnaryOutbound(out transport.UnaryOutbound) transport.UnaryOutbound {
	first := firstoutboundmiddleware.New()
	return middleware.ApplyUnaryOutbound(out, outboundmiddleware.UnaryChain(first, m))
}
