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
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

const _tchannelTracingKeyPrefix = "$tracing$"

// otelTChannelCarrier wraps a map[string]string as an OTel propagation.TextMapCarrier
// for TChannel. TChannel stores tracing headers with a "$tracing$" prefix; this
// carrier strips the prefix on reads and adds it on writes so the OTel propagator
// sees undecorated header names.
type otelTChannelCarrier map[string]string

var _ propagation.TextMapCarrier = otelTChannelCarrier{}

func (c otelTChannelCarrier) Get(key string) string {
	return c[_tchannelTracingKeyPrefix+key]
}

func (c otelTChannelCarrier) Set(key, value string) {
	c[_tchannelTracingKeyPrefix+key] = value
}

func (c otelTChannelCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		if strings.HasPrefix(k, _tchannelTracingKeyPrefix) {
			keys = append(keys, k[len(_tchannelTracingKeyPrefix):])
		}
	}
	return keys
}

// otelCarrier returns the appropriate OTel TextMapCarrier for the given transport.
// TChannel requires a custom carrier to handle the "$tracing$" header prefix;
// all other transports use the SDK-provided propagation.MapCarrier.
func otelCarrier(headers map[string]string, transportName string) propagation.TextMapCarrier {
	if transportName == "tchannel" {
		return otelTChannelCarrier(headers)
	}
	return propagation.MapCarrier(headers)
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
// TracerProvider and Propagator default to the OTel globals when unset.
type OTelParams struct {
	TracerProvider trace.TracerProvider
	Propagator     propagation.TextMapPropagator
	Transport      string
	Logger         *zap.Logger
}

// OTelInterceptor is the OTel-native tracing interceptor for all YARPC RPC types.
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
	i.tracer = tracerProvider.Tracer("go.uber.org/yarpc")
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
	ctx, span := i.startInboundSpan(ctx, req.Headers.Items(), req.Transport, req.Service, req.Procedure)
	defer span.End()

	err := h.Handle(ctx, req, resw)

	extWriter, ok := resw.(transport.ExtendedResponseWriter)
	if !ok {
		return otelUpdateSpan(span, false, nil, err)
	}
	return otelUpdateSpan(span, extWriter.IsApplicationError(), extWriter.ApplicationErrorMeta(), err)
}

// Call implements interceptor.UnaryOutbound.
func (i *OTelInterceptor) Call(ctx context.Context, req *transport.Request, out interceptor.UnaryOutboundChain) (*transport.Response, error) {
	ctx, span, req := i.startOutboundSpan(ctx, req)
	defer span.End()

	res, err := out.Next(ctx, req)
	if res != nil {
		return res, otelUpdateSpan(span, res.ApplicationError, res.ApplicationErrorMeta, err)
	}
	return nil, otelUpdateSpan(span, false, nil, err)
}

// HandleOneway implements interceptor.OnewayInbound.
func (i *OTelInterceptor) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	ctx, span := i.startInboundSpan(ctx, req.Headers.Items(), req.Transport, req.Service, req.Procedure)
	defer span.End()

	return otelUpdateSpan(span, false, nil, h.HandleOneway(ctx, req))
}

// CallOneway implements interceptor.OnewayOutbound.
func (i *OTelInterceptor) CallOneway(ctx context.Context, req *transport.Request, out interceptor.OnewayOutboundChain) (transport.Ack, error) {
	ctx, span, req := i.startOutboundSpan(ctx, req)
	defer span.End()

	ack, err := out.Next(ctx, req)
	return ack, otelUpdateSpan(span, false, nil, err)
}

// HandleStream implements interceptor.StreamInbound.
func (i *OTelInterceptor) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	req := s.Request()
	ctx, span := i.startInboundSpan(s.Context(), req.Meta.Headers.Items(), req.Meta.Transport, req.Meta.Service, req.Meta.Procedure)
	defer span.End()

	tracedRaw := &tracedOTelServerStream{serverStream: s, ctx: ctx}
	wrapped, err := transport.NewServerStream(tracedRaw)
	if err != nil {
		i.log.Debug("failed to wrap OTel traced server stream", zap.Error(err))
		return otelUpdateSpan(span, false, nil, h.HandleStream(s))
	}
	return otelUpdateSpan(span, false, nil, h.HandleStream(wrapped))
}

// CallStream implements interceptor.StreamOutbound.
func (i *OTelInterceptor) CallStream(ctx context.Context, req *transport.StreamRequest, out interceptor.StreamOutboundChain) (*transport.ClientStream, error) {
	transportReq := &transport.Request{
		Caller:    req.Meta.Caller,
		Service:   req.Meta.Service,
		Procedure: req.Meta.Procedure,
		Headers:   req.Meta.Headers,
		Transport: req.Meta.Transport,
	}
	ctx, span, updatedReq := i.startOutboundSpan(ctx, transportReq)
	req.Meta.Headers = updatedReq.Headers

	clientStream, err := out.Next(ctx, req)
	if err != nil {
		_ = otelUpdateSpan(span, false, nil, err)
		span.End()
		return nil, err
	}

	return wrapOTelClientStream(&tracedOTelClientStream{clientStream: clientStream, span: span})
}

// startInboundSpan extracts tracing context from the request headers and starts
// a server-side span.
func (i *OTelInterceptor) startInboundSpan(ctx context.Context, headers map[string]string, transportName, service, procedure string) (context.Context, trace.Span) {
	ctx = i.propagator.Extract(ctx, otelCarrier(headers, transportName))
	ctx, span := i.tracer.Start(ctx, procedure,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.RPCSystemKey.String("yarpc"),
			semconv.RPCServiceKey.String(service),
			semconv.RPCMethodKey.String(procedure),
		),
	)
	return ctx, span
}

// startOutboundSpan starts a client-side span and injects the tracing context
// into the request headers. Returns updated context, span, and a shallow copy
// of the request with tracing headers added.
func (i *OTelInterceptor) startOutboundSpan(ctx context.Context, req *transport.Request) (context.Context, trace.Span, *transport.Request) {
	ctx, span := i.tracer.Start(ctx, req.Procedure,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.RPCSystemKey.String("yarpc"),
			semconv.RPCServiceKey.String(req.Service),
			semconv.RPCMethodKey.String(req.Procedure),
		),
	)

	tracingHeaders := make(map[string]string)
	i.propagator.Inject(ctx, otelCarrier(tracingHeaders, i.transport))
	for k, v := range tracingHeaders {
		req.Headers = req.Headers.With(k, v)
	}

	return ctx, span, req
}

// otelUpdateSpan records error information on the span and returns the original error.
func otelUpdateSpan(span trace.Span, isApplicationError bool, appErrorMeta *transport.ApplicationErrorMeta, err error) error {
	if err == nil && !isApplicationError {
		return nil
	}

	span.SetStatus(codes.Error, "")
	if err != nil {
		span.RecordError(err)
		if status := yarpcerrors.FromError(err); status != nil {
			span.SetAttributes(semconv.RPCGRPCStatusCodeKey.Int(int(status.Code())))
		}
	}
	if isApplicationError {
		if appErrorMeta != nil {
			if appErrorMeta.Code != nil {
				span.SetAttributes(semconv.RPCGRPCStatusCodeKey.Int(int(*appErrorMeta.Code)))
			}
			if appErrorMeta.Name != "" {
				span.SetAttributes(semconv.RPCMethodKey.String(appErrorMeta.Name))
			}
		}
	}
	return err
}

// tracedOTelServerStream wraps a transport.ServerStream with an enriched context.
type tracedOTelServerStream struct {
	serverStream *transport.ServerStream
	ctx          context.Context
}

func (s *tracedOTelServerStream) Context() context.Context          { return s.ctx }
func (s *tracedOTelServerStream) Request() *transport.StreamRequest { return s.serverStream.Request() }
func (s *tracedOTelServerStream) SendMessage(ctx context.Context, m *transport.StreamMessage) error {
	return s.serverStream.SendMessage(ctx, m)
}
func (s *tracedOTelServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return s.serverStream.ReceiveMessage(ctx)
}

// tracedOTelClientStream wraps a transport.ClientStream and ends the span on close.
type tracedOTelClientStream struct {
	clientStream *transport.ClientStream
	span         trace.Span
}

func (s *tracedOTelClientStream) Context() context.Context          { return s.clientStream.Context() }
func (s *tracedOTelClientStream) Request() *transport.StreamRequest { return s.clientStream.Request() }
func (s *tracedOTelClientStream) SendMessage(ctx context.Context, m *transport.StreamMessage) error {
	return s.clientStream.SendMessage(ctx, m)
}
func (s *tracedOTelClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := s.clientStream.ReceiveMessage(ctx)
	if err != nil {
		_ = otelUpdateSpan(s.span, false, nil, err)
		s.span.End()
	}
	return msg, err
}
func (s *tracedOTelClientStream) Close(ctx context.Context) error {
	err := s.clientStream.Close(ctx)
	_ = otelUpdateSpan(s.span, false, nil, err)
	s.span.End()
	return err
}

func wrapOTelClientStream(s *tracedOTelClientStream) (*transport.ClientStream, error) {
	wrapped, err := transport.NewClientStream(s)
	if err != nil {
		s.span.End()
		return s.clientStream, nil
	}
	return wrapped, nil
}
