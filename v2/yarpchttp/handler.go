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
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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

func (h handler) ServeHTTP(w http.ResponseWriter, httpReq *http.Request) {
	responseWriter := newResponseWriter(w)
	service := popHeader(httpReq.Header, ServiceHeader)
	procedure := popHeader(httpReq.Header, ProcedureHeader)
	bothResponseError := popHeader(httpReq.Header, AcceptsBothResponseErrorHeader) == AcceptTrue
	// add response header to echo accepted rpc-service
	responseWriter.WriteSystemHeader(ServiceHeader, service)

	res, resBuf, err := h.callHandler(responseWriter, httpReq, service, procedure)
	if err == nil {
		responseWriter.SetResponse(res, resBuf)
		responseWriter.Close(http.StatusOK)
		return
	}

	status := yarpcerror.FromError(yarpcerror.WrapHandlerError(err, service, procedure))
	responseWriter.SetApplicationError()

	if statusCodeText, marshalErr := status.Code().MarshalText(); marshalErr != nil {
		status = yarpcerror.Newf(yarpcerror.CodeInternal, "error %s had code %v which is unknown", status.Error(), status.Code())
		responseWriter.WriteSystemHeader(ErrorCodeHeader, "internal")
	} else {
		responseWriter.WriteSystemHeader(ErrorCodeHeader, string(statusCodeText))
	}
	if status.Name() != "" {
		responseWriter.WriteSystemHeader(ErrorNameHeader, status.Name())
	}

	if bothResponseError && !h.legacyResponseError {
		// Write the error message as a header AND set the response body. This is
		// intended for returning structured errors (eg via proto.Any) and still
		// getting the server error string.
		responseWriter.WriteSystemHeader(BothResponseErrorHeader, AcceptTrue)
		responseWriter.WriteSystemHeader(ErrorMessageHeader, status.Message())
		responseWriter.SetResponse(res, resBuf)
	} else {
		// Set the error message as the response body (not in a header)
		responseWriter.WriteSystemHeader("Content-Type", "text/plain; charset=utf8")
		responseWriter.SetResponse(res, yarpc.NewBufferString(status.Message()))
	}

	httpStatusCode, ok := _codeToStatusCode[status.Code()]
	if !ok {
		httpStatusCode = http.StatusInternalServerError
	}
	responseWriter.Close(httpStatusCode)
}

func (h handler) callHandler(
	responseWriter *responseWriter,
	httpReq *http.Request,
	service string,
	procedure string,
) (res *yarpc.Response, resBuf *yarpc.Buffer, err error) {
	start := time.Now()
	if httpReq.Method != http.MethodPost {
		return nil, nil, yarpcerror.Newf(yarpcerror.CodeNotFound, "request method was %s but only %s is allowed", httpReq.Method, http.MethodPost)
	}
	req := &yarpc.Request{
		Caller:          popHeader(httpReq.Header, CallerHeader),
		Service:         service,
		Procedure:       procedure,
		Encoding:        yarpc.Encoding(popHeader(httpReq.Header, EncodingHeader)),
		Transport:       transportName,
		ShardKey:        popHeader(httpReq.Header, ShardKeyHeader),
		RoutingKey:      popHeader(httpReq.Header, RoutingKeyHeader),
		RoutingDelegate: popHeader(httpReq.Header, RoutingDelegateHeader),
		Headers:         applicationHeaders.FromHTTPHeaders(httpReq.Header, yarpc.Headers{}),
	}

	reqBuf, err := readCloserToBuffer(httpReq.Body)
	if err != nil {
		return nil, nil, err
	}

	for header := range h.grabHeaders {
		if value := httpReq.Header.Get(header); value != "" {
			req.Headers = req.Headers.With(header, value)
		}
	}
	if err = yarpc.ValidateRequest(req); err != nil {
		return nil, nil, err
	}

	ctx := httpReq.Context()
	ctx, cancel, parseTTLErr := parseTTL(ctx, req, popHeader(httpReq.Header, TTLMSHeader))
	// parseTTLErr != nil is a problem only if the request is unary.
	defer cancel()
	ctx, span := h.createSpan(ctx, httpReq, req, start)

	spec, err := h.router.Choose(ctx, req)
	if err != nil {
		return nil, nil, yarpctracing.UpdateSpanWithErr(span, err)
	}

	if parseTTLErr != nil {
		return nil, nil, parseTTLErr
	}
	if err = yarpc.ValidateRequestContext(ctx); err != nil {
		return nil, nil, err
	}
	switch spec.Type() {
	case yarpc.Unary:
		defer span.Finish()
		res, resBuf, err = yarpctransport.InvokeUnaryHandler(yarpctransport.UnaryInvokeRequest{
			Context:   ctx,
			StartTime: start,
			Request:   req,
			Buffer:    reqBuf,
			Handler:   spec.Unary(),
			Logger:    h.logger,
		})

	default:
		err = yarpcerror.Newf(yarpcerror.CodeUnimplemented, "transport http does not handle %s handlers", spec.Type().String())
	}

	if err != nil {
		return res, resBuf, yarpctracing.UpdateSpanWithErr(span, err)
	}

	if contentType := getContentType(req.Encoding); contentType != "" {
		responseWriter.WriteSystemHeader("Content-Type", contentType)
	}
	return res, resBuf, nil
}

func (h handler) createSpan(ctx context.Context, httpReq *http.Request, req *yarpc.Request, start time.Time) (context.Context, opentracing.Span) {
	// Extract opentracing etc baggage from headers
	// Annotate the inbound context with a trace span
	tracer := h.tracer
	carrier := opentracing.HTTPHeadersCarrier(httpReq.Header)
	parentSpanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, carrier)
	// parentSpanCtx may be nil, ext.RPCServerOption handles a nil parent
	// gracefully.
	tags := opentracing.Tags{
		"rpc.caller":    req.Caller,
		"rpc.service":   req.Service,
		"rpc.encoding":  req.Encoding,
		"rpc.transport": "http",
	}
	for k, v := range yarpctracing.Tags {
		tags[k] = v
	}
	span := tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(start),
		ext.RPCServerOption(parentSpanCtx), // implies ChildOf
		tags,
	)
	ext.PeerService.Set(span, req.Caller)
	ctx = opentracing.ContextWithSpan(ctx, span)
	return ctx, span
}

// responseWriter adapts a http.ResponseWriter into a yarpc.ResponseWriter.
type responseWriter struct {
	w   http.ResponseWriter
	buf *yarpc.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	w.Header().Set(ApplicationStatusHeader, ApplicationSuccessStatus)
	return &responseWriter{w: w}
}

func (rw *responseWriter) WriteSystemHeader(key string, value string) {
	rw.w.Header().Set(key, value)
}

// SetResponse stores the response that will be written when calling `Close(..)`.
func (rw *responseWriter) SetResponse(res *yarpc.Response, resBuf *yarpc.Buffer) {
	if res != nil {
		applicationHeaders.ToHTTPHeaders(res.Headers, rw.w.Header())
	}
	if resBuf != nil {
		rw.buf = resBuf
	}
}

func (rw *responseWriter) SetApplicationError() {
	rw.w.Header().Set(ApplicationStatusHeader, ApplicationErrorStatus)
}

func (rw *responseWriter) Close(httpStatusCode int) {
	// We MUST write the HTTP status code prior to writing the response body,
	// since writing to the response body is an implicit 200.
	rw.w.WriteHeader(httpStatusCode)
	if rw.buf != nil {
		// TODO: log this error?
		_, _ = rw.buf.WriteTo(rw.w)
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
