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

package tchannel

import (
	"bytes"
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/internal/observability"
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

// responseWriter enhances transport.ResponseWriter interface with transport specific
// methods.
type responseWriter interface {
	transport.ResponseWriter

	AddSystemHeader(key string, value string)
	Close() error
	ReleaseBuffer()
	IsApplicationError() bool
	SetApplicationErrorMeta(meta *transport.ApplicationErrorMeta)
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
	existing                       map[string]tchannel.Handler
	router                         transport.Router
	tracer                         opentracing.Tracer
	headerCase                     headerCase
	logger                         *zap.Logger
	reservedHeaderMetrics          *observability.ReservedHeaderMetrics
	newResponseWriter              responseWriterConstructor
	excludeServiceHeaderInResponse bool
}

func (h handler) Handle(ctx ncontext.Context, call *tchannel.InboundCall) {
	h.handle(ctx, tchannelCall{call})
}

func (h handler) handle(ctx context.Context, call inboundCall) {
	// you MUST close the responseWriter no matter what unless you have a tchannel.SystemError
	responseWriter := h.newResponseWriter(call.Response(), call.Format(), h.headerCase, h.reservedHeaderMetrics.With(call.CallerName(), call.ServiceName()))
	defer responseWriter.ReleaseBuffer()

	if !h.excludeServiceHeaderInResponse {
		// echo accepted rpc-service in response header
		responseWriter.AddSystemHeader(ServiceHeaderKey, call.ServiceName())
	}

	err := h.callHandler(ctx, call, responseWriter)

	// black-hole requests on resource exhausted errors
	if yarpcerrors.FromError(err).Code() == yarpcerrors.CodeResourceExhausted {
		// all TChannel clients will time out instead of receiving an error
		call.Response().Blackhole()
		return
	}

	clientTimedOut := ctx.Err() == context.DeadlineExceeded

	if err != nil && !responseWriter.IsApplicationError() {
		sendSysErr := call.Response().SendSystemError(getSystemError(err))
		if sendSysErr != nil && !clientTimedOut {
			// only log errors if client is still waiting for our response
			h.logger.Error("SendSystemError failed", zap.Error(sendSysErr))
		}
		return
	}
	if err != nil && responseWriter.IsApplicationError() {
		// we have an error, so we're going to propagate it as a yarpc error,
		// regardless of whether or not it is a system error.
		status := yarpcerrors.FromError(errors.WrapHandlerError(err, call.ServiceName(), call.MethodString()))
		// TODO: what to do with error? we could have a whole complicated scheme to
		// return a SystemError here, might want to do that
		text, _ := status.Code().MarshalText()
		responseWriter.AddSystemHeader(ErrorCodeHeaderKey, string(text))
		if status.Name() != "" {
			responseWriter.AddSystemHeader(ErrorNameHeaderKey, status.Name())
		}
		if status.Message() != "" {
			responseWriter.AddSystemHeader(ErrorMessageHeaderKey, status.Message())
		}
	}
	if reswErr := responseWriter.Close(); reswErr != nil && !clientTimedOut {
		if sendSysErr := call.Response().SendSystemError(getSystemError(reswErr)); sendSysErr != nil {
			h.logger.Error("SendSystemError failed", zap.Error(sendSysErr))
		}
		h.logger.Error("responseWriter failed to close", zap.Error(reswErr))
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
		Transport:       TransportName,
		Procedure:       call.MethodString(),
		ShardKey:        call.ShardKey(),
		RoutingKey:      call.RoutingKey(),
		RoutingDelegate: call.RoutingDelegate(),
	}

	ctx, headers, err := readRequestHeaders(ctx, call.Format(), call.Arg2Reader)
	if err != nil {
		return errors.RequestHeadersDecodeError(treq, err)
	}

	transportHeadersToRequest(treq, headers)
	deleteReservedPrefixHeaders(headers, h.reservedHeaderMetrics.With(call.CallerName(), call.ServiceName()))
	treq.Headers = headers

	if tcall, ok := call.(tchannelCall); ok {
		tracer := h.tracer
		ctx = tchannel.ExtractInboundSpan(ctx, tcall.InboundCall, headers.Items(), tracer)
	}

	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	body, err := call.Arg3Reader()
	if err != nil {
		return err
	}

	if _, err = buf.ReadFrom(body); err != nil {
		return err
	}
	if err = body.Close(); err != nil {
		return err
	}

	treq.Body = bytes.NewReader(buf.Bytes())
	treq.BodySize = buf.Len()

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
