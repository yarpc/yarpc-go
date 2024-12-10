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
	"go.uber.org/yarpc/transport/tchannel/tracing"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

var (
	_ interceptor.UnaryInbound   = (*Interceptor)(nil)
	_ interceptor.UnaryOutbound  = (*Interceptor)(nil)
	_ interceptor.OnewayInbound  = (*Interceptor)(nil)
	_ interceptor.OnewayOutbound = (*Interceptor)(nil)
	_ interceptor.StreamInbound  = (*Interceptor)(nil)
	_ interceptor.StreamOutbound = (*Interceptor)(nil)
)

// Params defines the parameters for creating the Interceptor
type Params struct {
	Tracer    opentracing.Tracer
	Transport string
	Logger    *zap.Logger
}

// Interceptor is the tracing interceptor for all RPC types.
type Interceptor struct {
	tracer            opentracing.Tracer
	transport         string
	propagationFormat opentracing.BuiltinFormat
	log               *zap.Logger
}

// PropagationCarrier is an interface to combine both reader and writer interface
type PropagationCarrier interface {
	opentracing.TextMapReader
	opentracing.TextMapWriter
}

// New constructs a tracing interceptor with the provided parameter.
func New(p Params) *Interceptor {
	i := &Interceptor{
		tracer:            p.Tracer,
		transport:         p.Transport,
		propagationFormat: getPropagationFormat(p.Transport),
		log:               p.Logger,
	}
	if i.tracer == nil {
		i.tracer = opentracing.GlobalTracer()
	}
	if i.log == nil {
		i.log = zap.NewNop()
	}
	return i
}

// Handle implements interceptor.UnaryInbound
func (i *Interceptor) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	parentSpanCtx, _ := i.tracer.Extract(i.propagationFormat, getPropagationCarrier(req.Headers.Items(), req.Transport))
	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            i.tracer,
		TransportName:     req.Transport,
		StartTime:         time.Now(),
		ExtraTags:         commonTracingTags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()
	err := h.Handle(ctx, req, resw)

	extendedWriter, ok := resw.(transport.ExtendedResponseWriter)
	if !ok {
		i.log.Debug("ResponseWriter does not implement ExtendedResponseWriter, passing false and nil for app error meta")
		return updateSpanWithErrorDetails(span, false, nil, err)
	}

	return updateSpanWithErrorDetails(span, extendedWriter.IsApplicationError(), extendedWriter.ApplicationErrorMeta(), err)
}

// Call implements interceptor.UnaryOutbound
func (i *Interceptor) Call(ctx context.Context, req *transport.Request, out interceptor.UnaryOutboundChain) (*transport.Response, error) {
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        i.tracer,
		TransportName: i.transport,
		StartTime:     time.Now(),
		ExtraTags:     commonTracingTags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	tracingHeaders := make(map[string]string)

	// We use i.transport here because this is an outbound call made by the interceptor.
	// In inbound handlers (e.g., Handle function), req.Transport is used because it's the transport from the incoming request.
	if err := i.tracer.Inject(span.Context(), i.propagationFormat, getPropagationCarrier(tracingHeaders, i.transport)); err != nil {
		span.LogFields(logFieldEventError, log.String("message", err.Error()))
	} else {
		for k, v := range tracingHeaders {
			req.Headers = req.Headers.With(k, v)
		}
	}

	res, err := out.Next(ctx, req)
	if res != nil {
		return res, updateSpanWithErrorDetails(span, res.ApplicationError, res.ApplicationErrorMeta, err)
	}
	return nil, updateSpanWithErrorDetails(span, false, nil, err)
}

// HandleOneway implements interceptor.OnewayInbound
func (i *Interceptor) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	parentSpanCtx, _ := i.tracer.Extract(i.propagationFormat, getPropagationCarrier(req.Headers.Items(), req.Transport))
	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            i.tracer,
		TransportName:     req.Transport,
		StartTime:         time.Now(),
		ExtraTags:         commonTracingTags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	err := h.HandleOneway(ctx, req)
	return updateSpanWithErrorDetails(span, false, nil, err)
}

// CallOneway implements interceptor.OnewayOutbound
func (i *Interceptor) CallOneway(ctx context.Context, req *transport.Request, out interceptor.DirectOnewayOutbound) (transport.Ack, error) {
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        i.tracer,
		TransportName: i.transport,
		StartTime:     time.Now(),
		ExtraTags:     commonTracingTags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	tracingHeaders := make(map[string]string)
	if err := i.tracer.Inject(span.Context(), i.propagationFormat, getPropagationCarrier(tracingHeaders, i.transport)); err != nil {
		span.LogFields(logFieldEventError, log.String("message", err.Error()))
	} else {
		for k, v := range tracingHeaders {
			req.Headers = req.Headers.With(k, v)
		}
	}

	ack, err := out.DirectCallOneway(ctx, req)
	return ack, updateSpanWithErrorDetails(span, false, nil, err)
}

// HandleStream implements interceptor.StreamInbound
func (i *Interceptor) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	req := s.Request()
	parentSpanCtx, _ := i.tracer.Extract(i.propagationFormat, getPropagationCarrier(req.Meta.Headers.Items(), i.transport))
	transportRequest := &transport.Request{
		Caller:    req.Meta.Caller,
		Service:   req.Meta.Service,
		Procedure: req.Meta.Procedure,
		Headers:   req.Meta.Headers,
		Transport: req.Meta.Transport,
	}

	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            i.tracer,
		TransportName:     s.Request().Meta.Transport,
		StartTime:         time.Now(),
		ExtraTags:         commonTracingTags,
	}
	_, span := extractOpenTracingSpan.Do(s.Context(), transportRequest)
	defer span.Finish()
	err := h.HandleStream(s)
	return updateSpanWithErrorDetails(span, err != nil, nil, err)
}

// CallStream implements interceptor.StreamOutbound
func (i *Interceptor) CallStream(ctx context.Context, req *transport.StreamRequest, out interceptor.DirectStreamOutbound) (*transport.ClientStream, error) {
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        i.tracer,
		TransportName: i.transport,
		StartTime:     time.Now(),
		ExtraTags:     commonTracingTags,
	}
	_, span := createOpenTracingSpan.Do(ctx, &transport.Request{
		Caller:    req.Meta.Caller,
		Service:   req.Meta.Service,
		Procedure: req.Meta.Procedure,
		Headers:   req.Meta.Headers,
		Transport: req.Meta.Transport,
	})

	// Inject span context into headers for tracing propagation
	tracingHeaders := make(map[string]string)
	if err := i.tracer.Inject(span.Context(), i.propagationFormat, getPropagationCarrier(tracingHeaders, i.transport)); err != nil {
		span.LogFields(logFieldEventError, log.String("message", err.Error()))
	} else {
		for k, v := range tracingHeaders {
			req.Meta.Headers = req.Meta.Headers.With(k, v)
		}
	}

	clientStream, err := out.DirectCallStream(ctx, req)
	if err != nil {
		_ = updateSpanWithErrorDetails(span, false, nil, err)
		span.Finish()
		return nil, err
	}

	tracedStream := &tracedClientStream{
		clientStream: clientStream,
		span:         span,
	}

	return wrapTracedClientStream(tracedStream), nil
}

func updateSpanWithErrorDetails(
	span opentracing.Span,
	isApplicationError bool,
	appErrorMeta *transport.ApplicationErrorMeta,
	err error,
) error {
	if err == nil && !isApplicationError {
		return nil
	}
	ext.Error.Set(span, true)
	if status := yarpcerrors.FromError(err); status != nil {
		errCode := status.Code()
		span.SetTag(rpcStatusCodeTag, int(errCode))
		return err
	}
	if isApplicationError {
		span.SetTag(rpcStatusCodeTag, applicationError)

		if appErrorMeta != nil {
			if appErrorMeta.Code != nil {
				span.SetTag(rpcStatusCodeTag, int(*appErrorMeta.Code))
			}
			if appErrorMeta.Name != "" {
				span.SetTag(errorNameTag, appErrorMeta.Name)
			}
		}
		return err
	}
	span.SetTag(rpcStatusCodeTag, int(yarpcerrors.CodeUnknown))
	return err
}

func getPropagationFormat(transport string) opentracing.BuiltinFormat {
	if transport == "tchannel" {
		return opentracing.TextMap
	}
	return opentracing.HTTPHeaders
}

func getPropagationCarrier(headers map[string]string, transport string) PropagationCarrier {
	if transport == "tchannel" {
		return tracing.HeadersCarrier(headers)
	}
	return opentracing.TextMapCarrier(headers)
}

func wrapTracedClientStream(tracedStream *tracedClientStream) *transport.ClientStream {
	wrapped, err := transport.NewClientStream(tracedStream)
	if err != nil {
		// This should not happen, since NewClientStream only fails for the nil streams.
		tracedStream.span.LogFields(logFieldEventError, log.String("message", "Failed to wrap traced client stream"))
		tracedStream.span.Finish()
		return tracedStream.clientStream
	}
	return wrapped
}
