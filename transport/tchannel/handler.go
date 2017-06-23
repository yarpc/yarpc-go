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
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/buffer"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/request"
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
	h.handle(ctx, tchannelCall{call})
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	// you MUST close the responseWriter no matter what
	responseWriter := newResponseWriter(call.Response(), call.Format())

	handlerErr := h.callHandler(ctx, call, responseWriter)
	if handlerErr != nil {
		// we have an error, so we're going to propagate it as a yarpc error,
		// regardless of whether or not it is a system error.
		yarpcError := yarpc.ToYARPCError(handlerErr)
		// TODO: what to do with error? we could have a whole complicated scheme to
		// return a SystemError here, might want to do that
		text, _ := yarpc.ErrorCode(yarpcError).MarshalText()
		responseWriter.addHeader(ErrorCodeHeaderKey, string(text))
		if name := yarpc.ErrorName(yarpcError); name != "" {
			responseWriter.addHeader(ErrorNameHeaderKey, name)
		}
		if message := yarpc.ErrorMessage(yarpcError); message != "" {
			responseWriter.addHeader(ErrorMessageHeaderKey, message)
		}
	}
	systemError, ok := getSystemError(handlerErr)
	if err := responseWriter.Close(ok); err != nil {
		// TODO: log error
		_ = call.Response().SendSystemError(tchannel.NewSystemError(tchannel.ErrCodeUnexpected, err.Error()))
		return
	}
	if ok {
		// TODO: log error
		_ = call.Response().SendSystemError(systemError)
	}
}

func (h handler) callHandler(ctx context.Context, call inboundCall, responseWriter *responseWriter) error {
	start := time.Now()
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

	if err := transport.ValidateRequest(treq); err != nil {
		return err
	}

	spec, err := h.router.Choose(ctx, treq)
	if err != nil {
		if yarpc.ErrorCode(err) != yarpc.CodeUnimplemented {
			return err
		}
		if tcall, ok := call.(tchannelCall); !ok {
			if m, ok := h.existing[call.MethodString()]; ok {
				m.Handle(ctx, tcall.InboundCall)
				return nil
			}
		}
		return err
	}

	switch spec.Type() {
	case transport.Unary:
		if err := request.ValidateUnaryContext(ctx); err != nil {
			return err
		}
		err = transport.DispatchUnaryHandler(ctx, spec.Unary(), start, treq, responseWriter)

	default:
		err = yarpc.UnimplementedErrorf("transport:tchannel type:%s", spec.Type().String())
	}

	return err
}

type responseWriter struct {
	// TODO: do we still need this?
	failedWith error
	format     tchannel.Format
	headers    transport.Headers
	buffer     *bytes.Buffer
	response   inboundCallResponse
}

func newResponseWriter(response inboundCallResponse, format tchannel.Format) *responseWriter {
	return &responseWriter{
		response: response,
		format:   format,
	}
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.Items() {
		// TODO: is this considered a breaking change?
		if isReservedHeaderKey(k) {
			panic("cannot use reserved header key " + k)
		}
		rw.addHeader(k, v)
	}
}

func (rw *responseWriter) addHeader(key string, value string) {
	rw.headers = rw.headers.With(key, value)
}

func (rw *responseWriter) SetApplicationError() {
	err := rw.response.SetApplicationError()
	if err != nil {
		// TODO: just set failedWith?
		panic(fmt.Sprintf("SetApplicationError() failed: %v", err))
	}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.failedWith != nil {
		return 0, rw.failedWith
	}

	if rw.buffer == nil {
		rw.buffer = buffer.Get()
	}

	n, err := rw.buffer.Write(s)
	if err != nil {
		rw.failedWith = err
	}
	return n, err
}

func (rw *responseWriter) Close(hasSystemError bool) error {
	retErr := writeHeaders(rw.format, rw.headers, rw.response.Arg2Writer)
	// TODO: the only reason the transport.Request is needed is for this error,
	// this whole setup with ResponseHeadersEncodeError should be changed
	//if retErr != nil {
	//retErr = encoding.ResponseHeadersEncodeError(rw.treq, err)
	//}

	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this
	// TODO: dhgfkjashhflkasjhflkjashflkasdjhfasldkjhasdlkjfhasdlkfhasdlkf
	if !hasSystemError && rw.buffer == nil {
		bodyWriter, err := rw.response.Arg3Writer()
		if err != nil {
			return multierr.Append(retErr, err)
		}
		defer func() { retErr = multierr.Append(retErr, bodyWriter.Close()) }()
	}
	if rw.buffer != nil {
		defer buffer.Put(rw.buffer)
		bodyWriter, err := rw.response.Arg3Writer()
		if err != nil {
			return multierr.Append(retErr, err)
		}
		defer func() { retErr = multierr.Append(retErr, bodyWriter.Close()) }()
		if _, err := bodyWriter.Write(rw.buffer.Bytes()); err != nil {
			return multierr.Append(retErr, err)
		}
	}

	if retErr != nil {
		return retErr
	}
	return rw.failedWith
}

// getSystemError returns a tchannel.SystemError if the given error represents one.
func getSystemError(err error) (tchannel.SystemError, bool) {
	// if there is no error, there is no SystemError
	if err == nil {
		return tchannel.SystemError{}, false
	}
	// if the error is a SystemError, return it
	if systemError, ok := err.(tchannel.SystemError); ok {
		return systemError, true
	}
	// if the error is not a YARPC error, return a SystemError of type ErrCodeUnexpected
	if !yarpc.IsYARPCError(err) {
		return tchannel.NewSystemError(tchannel.ErrCodeUnexpected, err.Error()).(tchannel.SystemError), true
	}

	// at this point, the error is a YARPC error that might be an application error
	// we figure out if there is a system error code that represents it

	tchannelCode, ok := CodeToTChannelCode[yarpc.ErrorCode(err)]
	if !ok {
		// there is no system code for the YARPC error, so it is an application error
		return tchannel.SystemError{}, false
	}
	// we have a system code, so we will make a SystemError
	return tchannel.NewSystemError(tchannelCode, yarpc.ErrorMessage(err)).(tchannel.SystemError), true
}
