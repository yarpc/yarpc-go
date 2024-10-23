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
	"go.uber.org/yarpc/transport/tchannel/tracing"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/yarpcerrors"
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
}

// Interceptor is the tracing interceptor for all RPC types.
// It handles both tracing observability and context propagation using OpenTracing APIs.
type Interceptor struct {
	tracer            opentracing.Tracer
	transport         string
	propagationFormat opentracing.BuiltinFormat
}

// PropagationCarrier is an interface to combine both reader and writer interface
type PropagationCarrier interface {
	opentracing.TextMapReader
	opentracing.TextMapWriter
}

// writer wraps a transport.ResponseWriter to capture additional information for tracing.
type writer struct {
	transport.ResponseWriter

	isApplicationError   bool
	applicationErrorMeta *transport.ApplicationErrorMeta
}

var _writerPool = sync.Pool{New: func() interface{} {
	return &writer{}
}}

func newWriter(rw transport.ResponseWriter) *writer {
	w := _writerPool.Get().(*writer)
	*w = writer{ResponseWriter: rw} // reset
	return w
}

func (w *writer) SetApplicationError() {
	w.isApplicationError = true
	w.ResponseWriter.SetApplicationError()
}

func (w *writer) SetApplicationErrorMeta(applicationErrorMeta *transport.ApplicationErrorMeta) {
	if applicationErrorMeta == nil {
		return
	}

	w.applicationErrorMeta = applicationErrorMeta
	if appErrMetaSetter, ok := w.ResponseWriter.(transport.ApplicationErrorMetaSetter); ok {
		appErrMetaSetter.SetApplicationErrorMeta(applicationErrorMeta)
	}
}

func (w *writer) free() {
	_writerPool.Put(w)
}

// New constructs a tracing interceptor with the provided parameter.
func New(p Params) *Interceptor {
	m := &Interceptor{
		tracer:            p.Tracer,
		transport:         p.Transport,
		propagationFormat: getPropagationFormat(p.Transport),
	}
	if m.tracer == nil {
		m.tracer = opentracing.GlobalTracer()
	}
	return m
}

// Handle implements interceptor.UnaryInbound
func (m *Interceptor) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	wrappedWriter := newWriter(resw)
	defer wrappedWriter.free()

	parentSpanCtx, _ := m.tracer.Extract(m.propagationFormat, GetPropagationCarrier(req.Headers.Items(), req.Transport))
	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            m.tracer,
		TransportName:     req.Transport,
		StartTime:         time.Now(),
		ExtraTags:         commonTracingTags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	err := h.Handle(ctx, req, wrappedWriter)
	return updateSpanWithErrorDetails(span, nil, wrappedWriter.isApplicationError, wrappedWriter.applicationErrorMeta, err)
}

// Call implements interceptor.UnaryOutbound
func (m *Interceptor) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        m.tracer,
		TransportName: m.transport,
		StartTime:     time.Now(),
		ExtraTags:     commonTracingTags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	tracingHeaders := make(map[string]string)
	if err := m.tracer.Inject(span.Context(), m.propagationFormat, GetPropagationCarrier(tracingHeaders, m.transport)); err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.String("event", "error"), log.String("message", err.Error()))
	} else {
		for k, v := range tracingHeaders {
			req.Headers = req.Headers.With(k, v)
		}
	}

	res, err := out.Call(ctx, req)
	return res, updateSpanWithErrorDetails(span, res, false, nil, err)
}

// HandleOneway implements interceptor.OnewayInbound
func (m *Interceptor) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	// TODO: implement
	panic("implement me")
}

// CallOneway implements interceptor.OnewayOutbound
func (m *Interceptor) CallOneway(ctx context.Context, req *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	// TODO: implement
	panic("implement me")
}

// HandleStream implements interceptor.StreamInbound
func (m *Interceptor) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	// TODO: implement
	panic("implement me")
}

// CallStream implements interceptor.StreamOutbound
func (m *Interceptor) CallStream(ctx context.Context, req *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	// TODO: implement
	panic("implement me")
}

func updateSpanWithErrorDetails(
	span opentracing.Span,
	res *transport.Response,
	isApplicationError bool,
	appErrorMeta *transport.ApplicationErrorMeta,
	err error,
) error {
	if err == nil && (res == nil || !isApplicationError) {
		return err
	}
	ext.Error.Set(span, true)
	if status := yarpcerrors.FromError(err); status != nil {
		errCode := status.Code()
		span.SetTag("rpc.yarpc.status_code", int(errCode))
		span.SetTag("error.type", errCode.String())
		return err
	}
	if res != nil && res.ApplicationError {
		span.SetTag("error.type", "application_error")

		if res.ApplicationErrorMeta != nil {
			meta := res.ApplicationErrorMeta
			if meta.Code != nil {
				span.SetTag("application_error_code", int(*meta.Code))
			}
			if meta.Details != "" {
				span.SetTag("application_error_name", meta.Name)
			}
		}
		return err
	}
	if isApplicationError {
		span.SetTag("error.type", "application_error")

		if appErrorMeta != nil {
			if appErrorMeta.Code != nil {
				span.SetTag("application_error_code", int(*appErrorMeta.Code))
			}
			if appErrorMeta.Details != "" {
				span.SetTag("application_error_name", appErrorMeta.Name)
			}
		}
		return err
	}

	span.SetTag("error.type", "unknown_internal_yarpc")
	return err
}

// getPropagationFormat returns the opentracing propagation depends on transport.
// For TChannel, the format is opentracing.TextMap
// For HTTP and gRPC, the format is opentracing.HTTPHeaders
func getPropagationFormat(transport string) opentracing.BuiltinFormat {
	if transport == "tchannel" {
		return opentracing.TextMap
	}
	return opentracing.HTTPHeaders
}

// GetPropagationCarrier get the propagation carrier depends on the transport.
// The carrier is used for accessing the transport headers.
// For TChannel, a special carrier is used. For details, see comments of HeadersCarrier
func GetPropagationCarrier(headers map[string]string, transport string) PropagationCarrier {
	if transport == "tchannel" {
		return tracing.HeadersCarrier(headers)
	}
	return opentracing.TextMapCarrier(headers)
}
