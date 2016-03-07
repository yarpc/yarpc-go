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

package tchannel

import (
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// inboundCall is an interface similiar tchannel.InboundCall.
//
// We use it instead of *tchannel.InboundCall because tchannel.InboundCall is
// not an interface, so we have little control over its behavior in tests.
type inboundCall interface {
	Arg2Reader() (tchannel.ArgReader, error)
	Arg3Reader() (tchannel.ArgReader, error)
	CallerName() string
	Format() tchannel.Format
	MethodString() string
	Response() *tchannel.InboundCallResponse
	ServiceName() string
}

// handler wraps a transport.Handler into a TChannel Handler.
type handler struct {
	Handler transport.Handler
}

func (h handler) Handle(ctx context.Context, call *tchannel.InboundCall) {
	h.handle(ctx, call)
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	deadline, ok := ctx.Deadline()
	if !ok {
		call.Response().SendSystemError(tchannel.ErrTimeoutRequired)
		return
	}

	headers, err := readHeaders(call.Format(), call.Arg2Reader)
	if err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "failed to read headers: %v", err))
		return
	}

	body, err := call.Arg3Reader()
	if err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "failed to read body: %v", err))
		return
	}
	defer body.Close()

	rw := newResponseWriter(call)
	defer rw.Close() // TODO(abg): log if this errors

	treq := &transport.Request{
		Caller:    call.CallerName(),
		Service:   call.ServiceName(),
		Encoding:  transport.Encoding(call.Format()),
		Procedure: call.MethodString(),
		Headers:   headers,
		Body:      body,
		TTL:       deadline.Sub(time.Now()),
	}

	if err := h.Handler.Handle(ctx, treq, rw); err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "internal error: %v", err))
		return
	}
}

type responseWriter struct {
	failedWith   error
	bodyWriter   tchannel.ArgWriter
	format       tchannel.Format
	headers      transport.Headers
	response     *tchannel.InboundCallResponse
	wroteHeaders bool
}

func newResponseWriter(call inboundCall) *responseWriter {
	return &responseWriter{
		response: call.Response(),
		headers:  make(transport.Headers),
		format:   call.Format(),
	}
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	if rw.wroteHeaders {
		panic("AddHeaders() cannot be called after calling Write().")
	}
	for k, v := range h {
		rw.headers.Set(k, v)
	}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.failedWith != nil {
		return 0, rw.failedWith
	}

	if !rw.wroteHeaders {
		rw.wroteHeaders = true
		if err := writeHeaders(rw.format, rw.headers, rw.response.Arg2Writer); err != nil {
			rw.failedWith = err
			return 0, err
		}
	}

	if rw.bodyWriter == nil {
		var err error
		rw.bodyWriter, err = rw.response.Arg3Writer()
		if err != nil {
			rw.failedWith = err
			return 0, err
		}
	}

	return rw.bodyWriter.Write(s)
}

func (rw *responseWriter) Close() error {
	if rw.bodyWriter != nil {
		return rw.bodyWriter.Close()
	}
	if rw.failedWith != nil {
		return rw.failedWith
	}
	return nil
}
