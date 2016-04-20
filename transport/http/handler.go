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

	"github.com/yarpc/yarpc-go/internal/request"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

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

	err := h.callHandler(w, req)
	if err == nil {
		return
	}

	status := http.StatusInternalServerError
	if _, ok := err.(transport.BadRequestError); ok {
		status = http.StatusBadRequest
	}
	http.Error(w, err.Error(), status)
}

func (h handler) callHandler(w http.ResponseWriter, req *http.Request) error {
	treq := &transport.Request{
		Caller:    popHeader(req.Header, CallerHeader),
		Service:   popHeader(req.Header, ServiceHeader),
		Procedure: popHeader(req.Header, ProcedureHeader),
		Encoding:  transport.Encoding(popHeader(req.Header, EncodingHeader)),
		Headers:   fromHTTPHeader(req.Header, nil),
		Body:      req.Body,
	}

	v := request.Validator{Request: treq}
	v.ParseTTL(popHeader(req.Header, TTLMSHeader))
	treq, err := v.Validate()
	if err != nil {
		return transport.BadRequestError{Reason: err}
	}

	err = h.Handler.Handle(context.TODO(), treq, newResponseWriter(w))
	if err != nil {
		err = transport.AsHandlerError(treq.Service, treq.Procedure, err)
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
	toHTTPHeader(h, rw.w.Header())
}
