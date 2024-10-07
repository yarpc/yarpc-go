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
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_ interceptor.UnaryInbound  = (*Interceptor)(nil)
	_ interceptor.UnaryOutbound = (*Interceptor)(nil)
)

// Params defines the parameters for creating the Middleware
type Params struct {
	Tracer    opentracing.Tracer
	Transport string
}

// Interceptor is the tracing interceptor for Unary RPC types.
// It handles both observability and inter-process context propagation.
type Interceptor struct {
	tracer            opentracing.Tracer
	transport         string
	propagationFormat opentracing.BuiltinFormat
}

// New constructs a tracing interceptor with the provided configuration.
func New(p Params) *Interceptor {
	m := &Interceptor{
		tracer:            p.Tracer,
		transport:         p.Transport,
		propagationFormat: transport.GetPropagationFormat(p.Transport),
	}
	if m.tracer == nil {
		m.tracer = opentracing.GlobalTracer()
	}
	return m
}

// Handle is the tracing handler for Unary Inbound requests.
func (m *Interceptor) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	parentSpanCtx, _ := m.tracer.Extract(m.propagationFormat, transport.GetPropagationCarrier(req.Headers.Items(), req.Transport))
	tags := ExtractTracingTags(req)

	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            m.tracer,
		TransportName:     req.Transport,
		StartTime:         time.Now(),
		ExtraTags:         tags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	err := h.Handle(ctx, req, resw)
	return updateSpanWithError(span, err)
}

// Call is the tracing handler for Unary Outbound requests.
func (m *Interceptor) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	tags := ExtractTracingTags(req)

	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        m.tracer,
		TransportName: m.transport,
		StartTime:     time.Now(),
		ExtraTags:     tags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	tracingHeaders := make(map[string]string)
	if err := m.tracer.Inject(span.Context(), m.propagationFormat, transport.GetPropagationCarrier(tracingHeaders, m.transport)); err != nil {
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
