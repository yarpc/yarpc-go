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

package tchannel

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/yarpcerrors"
	ncontext "golang.org/x/net/context"
)

// inboundCall provides an interface similar tchannel.InboundCall.
//
// We use it instead of *tchannel.InboundCall because tchannel.InboundCall is
// not an interface, so we have little control over its behavior in tests.
type inboundCall interface {
	ServiceName() string
	CallerName() string
	MethodString() string
	ShardKey() string
	RoutingKey() string
	RoutingDelegate() string

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
	Blackhole()
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
	existing   map[string]tchannel.Handler
	router     transport.Router
	tracer     opentracing.Tracer
	headerCase headerCase
}

func (h handler) Handle(ctx ncontext.Context, call *tchannel.InboundCall) {
	h.handle(ctx, tchannelCall{call})
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	// you MUST close the responseWriter no matter what unless you have a tchannel.SystemError
	responseWriter := newResponseWriter(call.Response(), call.Format(), h.headerCase)

	err := h.callHandler(ctx, call, responseWriter)
	if err != nil && !responseWriter.isApplicationError {
		// TODO: log error
		_ = call.Response().SendSystemError(getSystemError(err))
		return
	}
	if err != nil && responseWriter.isApplicationError {
		// we have an error, so we're going to propagate it as a yarpc error,
		// regardless of whether or not it is a system error.
		status := yarpcerrors.FromError(errors.WrapHandlerError(err, call.ServiceName(), call.MethodString()))
		// TODO: what to do with error? we could have a whole complicated scheme to
		// return a SystemError here, might want to do that
		text, _ := status.Code().MarshalText()
		responseWriter.addHeader(ErrorCodeHeaderKey, string(text))
		if status.Name() != "" {
			responseWriter.addHeader(ErrorNameHeaderKey, status.Name())
		}
		if status.Message() != "" {
			responseWriter.addHeader(ErrorMessageHeaderKey, status.Message())
		}
	}
	if err := responseWriter.Close(); err != nil {
		// TODO: log error
		_ = call.Response().SendSystemError(getSystemError(err))
	}
}

func (h handler) callHandler(ctx context.Context, call inboundCall, responseWriter *responseWriter) error {
	start := time.Now()
	_, ok := ctx.Deadline()
	if !ok {
		return tchannel.ErrTimeoutRequired
	}

	treq := &transport.Request{
		Caller:          call.CallerName(),
		Service:         call.ServiceName(),
		Encoding:        transport.Encoding(call.Format()),
		Transport:       transportName,
		Procedure:       call.MethodString(),
		ShardKey:        call.ShardKey(),
		RoutingKey:      call.RoutingKey(),
		RoutingDelegate: call.RoutingDelegate(),
	}

	ctx, headers, err := readRequestHeaders(ctx, call.Format(), call.Arg2Reader)
	if err != nil {
		return errors.RequestHeadersDecodeError(treq, err)
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
		if yarpcerrors.FromError(err).Code() != yarpcerrors.CodeUnimplemented {
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

	if err := transport.ValidateRequestContext(ctx); err != nil {
		return err
	}
	switch spec.Type() {
	case transport.Unary:
		return transport.DispatchUnaryHandler(ctx, spec.Unary(), start, treq, responseWriter)

	default:
		return yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "transport tchannel does not handle %s handlers", spec.Type().String())
	}
}

type responseWriter struct {
	failedWith         error
	format             tchannel.Format
	headers            transport.Headers
	buffer             *bufferpool.Buffer
	response           inboundCallResponse
	isApplicationError bool
	headerCase         headerCase
}

func newResponseWriter(response inboundCallResponse, format tchannel.Format, headerCase headerCase) *responseWriter {
	return &responseWriter{
		response:   response,
		format:     format,
		headerCase: headerCase,
	}
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.OriginalItems() {
		// TODO: is this considered a breaking change?
		if isReservedHeaderKey(k) {
			rw.failedWith = appendError(rw.failedWith, fmt.Errorf("cannot use reserved header key: %s", k))
			return
		}
		rw.addHeader(k, v)
	}
}

func (rw *responseWriter) addHeader(key string, value string) {
	rw.headers = rw.headers.With(key, value)
}

func (rw *responseWriter) SetApplicationError() {
	rw.isApplicationError = true
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.failedWith != nil {
		return 0, rw.failedWith
	}

	if rw.buffer == nil {
		rw.buffer = bufferpool.Get()
	}

	n, err := rw.buffer.Write(s)
	if err != nil {
		rw.failedWith = appendError(rw.failedWith, err)
	}
	return n, err
}

func (rw *responseWriter) Close() error {
	retErr := rw.failedWith
	if rw.isApplicationError {
		if err := rw.response.SetApplicationError(); err != nil {
			retErr = appendError(retErr, fmt.Errorf("SetApplicationError() failed: %v", err))
		}
	}

	headers := headerMap(rw.headers, rw.headerCase)
	retErr = appendError(retErr, writeHeaders(rw.format, headers, nil, rw.response.Arg2Writer))

	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this
	bodyWriter, err := rw.response.Arg3Writer()
	if err != nil {
		return appendError(retErr, err)
	}
	defer func() { retErr = appendError(retErr, bodyWriter.Close()) }()
	if rw.buffer != nil {
		defer bufferpool.Put(rw.buffer)
		if _, err := rw.buffer.WriteTo(bodyWriter); err != nil {
			return appendError(retErr, err)
		}
	}

	return retErr
}

func getSystemError(err error) error {
	if _, ok := err.(tchannel.SystemError); ok {
		return err
	}
	if !yarpcerrors.IsStatus(err) {
		return tchannel.NewSystemError(tchannel.ErrCodeUnexpected, err.Error())
	}
	status := yarpcerrors.FromError(err)
	tchannelCode, ok := _codeToTChannelCode[status.Code()]
	if !ok {
		tchannelCode = tchannel.ErrCodeUnexpected
	}
	return tchannel.NewSystemError(tchannelCode, status.Message())
}

func appendError(left error, right error) error {
	if _, ok := left.(tchannel.SystemError); ok {
		return left
	}
	if _, ok := right.(tchannel.SystemError); ok {
		return right
	}
	return multierr.Append(left, right)
}
