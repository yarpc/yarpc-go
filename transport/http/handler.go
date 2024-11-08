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

package http

import (
	"bytes"
	"context"
	"fmt"
	"go.uber.org/yarpc/api/middleware"
	"net/http"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/internal/iopool"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

func popHeader(h http.Header, n string) string {
	v := h.Get(n)
	h.Del(n)
	return v
}

// handler adapts a transport.Handler into a handler for net/http.
type handler struct {
	router            transport.Router
	tracer            opentracing.Tracer
	grabHeaders       map[string]struct{}
	bothResponseError bool
	logger            *zap.Logger
	transport         *Transport
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	responseWriter := newResponseWriter(w)
	service := popHeader(req.Header, ServiceHeader)
	procedure := popHeader(req.Header, ProcedureHeader)
	bothResponseError := popHeader(req.Header, AcceptsBothResponseErrorHeader) == AcceptTrue
	// add response header to echo accepted rpc-service
	responseWriter.AddSystemHeader(ServiceHeader, service)
	status := yarpcerrors.FromError(errors.WrapHandlerError(h.callHandler(responseWriter, req, service, procedure), service, procedure))
	if status == nil {
		responseWriter.Close(http.StatusOK)
		return
	}
	if statusCodeText, marshalErr := status.Code().MarshalText(); marshalErr != nil {
		status = yarpcerrors.Newf(yarpcerrors.CodeInternal, "error %s had code %v which is unknown", status.Error(), status.Code())
		responseWriter.AddSystemHeader(ErrorCodeHeader, "internal")
	} else {
		responseWriter.AddSystemHeader(ErrorCodeHeader, string(statusCodeText))
	}
	if status.Name() != "" {
		responseWriter.AddSystemHeader(ErrorNameHeader, status.Name())
	}
	if bothResponseError && h.bothResponseError {
		responseWriter.AddSystemHeader(BothResponseErrorHeader, AcceptTrue)
		responseWriter.AddSystemHeader(ErrorMessageHeader, status.Message())
		if details := status.Details(); details != nil {
			responseWriter.AddSystemHeader(ErrorDetailsHeader, string(details))
			responseWriter.ResetBuffer()
			_, _ = responseWriter.Write(details)
		}
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

func (h handler) callHandler(responseWriter *responseWriter, req *http.Request, service string, procedure string) (retErr error) {
	start := time.Now()
	defer req.Body.Close()
	if req.Method != http.MethodPost {
		return yarpcerrors.Newf(yarpcerrors.CodeNotFound, "request method was %s but only %s is allowed", req.Method, http.MethodPost)
	}

	treq := &transport.Request{
		Caller:          popHeader(req.Header, CallerHeader),
		Service:         service,
		Procedure:       procedure,
		Encoding:        transport.Encoding(popHeader(req.Header, EncodingHeader)),
		Transport:       TransportName,
		ShardKey:        popHeader(req.Header, ShardKeyHeader),
		RoutingKey:      popHeader(req.Header, RoutingKeyHeader),
		RoutingDelegate: popHeader(req.Header, RoutingDelegateHeader),
		CallerProcedure: popHeader(req.Header, CallerProcedureHeader),
		Headers:         applicationHeaders.FromHTTPHeaders(req.Header, transport.Headers{}),
		Body:            req.Body,
		BodySize:        int(req.ContentLength),
	}
	for header := range h.grabHeaders {
		if value := req.Header.Get(header); value != "" {
			treq.Headers = treq.Headers.With(header, value)
		}
	}
	if err := transport.ValidateRequest(treq); err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			if contentType := getContentType(treq.Encoding); contentType != "" {
				responseWriter.AddSystemHeader("Content-Type", contentType)
			}
		}
	}()

	ctx := req.Context()
	ctx, cancel, parseTTLErr := parseTTL(ctx, treq, popHeader(req.Header, TTLMSHeader))
	// parseTTLErr != nil is a problem only if the request is unary.
	defer cancel()

	spec, err := h.router.Choose(ctx, treq)
	if err != nil {
		return err
	}

	if parseTTLErr != nil {
		return parseTTLErr
	}
	if err := transport.ValidateRequestContext(ctx); err != nil {
		return err
	}
	switch spec.Type() {
	case transport.Unary:
		err = transport.InvokeUnaryHandler(transport.UnaryInvokeRequest{
			Context:        ctx,
			StartTime:      start,
			Request:        treq,
			Handler:        middleware.ApplyUnaryInbound(spec.Unary(), h.transport.unaryInboundInterceptor),
			ResponseWriter: responseWriter,
			Logger:         h.logger,
		})

	case transport.Oneway:
		err = handleOnewayRequest(nil, treq, middleware.ApplyOnewayInbound(spec.Oneway(), h.transport.onewayInboundInterceptor), h.logger)

	default:
		err = yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "transport http does not handle %s handlers", spec.Type().String())
	}

	return err
}

func handleOnewayRequest(
	span opentracing.Span,
	treq *transport.Request,
	onewayHandler transport.OnewayHandler,
	logger *zap.Logger,
) error {
	// we will lose access to the body unless we read all the bytes before
	// returning from the request
	var buff bytes.Buffer
	if _, err := iopool.Copy(&buff, treq.Body); err != nil {
		return err
	}
	treq.Body = &buff

	// create a new context for oneway requests since the HTTP handler cancels
	// http.Request's context when ServeHTTP returns
	ctx := context.Background()

	go func() {
		_ = transport.InvokeOnewayHandler(transport.OnewayInvokeRequest{
			Context: ctx,
			Request: treq,
			Handler: onewayHandler,
			Logger:  logger,
		})
	}()
	return nil
}

var (
	_ transport.ExtendedResponseWriter     = (*responseWriter)(nil)
	_ transport.ApplicationErrorMetaSetter = (*responseWriter)(nil)
)

// responseWriter adapts a http.ResponseWriter into a transport.ResponseWriter.
type responseWriter struct {
	w                  http.ResponseWriter
	buffer             *bufferpool.Buffer
	isApplicationError bool
	appErrorMeta       *transport.ApplicationErrorMeta
	responseSize       int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	w.Header().Set(ApplicationStatusHeader, ApplicationSuccessStatus)
	return &responseWriter{w: w}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.buffer == nil {
		rw.buffer = bufferpool.Get()
	}
	n, err := rw.buffer.Write(s)
	rw.responseSize += n
	return n, err
}

func (rw *responseWriter) ResponseSize() int {
	return rw.responseSize
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	applicationHeaders.ToHTTPHeaders(h, rw.w.Header())
}

func (rw *responseWriter) SetApplicationError() {
	rw.isApplicationError = true
	rw.w.Header().Set(ApplicationStatusHeader, ApplicationErrorStatus)
}

func (rw *responseWriter) SetApplicationErrorMeta(meta *transport.ApplicationErrorMeta) {
	if meta == nil {
		return
	}
	rw.appErrorMeta = meta
	if meta.Code != nil {
		rw.w.Header().Set(_applicationErrorCodeHeader, strconv.Itoa(int(*meta.Code)))
	}
	if meta.Name != "" {
		rw.w.Header().Set(_applicationErrorNameHeader, meta.Name)
	}
	if meta.Details != "" {
		rw.w.Header().Set(_applicationErrorDetailsHeader, truncateAppErrDetails(meta.Details))
	}
}

func (rw *responseWriter) IsApplicationError() bool {
	return rw.isApplicationError
}

func (rw *responseWriter) ApplicationErrorMeta() *transport.ApplicationErrorMeta {
	return rw.appErrorMeta
}

func truncateAppErrDetails(val string) string {
	if len(val) <= _maxAppErrDetailsHeaderLen {
		return val
	}
	stripIndex := _maxAppErrDetailsHeaderLen - len(_truncatedHeaderMessage)
	return val[:stripIndex] + _truncatedHeaderMessage
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

func getContentType(encoding transport.Encoding) string {
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
