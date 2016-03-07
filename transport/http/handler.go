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
	"strconv"
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// handler adapts a transport.Handler into a handler for net/http.
type handler struct {
	Handler transport.Handler
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.NotFound(w, req)
		return
	}

	defer req.Body.Close()

	caller := req.Header.Get(CallerHeader)
	if len(caller) == 0 {
		http.Error(w, "caller name is required", http.StatusBadRequest)
		return
	}
	req.Header.Del(CallerHeader)

	service := req.Header.Get(ServiceHeader)
	if len(service) == 0 {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}
	req.Header.Del(ServiceHeader)

	procedure := req.Header.Get(ProcedureHeader)
	if len(procedure) == 0 {
		http.Error(w, "procedure name is required", http.StatusBadRequest)
		return
	}
	req.Header.Del(ProcedureHeader)

	strttlms := req.Header.Get(TTLMSHeader)
	if len(strttlms) == 0 {
		http.Error(w, "ttlms is required", http.StatusBadRequest)
		return
	}
	req.Header.Del(TTLMSHeader)
	ttlms, err := strconv.Atoi(strttlms)
	if err != nil {
		http.Error(w, "ttlms must be an int", http.StatusBadRequest)
		return
	}

	// Optional headers:

	encoding := req.Header.Get(EncodingHeader)
	req.Header.Del(EncodingHeader)

	treq := &transport.Request{
		Caller:    caller,
		Service:   service,
		Encoding:  transport.Encoding(encoding),
		Procedure: procedure,
		Headers:   fromHTTPHeader(req.Header, nil),
		Body:      req.Body,
		TTL:       time.Duration(ttlms) * time.Millisecond,
	}

	err = h.Handler.Handle(context.TODO(), treq, newResponseWriter(w))
	if err != nil {
		// TODO structured responses?
		err = internalError{Reason: err}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
