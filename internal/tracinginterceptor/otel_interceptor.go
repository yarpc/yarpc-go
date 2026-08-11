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
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/transport/tchannel/tracing"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

// commonOTelAttributes are the static attributes set on every span, matching
// commonTracingTags on the OpenTracing side.
var commonOTelAttributes = []attribute.KeyValue{
	attribute.String("go.version", runtime.Version()),
	attribute.String("component", tracingComponentName),
}

var (
	_ interceptor.UnaryInbound   = (*OTelInterceptor)(nil)
	_ interceptor.UnaryOutbound  = (*OTelInterceptor)(nil)
	_ interceptor.OnewayInbound  = (*OTelInterceptor)(nil)
	_ interceptor.OnewayOutbound = (*OTelInterceptor)(nil)
	_ interceptor.StreamInbound  = (*OTelInterceptor)(nil)
	_ interceptor.StreamOutbound = (*OTelInterceptor)(nil)
)

// OTelParams defines the parameters for creating the OTelInterceptor.
// TracerProvider and Propagator default to the OpenTelemetry globals when unset.
type OTelParams struct {
	TracerProvider trace.TracerProvider
	Propagator     propagation.TextMapPropagator
	Transport      string
	Logger         *zap.Logger
}

// OTelInterceptor is the OpenTelemetry-native tracing interceptor for all YARPC
// RPC types.
type OTelInterceptor struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	transport  string
	log        *zap.Logger
}

// NewOTel constructs an OTelInterceptor from the provided parameters.
func NewOTel(p OTelParams) *OTelInterceptor {
	i := &OTelInterceptor{
		propagator: p.Propagator,
		transport:  p.Transport,
		log:        p.Logger,
	}
	tracerProvider := p.TracerProvider
	if tracerProvider == nil {
		tracerProvider = otel.GetTracerProvider()
	}
	i.tracer = tracerProvider.Tracer(instrumentationScopeName)
	if i.propagator == nil {
		i.propagator = otel.GetTextMapPropagator()
	}
	if i.log == nil {
		i.log = zap.NewNop()
	}
	return i
}

// Handle implements interceptor.UnaryInbound.
func (i *OTelInterceptor) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	ctx, span := i.startInboundSpan(ctx, req)
	defer span.End()

	err := h.Handle(ctx, req, resw)

	extWriter, ok := resw.(transport.ExtendedResponseWriter)
	if !ok {
		i.log.Debug("ResponseWriter does not implement ExtendedResponseWriter, passing false and nil for app error meta")
		return otelUpdateSpanWithErrorDetails(span, false, nil, err)
	}
	return otelUpdateSpanWithErrorDetails(span, extWriter.IsApplicationError(), extWriter.ApplicationErrorMeta(), err)
}

// Call implements interceptor.UnaryOutbound.
func (i *OTelInterceptor) Call(ctx context.Context, req *transport.Request, out interceptor.UnaryOutboundChain) (*transport.Response, error) {
	ctx, span := i.startOutboundSpan(ctx, req)
	defer span.End()

	res, err := out.Next(ctx, req)
	if res != nil {
		return res, otelUpdateSpanWithErrorDetails(span, res.ApplicationError, res.ApplicationErrorMeta, err)
	}
	return nil, otelUpdateSpanWithErrorDetails(span, false, nil, err)
}

// HandleOneway implements interceptor.OnewayInbound.
func (i *OTelInterceptor) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	ctx, span := i.startInboundSpan(ctx, req)
	defer span.End()

	return otelUpdateSpanWithErrorDetails(span, false, nil, h.HandleOneway(ctx, req))
}

// CallOneway implements interceptor.OnewayOutbound.
func (i *OTelInterceptor) CallOneway(ctx context.Context, req *transport.Request, out interceptor.OnewayOutboundChain) (transport.Ack, error) {
	ctx, span := i.startOutboundSpan(ctx, req)
	defer span.End()

	ack, err := out.Next(ctx, req)
	return ack, otelUpdateSpanWithErrorDetails(span, false, nil, err)
}

// HandleStream implements interceptor.StreamInbound.
func (i *OTelInterceptor) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	ctx, span := i.startInboundSpan(s.Context(), s.Request().Meta.ToRequest())
	defer span.End()

	tracedRaw := &tracedOTelServerStream{
		serverStream: s,
		span:         span,
		ctx:          ctx,
	}
	wrapped, err := transport.NewServerStream(tracedRaw)
	if err != nil {
		return otelUpdateSpanWithErrorDetails(span, false, nil, err)
	}
	return otelUpdateSpanWithErrorDetails(span, false, nil, h.HandleStream(wrapped))
}

// CallStream implements interceptor.StreamOutbound.
func (i *OTelInterceptor) CallStream(ctx context.Context, req *transport.StreamRequest, out interceptor.StreamOutboundChain) (*transport.ClientStream, error) {
	transportReq := req.Meta.ToRequest()
	ctx, span := i.startOutboundSpan(ctx, transportReq)
	req.Meta.Headers = transportReq.Headers

	clientStream, err := out.Next(ctx, req)
	if err != nil {
		_ = otelUpdateSpanWithErrorDetails(span, false, nil, err)
		span.End()
		return nil, err
	}

	return wrapOTelClientStream(&tracedOTelClientStream{clientStream: clientStream, span: span}), nil
}

// startInboundSpan extracts the caller's tracing context from the request
// headers and starts a server-side span.
func (i *OTelInterceptor) startInboundSpan(ctx context.Context, req *transport.Request) (context.Context, trace.Span) {
	ctx = i.propagator.Extract(ctx, getOTelPropagationCarrier(req.Headers.Items(), req.Transport))
	return i.tracer.Start(ctx, req.Procedure,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(spanAttributes(req, req.Transport, req.Caller)...),
	)
}

// startOutboundSpan starts a client-side span and injects its tracing context
// into req.Headers. req is modified in place, matching the OpenTracing
// interceptor.
func (i *OTelInterceptor) startOutboundSpan(ctx context.Context, req *transport.Request) (context.Context, trace.Span) {
	// i.transport is used rather than req.Transport because this is an outbound
	// call made by the interceptor itself, not one arriving off the wire.
	ctx, span := i.tracer.Start(ctx, req.Procedure,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(spanAttributes(req, i.transport, req.Service)...),
	)

	tracingHeaders := make(map[string]string)
	i.propagator.Inject(ctx, getOTelPropagationCarrier(tracingHeaders, i.transport))
	for k, v := range tracingHeaders {
		req.Headers = req.Headers.With(k, v)
	}

	return ctx, span
}

// spanAttributes builds the attribute set shared by inbound and outbound spans.
func spanAttributes(req *transport.Request, transportName, peerService string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 5+len(commonOTelAttributes))
	attrs = append(attrs,
		attribute.String("rpc.caller", req.Caller),
		attribute.String("rpc.service", req.Service),
		attribute.String("rpc.encoding", string(req.Encoding)),
		attribute.String("rpc.transport", transportName),
		semconv.PeerServiceKey.String(peerService),
	)
	return append(attrs, commonOTelAttributes...)
}

// otelUpdateSpanWithErrorDetails records error information on the span and
// returns the original error unchanged.
func otelUpdateSpanWithErrorDetails(
	span trace.Span,
	isApplicationError bool,
	appErrorMeta *transport.ApplicationErrorMeta,
	err error,
) error {
	if err == nil && !isApplicationError {
		return nil
	}
	// The error message is deliberately not recorded on the span. It can carry
	// request data, and the OpenTracing interceptor never emitted it either.
	span.SetStatus(codes.Error, "")

	if status := yarpcerrors.FromError(err); status != nil {
		span.SetAttributes(attribute.Int(rpcStatusCodeTag, int(status.Code())))
		return err
	}
	if isApplicationError {
		// An application error without a code has no numeric equivalent, so the
		// status code attribute is a string here and an int elsewhere. This
		// matches the OpenTracing interceptor.
		span.SetAttributes(attribute.String(rpcStatusCodeTag, applicationError))
		if appErrorMeta != nil {
			if appErrorMeta.Code != nil {
				span.SetAttributes(attribute.Int(rpcStatusCodeTag, int(*appErrorMeta.Code)))
			}
			if appErrorMeta.Name != "" {
				span.SetAttributes(attribute.String(errorNameTag, appErrorMeta.Name))
			}
		}
		return err
	}
	span.SetAttributes(attribute.Int(rpcStatusCodeTag, int(yarpcerrors.CodeUnknown)))
	return err
}

// getOTelPropagationCarrier returns the appropriate OTel TextMapCarrier for
// the given transport. TChannel needs a dedicated carrier to handle the
// "$tracing$" header prefix.
func getOTelPropagationCarrier(headers map[string]string, transportName string) propagation.TextMapCarrier {
	if transportName == "tchannel" {
		return tracing.OTelHeadersCarrier(headers)
	}
	return propagation.MapCarrier(headers)
}
