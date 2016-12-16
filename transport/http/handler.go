// Copyright (c) 2016 Uber Technologies, Inc.
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

package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func popHeader(h http.Header, n string) string {
	v := h.Get(n)
	h.Del(n)
	return v
}

// handler adapts a transport.Handler into a handler for net/http.
type handler struct {
	registry transport.Registry
	tracer   opentracing.Tracer
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	defer req.Body.Close()
	if req.Method != "POST" {
		http.NotFound(w, req)
		return
	}

	service := req.Header.Get(ServiceHeader)
	procedure := req.Header.Get(ProcedureHeader)

	err := h.callHandler(w, req, start)
	if err == nil {
		return
	}

	err = errors.AsHandlerError(service, procedure, err)
	status := http.StatusInternalServerError
	if transport.IsBadRequestError(err) {
		status = http.StatusBadRequest
	} else if transport.IsTimeoutError(err) {
		status = http.StatusGatewayTimeout
	}
	http.Error(w, err.Error(), status)
}

func (h handler) callHandler(w http.ResponseWriter, req *http.Request, start time.Time) error {
	treq := &transport.Request{
		Caller:    popHeader(req.Header, CallerHeader),
		Service:   popHeader(req.Header, ServiceHeader),
		Procedure: popHeader(req.Header, ProcedureHeader),
		Encoding:  transport.Encoding(popHeader(req.Header, EncodingHeader)),
		Headers:   applicationHeaders.FromHTTPHeaders(req.Header, transport.Headers{}),
		Body:      req.Body,
	}

	ctx := req.Context()

	v := request.Validator{Request: treq}
	ctx, cancel := v.ParseTTL(ctx, popHeader(req.Header, TTLMSHeader))
	defer cancel()

	ctx, span := h.createSpan(ctx, req, treq, start)

	err := v.ValidateCommon(ctx)
	if err != nil {
		return err
	}

	spec, err := h.registry.Choose(ctx, treq)
	if err != nil {
		return updateSpanWithErr(span, err)
	}

	switch spec.Type() {
	case transport.Unary:
		defer span.Finish()

		if err := v.ValidateUnary(ctx); err != nil {
			return err
		}
		err = transport.DispatchUnaryHandler(ctx, spec.Unary(), start, treq, newResponseWriter(w))

	case transport.Oneway:
		if err := v.ValidateOneway(ctx); err != nil {
			return err
		}
		err = handleOnewayRequest(span, treq, spec.Oneway())

	default:
		err = errors.UnsupportedTypeError{Transport: "HTTP", Type: string(spec.Type())}
	}

	return updateSpanWithErr(span, err)
}

func handleOnewayRequest(
	span opentracing.Span,
	treq *transport.Request,
	onewayHandler transport.OnewayHandler,
) error {
	// we will lose access to the body unless we read all the bytes before
	// returning from the request
	var buff bytes.Buffer
	if _, err := io.Copy(&buff, treq.Body); err != nil {
		return err
	}
	treq.Body = &buff

	// create a new context for oneway requests since the HTTP handler cancels
	// http.Request's context when ServeHTTP returns
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	go func() {
		// ensure the span lasts for length of the handler in case of errors
		defer span.Finish()

		err := transport.DispatchOnewayHandler(ctx, onewayHandler, treq)
		updateSpanWithErr(span, err)
	}()
	return nil
}

func updateSpanWithErr(span opentracing.Span, err error) error {
	if err != nil {
		span.SetTag("error", true)
		span.LogEvent(err.Error())
	}

	return err
}

func (h handler) createSpan(ctx context.Context, req *http.Request, treq *transport.Request, start time.Time) (context.Context, opentracing.Span) {
	// Extract opentracing etc baggage from headers
	// Annotate the inbound context with a trace span
	tracer := h.tracer
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	parentSpanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)
	// parentSpanCtx may be nil, ext.RPCServerOption handles a nil parent
	// gracefully.
	span := tracer.StartSpan(
		treq.Procedure,
		opentracing.StartTime(start),
		opentracing.Tags{
			"rpc.caller":    treq.Caller,
			"rpc.service":   treq.Service,
			"rpc.encoding":  treq.Encoding,
			"rpc.transport": "http",
		},
		ext.RPCServerOption(parentSpanCtx), // implies ChildOf
	)
	ext.PeerService.Set(span, treq.Caller)
	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// responseWriter adapts a http.ResponseWriter into a transport.ResponseWriter.
type responseWriter struct {
	w http.ResponseWriter
}

func newResponseWriter(w http.ResponseWriter) responseWriter {
	return responseWriter{w: w}
}

func (rw responseWriter) Write(s []byte) (int, error) {
	return rw.w.Write(s)
}

func (rw responseWriter) AddHeaders(h transport.Headers) {
	applicationHeaders.ToHTTPHeaders(h, rw.w.Header())
}

func (responseWriter) SetApplicationError() {
	// Nothing to do.
}
