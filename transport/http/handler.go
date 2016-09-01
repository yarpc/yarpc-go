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
	"net/http"
	"time"

	"github.com/yarpc/yarpc-go/internal/baggage"
	"github.com/yarpc/yarpc-go/internal/errors"
	"github.com/yarpc/yarpc-go/internal/request"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

var httpOptions transport.Options

func popHeader(h http.Header, n string) string {
	v := h.Get(n)
	h.Del(n)
	return v
}

// handler adapts a transport.Handler into a handler for net/http.
type handler struct {
	Handler transport.Handler
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	if req.Method != "POST" {
		http.NotFound(w, req)
		return
	}

	service := req.Header.Get(ServiceHeader)
	procedure := req.Header.Get(ProcedureHeader)

	err := h.callHandler(w, req)
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

func (h handler) callHandler(w http.ResponseWriter, req *http.Request) error {
	treq := &transport.Request{
		Caller:    popHeader(req.Header, CallerHeader),
		Service:   popHeader(req.Header, ServiceHeader),
		Procedure: popHeader(req.Header, ProcedureHeader),
		Encoding:  transport.Encoding(popHeader(req.Header, EncodingHeader)),
		Headers:   applicationHeaders.FromHTTPHeaders(req.Header, transport.Headers{}),
		Body:      req.Body,
	}

	ctx := context.Background()

	v := request.Validator{Request: treq}
	ctx, cancel := v.ParseTTL(ctx, popHeader(req.Header, TTLMSHeader))
	defer cancel()

	treq, err := v.Validate(ctx)
	if err != nil {
		return err
	}

	headers := baggageHeaders.FromHTTPHeaders(req.Header, transport.Headers{})
	if headers.Len() > 0 {
		ctx = baggage.NewContextWithHeaders(ctx, headers.Items())
	}

	start := time.Now()

	// TODO capture and handle panic
	err = h.Handler.Handle(ctx, httpOptions, treq, newResponseWriter(w))

	// The handler is well behaved and stopped work on context deadline. We
	// forward this information to the client diligently.
	if err == context.DeadlineExceeded && err == ctx.Err() {
		deadline, _ := ctx.Deadline()
		err = errors.HandlerTimeoutError(treq.Caller, treq.Service,
			treq.Procedure, deadline.Sub(start))
	}

	return err
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
