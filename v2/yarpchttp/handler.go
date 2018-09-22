// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpchttp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/internal/bufferpool"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpctracing"
	"go.uber.org/yarpc/v2/yarpctransport"
	"go.uber.org/zap"
)

func popHeader(h http.Header, n string) string {
	v := h.Get(n)
	h.Del(n)
	return v
}

// handler adapts a yarpc.Handler into a handler for net/http.
type handler struct {
	router              yarpc.Router
	tracer              opentracing.Tracer
	grabHeaders         map[string]struct{}
	legacyResponseError bool
	logger              *zap.Logger
}

func (h handler) ServeHTTP(w http.ResponseWriter, hRequest *http.Request) {
	responseWriter := newResponseWriter(w)
	service := popHeader(hRequest.Header, ServiceHeader)
	procedure := popHeader(hRequest.Header, ProcedureHeader)
	bothResponseError := popHeader(hRequest.Header, AcceptsBothResponseErrorHeader) == AcceptTrue
	// add response header to echo accepted rpc-service
	responseWriter.AddSystemHeader(ServiceHeader, service)

	err := h.callHandler(responseWriter, hRequest, service, procedure)
	status := yarpcerror.FromError(yarpcerror.WrapHandlerError(err, service, procedure))
	if status == nil {
		responseWriter.Close(http.StatusOK)
		return
	}
	if statusCodeText, marshalErr := status.Code().MarshalText(); marshalErr != nil {
		status = yarpcerror.Newf(yarpcerror.CodeInternal, "error %s had code %v which is unknown", status.Error(), status.Code())
		responseWriter.AddSystemHeader(ErrorCodeHeader, "internal")
	} else {
		responseWriter.AddSystemHeader(ErrorCodeHeader, string(statusCodeText))
	}
	if status.Name() != "" {
		responseWriter.AddSystemHeader(ErrorNameHeader, status.Name())
	}
	if bothResponseError && !h.legacyResponseError {
		responseWriter.AddSystemHeader(BothResponseErrorHeader, AcceptTrue)
		responseWriter.AddSystemHeader(ErrorMessageHeader, status.Message())
	} else {
		responseWriter.ResetBuffer()
		_, _ = fmt.Fprintln(responseWriter, status.Message())
		responseWriter.AddSystemHeader("Content-Type", "text/plain; charset=utf8")
	}
	httpStatusCode, ok := _codeToStatusCode[status.Code()]
	if !ok {
		httpStatusCode = http.StatusInternalServerError
	}
	responseWriter.Close(httpStatusCode)
}

func (h handler) callHandler(responseWriter *responseWriter, hRequest *http.Request, service string, procedure string) (retErr error) {
	start := time.Now()
	defer hRequest.Body.Close()
	if hRequest.Method != http.MethodPost {
		return yarpcerror.Newf(yarpcerror.CodeNotFound, "request method was %s but only %s is allowed", hRequest.Method, http.MethodPost)
	}
	yRequest := &yarpc.Request{
		Caller:          popHeader(hRequest.Header, CallerHeader),
		Service:         service,
		Procedure:       procedure,
		Encoding:        yarpc.Encoding(popHeader(hRequest.Header, EncodingHeader)),
		Transport:       transportName,
		ShardKey:        popHeader(hRequest.Header, ShardKeyHeader),
		RoutingKey:      popHeader(hRequest.Header, RoutingKeyHeader),
		RoutingDelegate: popHeader(hRequest.Header, RoutingDelegateHeader),
		Headers:         applicationHeaders.FromHTTPHeaders(hRequest.Header, yarpc.Headers{}),
	}

	body, err := ioutil.ReadAll(hRequest.Body)
	if err != nil {
		return err
	}
	requestBuf := yarpc.NewBufferBytes(body)

	for header := range h.grabHeaders {
		if value := hRequest.Header.Get(header); value != "" {
			yRequest.Headers = yRequest.Headers.With(header, value)
		}
	}
	if err := yarpc.ValidateRequest(yRequest); err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			if contentType := getContentType(yRequest.Encoding); contentType != "" {
				responseWriter.AddSystemHeader("Content-Type", contentType)
			}
		}
	}()

	ctx := hRequest.Context()
	ctx, cancel, parseTTLErr := parseTTL(ctx, yRequest, popHeader(hRequest.Header, TTLMSHeader))
	// parseTTLErr != nil is a problem only if the request is unary.
	defer cancel()
	ctx, span := h.createSpan(ctx, hRequest, yRequest, start)

	spec, err := h.router.Choose(ctx, yRequest)
	if err != nil {
		return updateSpanWithErr(span, err)
	}

	if parseTTLErr != nil {
		return parseTTLErr
	}
	if err := yarpc.ValidateRequestContext(ctx); err != nil {
		return err
	}
	switch spec.Type() {
	case yarpc.Unary:
		defer span.Finish()
		_, _, err = yarpctransport.InvokeUnaryHandler(yarpctransport.UnaryInvokeRequest{
			Context:   ctx,
			StartTime: start,
			Request:   yRequest,
			Buffer:    requestBuf,
			Handler:   spec.Unary(),
			Logger:    h.logger,
		})

	default:
		err = yarpcerror.Newf(yarpcerror.CodeUnimplemented, "transport http does not handle %s handlers", spec.Type().String())
	}

	return updateSpanWithErr(span, err)
}

func updateSpanWithErr(span opentracing.Span, err error) error {
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(opentracinglog.String("event", err.Error()))
	}
	return err
}

func (h handler) createSpan(ctx context.Context, hRequest *http.Request, yRequest *yarpc.Request, start time.Time) (context.Context, opentracing.Span) {
	// Extract opentracing etc baggage from headers
	// Annotate the inbound context with a trace span
	tracer := h.tracer
	carrier := opentracing.HTTPHeadersCarrier(hRequest.Header)
	parentSpanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)
	// parentSpanCtx may be nil, ext.RPCServerOption handles a nil parent
	// gracefully.
	tags := opentracing.Tags{
		"rpc.caller":    yRequest.Caller,
		"rpc.service":   yRequest.Service,
		"rpc.encoding":  yRequest.Encoding,
		"rpc.transport": "http",
	}
	for k, v := range yarpctracing.Tags {
		tags[k] = v
	}
	span := tracer.StartSpan(
		yRequest.Procedure,
		opentracing.StartTime(start),
		ext.RPCServerOption(parentSpanCtx), // implies ChildOf
		tags,
	)
	ext.PeerService.Set(span, yRequest.Caller)
	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// responseWriter adapts a http.ResponseWriter into a yarpc.ResponseWriter.
type responseWriter struct {
	w      http.ResponseWriter
	buffer *bufferpool.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	w.Header().Set(ApplicationStatusHeader, ApplicationSuccessStatus)
	return &responseWriter{w: w}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.buffer == nil {
		rw.buffer = bufferpool.Get()
	}
	return rw.buffer.Write(s)
}

func (rw *responseWriter) AddHeaders(h yarpc.Headers) {
	applicationHeaders.ToHTTPHeaders(h, rw.w.Header())
}

func (rw *responseWriter) SetApplicationError() {
	rw.w.Header().Set(ApplicationStatusHeader, ApplicationErrorStatus)
}

func (rw *responseWriter) AddSystemHeader(key string, value string) {
	rw.w.Header().Set(key, value)
}

func (rw *responseWriter) ResetBuffer() {
	if rw.buffer != nil {
		rw.buffer.Reset()
	}
}

func (rw *responseWriter) Close(httpStatusCode int) {
	rw.w.WriteHeader(httpStatusCode)
	if rw.buffer != nil {
		// TODO: what to do with error?
		_, _ = rw.buffer.WriteTo(rw.w)
		bufferpool.Put(rw.buffer)
	}
}

func getContentType(encoding yarpc.Encoding) string {
	switch encoding {
	case "json":
		return "application/json"
	case "raw":
		return "application/octet-stream"
	case "thrift":
		return "application/vnd.apache.thrift.binary"
	case "proto":
		return "application/x-protobuf"
	default:
		return ""
	}
}
