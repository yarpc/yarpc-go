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
	"go.uber.org/zap"
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

// responseWriter provides an interface similar to newTchannelResponseWriter.
//
// It allows us to control tchannelResponseWriter during testing.
type responseWriter interface {
	AddHeaders(h transport.Headers)
	AddHeader(key string, value string)
	SetApplicationError()
	IsApplicationError() bool
	Write(s []byte) (int, error)
	Close() error
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
	existing          map[string]tchannel.Handler
	router            transport.Router
	tracer            opentracing.Tracer
	headerCase        headerCase
	logger            *zap.Logger
	newResponseWriter func(inboundCallResponse, tchannel.Format, headerCase) responseWriter
}

func (h handler) Handle(ctx ncontext.Context, call *tchannel.InboundCall) {
	h.handle(ctx, tchannelCall{call})
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	// you MUST close the responseWriter no matter what unless you have a tchannel.SystemError
	responseWriter := h.newResponseWriter(call.Response(), call.Format(), h.headerCase)

	// echo accepted rpc-service in response header
	responseWriter.AddHeader(ServiceHeaderKey, call.ServiceName())

	err := h.callHandler(ctx, call, responseWriter)

	// black-hole requests on resource exhausted errors
	if yarpcerrors.FromError(err).Code() == yarpcerrors.CodeResourceExhausted {
		// all TChannel clients will time out instead of receiving an error
		call.Response().Blackhole()
		return
	}
	if err != nil && !responseWriter.IsApplicationError() {

		_ = call.Response().SendSystemError(getSystemError(err))
		h.logger.Error("tchannel transport handler request failed", zap.Error(err))
		return
	}
	if err != nil && responseWriter.IsApplicationError() {
		// we have an error, so we're going to propagate it as a yarpc error,
		// regardless of whether or not it is a system error.
		status := yarpcerrors.FromError(errors.WrapHandlerError(err, call.ServiceName(), call.MethodString()))
		// TODO: what to do with error? we could have a whole complicated scheme to
		// return a SystemError here, might want to do that
		text, _ := status.Code().MarshalText()
		responseWriter.AddHeader(ErrorCodeHeaderKey, string(text))
		if status.Name() != "" {
			responseWriter.AddHeader(ErrorNameHeaderKey, status.Name())
		}
		if status.Message() != "" {
			responseWriter.AddHeader(ErrorMessageHeaderKey, status.Message())
		}
	}
	if err := responseWriter.Close(); err != nil {
		_ = call.Response().SendSystemError(getSystemError(err))
		h.logger.Error("tchannel responseWriter failed to close", zap.Error(err))
	}
}

func (h handler) callHandler(ctx context.Context, call inboundCall, responseWriter responseWriter) error {
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
		return transport.InvokeUnaryHandler(transport.UnaryInvokeRequest{
			Context:        ctx,
			StartTime:      start,
			Request:        treq,
			ResponseWriter: responseWriter,
			Handler:        spec.Unary(),
			Logger:         h.logger,
		})

	default:
		return yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "transport tchannel does not handle %s handlers", spec.Type().String())
	}
}

type tchannelResponseWriter struct {
	FailedWith       error
	Format           tchannel.Format
	Headers          transport.Headers
	Buffer           *bufferpool.Buffer
	Response         inboundCallResponse
	ApplicationError bool
	HeaderCase       headerCase
}

func newTchannelResponseWriter(response inboundCallResponse, format tchannel.Format, headerCase headerCase) responseWriter {
	return &tchannelResponseWriter{
		Response:   response,
		Format:     format,
		HeaderCase: headerCase,
	}
}

func (rw *tchannelResponseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.OriginalItems() {
		// TODO: is this considered a breaking change?
		if isReservedHeaderKey(k) {
			rw.FailedWith = appendError(rw.FailedWith, fmt.Errorf("cannot use reserved header key: %s", k))
			return
		}
		rw.AddHeader(k, v)
	}
}

func (rw *tchannelResponseWriter) AddHeader(key string, value string) {
	rw.Headers = rw.Headers.With(key, value)
}

func (rw *tchannelResponseWriter) SetApplicationError() {
	rw.ApplicationError = true
}

func (rw *tchannelResponseWriter) IsApplicationError() bool {
	return rw.ApplicationError
}

func (rw *tchannelResponseWriter) Write(s []byte) (int, error) {
	if rw.FailedWith != nil {
		return 0, rw.FailedWith
	}

	if rw.Buffer == nil {
		rw.Buffer = bufferpool.Get()
	}

	n, err := rw.Buffer.Write(s)
	if err != nil {
		rw.FailedWith = appendError(rw.FailedWith, err)
	}
	return n, err
}

func (rw *tchannelResponseWriter) Close() error {
	retErr := rw.FailedWith
	if rw.IsApplicationError() {
		if err := rw.Response.SetApplicationError(); err != nil {
			retErr = appendError(retErr, fmt.Errorf("SetApplicationError() failed: %v", err))
		}
	}

	headers := headerMap(rw.Headers, rw.HeaderCase)
	retErr = appendError(retErr, writeHeaders(rw.Format, headers, nil, rw.Response.Arg2Writer))

	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this
	bodyWriter, err := rw.Response.Arg3Writer()
	if err != nil {
		return appendError(retErr, err)
	}
	defer func() { retErr = appendError(retErr, bodyWriter.Close()) }()
	if rw.Buffer != nil {
		defer bufferpool.Put(rw.Buffer)
		if _, err := rw.Buffer.WriteTo(bodyWriter); err != nil {
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
