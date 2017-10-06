// Copyright (c) 2017 Uber Technologies, Inc.
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
	"io"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/internal/iopool"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/yarpcerrors"
)

func popHeader(h http.Header, n string) string {
	v := h.Get(n)
	h.Del(n)
	return v
}

// handler adapts a transport.Handler into a handler for net/http.
type handler struct {
	router      transport.Router
	tracer      opentracing.Tracer
	grabHeaders map[string]struct{}
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Should we have Content Type/Encoding on the transport.Response?
	resp, contentType, err := h.call(req)

	// We set this on every response, might be able to remove it for pure
	// errors, but, it's technically a backwards breaking change.
	w.Header().Set(ApplicationStatusHeader, ApplicationSuccessStatus)

	// In non-application errors, we serialize the error on the wire.
	if err != nil {
		status := yarpcerrors.FromError(err)
		writeErrorHeaders(w, status)
		w.Header().Set("Content-Type", "text/plain; charset=utf8")
		w.WriteHeader(getHTTPStatusCode(status))
		_, _ = fmt.Fprintln(w, status.Message())
		return
	}

	// If there is no response this is a Oneway Request.
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	applicationHeaders.ToHTTPHeaders(resp.Headers, w.Header())

	if resp.ApplicationError {
		w.Header().Set(ApplicationStatusHeader, ApplicationErrorStatus)
	}

	if supportsRespAndErr(req) && resp.FullApplicationError != nil {
		status := yarpcerrors.FromError(resp.FullApplicationError)
		w.Header().Set(BothResponseErrorHeader, AcceptTrue)
		writeErrorHeaders(w, status)
		w.WriteHeader(getHTTPStatusCode(status))
		writeBody(w, resp.Body)
		return
	}

	// If we're not using the advanced path, set status code to ok, and write
	// the body as usual.
	w.WriteHeader(http.StatusOK)
	writeBody(w, resp.Body)
}

func supportsRespAndErr(req *http.Request) bool {
	return isAcceptTrue(req.Header.Get(AcceptsBothResponseErrorHeader))
}

func writeBody(w http.ResponseWriter, resp io.ReadCloser) {
	if resp == nil {
		return
	}
	_, _ = io.Copy(w, resp) // TODO do something with the error
	resp.Close()
}

func writeErrorHeaders(w http.ResponseWriter, status *yarpcerrors.Status) {
	if statusCodeText, marshalErr := status.Code().MarshalText(); marshalErr != nil {
		status = yarpcerrors.Newf(yarpcerrors.CodeInternal, "error %s had code %v which is unknown", status.Error(), status.Code())
		w.Header().Set(ErrorCodeHeader, "internal")
	} else {
		w.Header().Set(ErrorCodeHeader, string(statusCodeText))
	}
	if status.Name() != "" {
		w.Header().Set(ErrorNameHeader, status.Name())
	}
}

func getHTTPStatusCode(status *yarpcerrors.Status) int {
	httpStatusCode, ok := _codeToStatusCode[status.Code()]
	if !ok {
		return http.StatusInternalServerError
	}
	return httpStatusCode
}

func (h handler) call(req *http.Request) (resp *transport.Response, contentType string, err error) {
	start := time.Now()
	defer req.Body.Close()
	if req.Method != http.MethodPost {
		return nil, "", yarpcerrors.Newf(yarpcerrors.CodeNotFound, "request method was %s but only %s is allowed", req.Method, http.MethodPost)
	}

	treq := h.toTransportRequest(req)
	if err := transport.ValidateRequest(treq); err != nil {
		return nil, "", errors.WrapHandlerError(err, treq.Service, treq.Procedure)
	}

	ctx := req.Context()
	ctx, cancel, parseTTLErr := parseTTL(ctx, treq, popHeader(req.Header, TTLMSHeader))
	// parseTTLErr != nil is a problem only if the request is unary.

	defer cancel()
	ctx, span := h.createSpan(ctx, req, treq, start)

	spec, err := h.router.Choose(ctx, treq)
	if err != nil {
		updateSpanWithErr(span, err)
		return nil, "", errors.WrapHandlerError(err, treq.Service, treq.Procedure)
	}

	switch spec.Type() {
	case transport.Unary:
		defer span.Finish()
		if parseTTLErr != nil {
			return nil, "", parseTTLErr
		}

		if err := transport.ValidateUnaryContext(ctx); err != nil {
			return nil, "", err
		}
		responseWriter := newResponseWriter()
		err = transport.DispatchUnaryHandler(ctx, spec.Unary(), start, treq, responseWriter)
		resp = responseWriter.Response()
	case transport.Oneway:
		err = handleOnewayRequest(span, treq, spec.Oneway())
	default:
		err = yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "transport http does not handle %s handlers", spec.Type().String())
	}

	updateSpanWithErr(span, err)
	return resp, getContentType(treq.Encoding), errors.WrapHandlerError(err, treq.Service, treq.Procedure)
}

func (h handler) toTransportRequest(req *http.Request) *transport.Request {
	treq := &transport.Request{
		Caller:          popHeader(req.Header, CallerHeader),
		Service:         popHeader(req.Header, ServiceHeader),
		Procedure:       popHeader(req.Header, ProcedureHeader),
		Encoding:        transport.Encoding(popHeader(req.Header, EncodingHeader)),
		ShardKey:        popHeader(req.Header, ShardKeyHeader),
		RoutingKey:      popHeader(req.Header, RoutingKeyHeader),
		RoutingDelegate: popHeader(req.Header, RoutingDelegateHeader),
		Headers:         applicationHeaders.FromHTTPHeaders(req.Header, transport.Headers{}),
		Body:            req.Body,
	}
	for header := range h.grabHeaders {
		if value := req.Header.Get(header); value != "" {
			treq.Headers = treq.Headers.With(header, value)
		}
	}
	return treq
}

func handleOnewayRequest(
	span opentracing.Span,
	treq *transport.Request,
	onewayHandler transport.OnewayHandler,
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
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	go func() {
		// ensure the span lasts for length of the handler in case of errors
		defer span.Finish()

		err := transport.DispatchOnewayHandler(ctx, onewayHandler, treq)
		updateSpanWithErr(span, err)
	}()
	return nil
}

func updateSpanWithErr(span opentracing.Span, err error) {
	if err != nil {
		span.SetTag("error", true)
		span.LogEvent(err.Error())
	}
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
	resp   *transport.Response
	buffer *bufferCloser
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		resp: &transport.Response{
			ApplicationError: false,
			Headers:          transport.NewHeaders(),
		},
	}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.buffer == nil {
		rw.buffer = &bufferCloser{
			Buffer: bufferpool.Get(),
		}
	}
	return rw.buffer.Write(s)
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.Items() {
		rw.resp.Headers = rw.resp.Headers.With(k, v)
	}
}

func (rw *responseWriter) SetApplicationError() {
	rw.resp.ApplicationError = true
}

func (rw *responseWriter) Response() *transport.Response {
	if rw.buffer != nil {
		rw.resp.Body = rw.buffer
	}
	return rw.resp
}

type bufferCloser struct {
	*bytes.Buffer
}

func (b *bufferCloser) Close() error {
	bufferpool.Put(b.Buffer)
	return nil
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
