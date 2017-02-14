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

package tchannel

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	ncontext "golang.org/x/net/context"
)

// inboundCall provides an interface similiar tchannel.InboundCall.
//
// We use it instead of *tchannel.InboundCall because tchannel.InboundCall is
// not an interface, so we have little control over its behavior in tests.
type inboundCall interface {
	ServiceName() string
	CallerName() string
	MethodString() string
	Format() tchannel.Format

	Arg2Reader() (tchannel.ArgReader, error)
	Arg3Reader() (tchannel.ArgReader, error)

	Response() inboundCallResponse
}

// inboundCallResponse provides an interface similar to
// tchannel.InboundCallResponse.
//
// Its purpose is the same as inboundCall: Make it easier to test functions
// that consume InboundCallResponse without having control of
// InboundCallResponse's behavior.
type inboundCallResponse interface {
	Arg2Writer() (tchannel.ArgWriter, error)
	Arg3Writer() (tchannel.ArgWriter, error)
	SendSystemError(err error) error
	SetApplicationError() error
}

// tchannelCall wraps a TChannel InboundCall into an inboundCall.
//
// We need to do this so that we can change the return type of call.Response()
// to match inboundCall's Response().
type tchannelCall struct{ *tchannel.InboundCall }

func (c tchannelCall) Response() inboundCallResponse {
	return c.InboundCall.Response()
}

// handler wraps a transport.UnaryHandler into a TChannel Handler.
type handler struct {
	existing map[string]tchannel.Handler
	router   transport.Router
	tracer   opentracing.Tracer
}

func (h handler) Handle(ctx ncontext.Context, call *tchannel.InboundCall) {
	if m, ok := h.existing[call.MethodString()]; ok {
		m.Handle(ctx, call)
		return
	}

	h.handle(ctx, tchannelCall{call})
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	start := time.Now()
	err := h.callHandler(ctx, call, start)
	if err == nil {
		return
	}

	if _, ok := err.(tchannel.SystemError); ok {
		// TODO: log error
		_ = call.Response().SendSystemError(err)
		return
	}

	err = errors.AsHandlerError(call.ServiceName(), call.MethodString(), err)
	status := tchannel.ErrCodeUnexpected
	if transport.IsBadRequestError(err) {
		status = tchannel.ErrCodeBadRequest
	} else if transport.IsTimeoutError(err) {
		status = tchannel.ErrCodeTimeout
	}

	// TODO: log error
	_ = call.Response().SendSystemError(tchannel.NewSystemError(status, err.Error()))
}

func (h handler) callHandler(ctx context.Context, call inboundCall, start time.Time) error {
	_, ok := ctx.Deadline()
	if !ok {
		return tchannel.ErrTimeoutRequired
	}

	treq := &transport.Request{
		Caller:    call.CallerName(),
		Service:   call.ServiceName(),
		Encoding:  transport.Encoding(call.Format()),
		Procedure: call.MethodString(),
	}

	ctx, headers, err := readRequestHeaders(ctx, call.Format(), call.Arg2Reader)
	if err != nil {
		return encoding.RequestHeadersDecodeError(treq, err)
	}
	treq.Headers = headers

	if tcall, ok := call.(tchannelCall); ok {
		tracer := h.tracer
		ctx = tchannel.ExtractInboundSpan(ctx, tcall.InboundCall, headers.Items(), tracer)
	}

	body, err := call.Arg3Reader()
	if err != nil {
		return err
	}
	defer body.Close()
	treq.Body = body

	rw := newResponseWriter(treq, call)
	defer rw.Close() // TODO(abg): log if this errors

	if err := transport.ValidateRequest(treq); err != nil {
		return err
	}

	spec, err := h.router.Choose(ctx, treq)
	if err != nil {
		return err
	}

	switch spec.Type() {
	case transport.Unary:
		if err := request.ValidateUnaryContext(ctx); err != nil {
			return err
		}
		err = transport.DispatchUnaryHandler(ctx, spec.Unary(), start, treq, rw)

	default:
		err = errors.UnsupportedTypeError{Transport: "TChannel", Type: spec.Type().String()}
	}

	return err
}

type responseWriter struct {
	treq         *transport.Request
	failedWith   error
	bodyWriter   tchannel.ArgWriter
	format       tchannel.Format
	headers      transport.Headers
	response     inboundCallResponse
	wroteHeaders bool
}

func newResponseWriter(treq *transport.Request, call inboundCall) *responseWriter {
	return &responseWriter{
		treq:     treq,
		response: call.Response(),
		format:   call.Format(),
	}
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	if rw.wroteHeaders {
		panic("AddHeaders() cannot be called after calling Write().")
	}
	for k, v := range h.Items() {
		rw.headers = rw.headers.With(k, v)
	}
}

func (rw *responseWriter) SetApplicationError() {
	if rw.wroteHeaders {
		panic("SetApplicationError() cannot be called after calling Write().")
	}
	err := rw.response.SetApplicationError()
	if err != nil {
		panic(fmt.Sprintf("SetApplicationError() failed: %v", err))
	}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.failedWith != nil {
		return 0, rw.failedWith
	}

	if !rw.wroteHeaders {
		rw.wroteHeaders = true
		if err := writeHeaders(rw.format, rw.headers, rw.response.Arg2Writer); err != nil {
			err = encoding.ResponseHeadersEncodeError(rw.treq, err)
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

	n, err := rw.bodyWriter.Write(s)
	if err != nil {
		rw.failedWith = err
	}
	return n, err
}

func (rw *responseWriter) Close() error {
	var err error
	if rw.bodyWriter != nil {
		err = rw.bodyWriter.Close()
	}
	if rw.failedWith != nil {
		return rw.failedWith
	}
	return err
}
