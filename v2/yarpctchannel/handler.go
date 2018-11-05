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

package yarpctchannel

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/multierr"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaliopool"
	"go.uber.org/yarpc/v2/yarpcencoding"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpctransport"
	"go.uber.org/zap"
	netcontext "golang.org/x/net/context"
)

// handler wraps a yarpc.UnaryHandler into a TChannel Handler.
type handler struct {
	router     yarpc.Router
	headerCase HeaderCase
	addr       string
	tracer     opentracing.Tracer
	logger     *zap.Logger
}

// Handle implements the interface of a TChannel call request handler.
func (h handler) Handle(ctx netcontext.Context, call *tchannel.InboundCall) {
	// We pass the request on to the testable handle, by wrapping the TChannel
	// type inbound call in a private interface we can mock or fake.
	h.handle(ctx, tchannelCall{call})
}

// handle is the testable entry point of a handler, accepting both genuine
// TChannel and mock inbound calls.
func (h handler) handle(ctx context.Context, call inboundCall) {
	if err := h.handleOrSystemError(ctx, call); err != nil {
		err = call.Response().SendSystemError(getSystemError(err))
		if err != nil {
			// TODO tag this error sufficiently enough that it can be traced.
			h.logger.Error("failed to respond with tchannel system error", zap.Error(err))
		}
	}
}

// handleOrSystemError drives the read request, handle, and write response
// cycle, returning an error only if that error should effect a TChannel
// system error frame emission.
func (h handler) handleOrSystemError(ctx context.Context, call inboundCall) error {
	start := time.Now()
	_, ok := ctx.Deadline()
	if !ok {
		return tchannel.ErrTimeoutRequired
	}

	ctx, req, reqBody, err := h.readRequest(ctx, call)
	if err != nil {
		return err
	}
	response, responseBody, err := h.handleKernel(ctx, start, req, reqBody)
	return h.writeResponse(ctx, call, response, responseBody, err)
}

func (h handler) readRequest(ctx context.Context, call inboundCall) (context.Context, *yarpc.Request, *yarpc.Buffer, error) {
	req := &yarpc.Request{
		Caller:          call.CallerName(),
		Service:         call.ServiceName(),
		Encoding:        yarpc.Encoding(call.Format()),
		Transport:       transportName,
		Procedure:       call.MethodString(),
		ShardKey:        call.ShardKey(),
		RoutingKey:      call.RoutingKey(),
		RoutingDelegate: call.RoutingDelegate(),
	}

	ctx, headers, err := readRequestHeaders(ctx, call.Format(), call.Arg2Reader)
	if err != nil {
		return nil, nil, nil, yarpcencoding.RequestHeadersDecodeError(req, err)
	}
	req.Headers = headers

	if tcall, ok := call.(tchannelCall); ok {
		ctx = tchannel.ExtractInboundSpan(ctx, tcall.InboundCall, headers.Items(), h.tracer)
	}

	// Read request body into a buffer.
	requestBodyReader, err := call.Arg3Reader()
	if err != nil {
		return nil, nil, nil, err
	}
	defer requestBodyReader.Close()
	reqBody := yarpc.NewBufferBytes(nil)
	_, err = internaliopool.Copy(reqBody, requestBodyReader)
	if err != nil {
		return nil, nil, nil, err
	}

	return ctx, req, reqBody, nil
}

// handleKernel implements the portion of the request, handle, response
// lifecycle that is incidental to TChannel, operating within the YARPC
// abstraction.
func (h handler) handleKernel(ctx context.Context, start time.Time, req *yarpc.Request, reqBody *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if err := yarpc.ValidateRequest(req); err != nil {
		return nil, nil, err
	}

	spec, err := h.router.Choose(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if err := yarpc.ValidateRequestContext(ctx); err != nil {
		return nil, nil, err
	}

	switch spec.Type() {
	case yarpc.Unary:
		return yarpctransport.InvokeUnaryHandler(yarpctransport.UnaryInvokeRequest{
			Context:   ctx,
			StartTime: start,
			Request:   req,
			Buffer:    reqBody,
			Handler:   spec.Unary(),
			Logger:    h.logger,
		})

	default:
		return nil, nil, yarpcerror.Newf(yarpcerror.CodeUnimplemented, "transport tchannel does not handle %s requests", spec.Type().String())
	}
}

// writeResponse takes a YARPC response or error and writes a TChannel call
// response, returns an error that should effect a TChannel system error frame,
// or logs an error in the attempt and returns nothing.
// Any error returned by writeResponse indicates that no response has been made
// and to instead send a system error.
// All other success and errors must be handled before returning.
func (h handler) writeResponse(ctx context.Context, call inboundCall, res *yarpc.Response, responseBody *yarpc.Buffer, retErr error) error {
	// Black-hole requests on resource exhausted errors.
	if yarpcerror.FromError(retErr).Code() == yarpcerror.CodeResourceExhausted {
		// All TChannel clients will time out instead of receiving an error.
		call.Response().Blackhole()
		// Nothing to see here. Move along.
		return nil
	}
	if retErr != nil && (res == nil || res.ApplicationError == nil) {
		// System error.
		return retErr
	}

	res.Headers = res.Headers.With(PeerHeaderKey, h.addr)

	if retErr != nil && res != nil && res.ApplicationError != nil {
		// We have an error, so we're going to propagate it as a yarpc error,
		// regardless of whether or not it is a system error.
		status := yarpcerror.FromError(yarpcerror.WrapHandlerError(retErr, call.ServiceName(), call.MethodString()))
		text, err := status.Code().MarshalText()
		if err != nil {
			return appendError(retErr, err)
		}
		res.Headers = res.Headers.With(ErrorCodeHeaderKey, string(text))
		if status.Name() != "" {
			res.Headers = res.Headers.With(ErrorNameHeaderKey, status.Name())
		}
		if status.Message() != "" {
			res.Headers = res.Headers.With(ErrorMessageHeaderKey, status.Message())
		}
	}

	// This is the point of no return. We have committed to sending a call
	// response. Hereafter, all failures while sending the error must be logged
	// and the response aborted.
	if res.ApplicationError != nil {
		if err := call.Response().SetApplicationError(); err != nil {
			retErr = appendError(retErr, fmt.Errorf("SetApplicationError() failed: %v", err))
		}
	}

	// Echo accepted service in response header for client side verification.
	res.Headers = res.Headers.With(ServiceHeaderKey, call.ServiceName())

	// Write application headers.
	headers := headerMap(res.Headers, h.headerCase)
	retErr = appendError(retErr, writeHeaders(call.Format(), headers, nil, call.Response().Arg2Writer))

	// Write response body.
	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this.
	responseBodyWriter, err := call.Response().Arg3Writer()
	if err != nil {
		return appendError(retErr, err)
	}
	// Hereafter, the response body writer must be closed.
	defer func() {
		retErr = appendError(retErr, responseBodyWriter.Close())
	}()
	if responseBody != nil {
		// TODO CAREFULLY restore buffer pooling
		// defer yarpcbufferpool.Put(responseBody)
		if _, err := responseBody.WriteTo(responseBodyWriter); err != nil {
			retErr = appendError(retErr, err)
		}
	}

	if retErr != nil {
		// TODO tag this error sufficiently enough that it can be traced.
		h.logger.Error("failed to respond to tchannel request", zap.Error(retErr))
	}

	return nil
}

func getSystemError(err error) error {
	if _, ok := err.(tchannel.SystemError); ok {
		return err
	}
	if !yarpcerror.IsStatus(err) {
		return tchannel.NewSystemError(tchannel.ErrCodeUnexpected, err.Error())
	}
	status := yarpcerror.FromError(err)
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
