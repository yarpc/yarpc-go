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

package transport

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

// UnaryInvokeRequest encapsulates minimum arguments to invoke a unary handler
type UnaryInvokeRequest struct {
	Context        context.Context
	StartTime      time.Time
	Request        *Request
	ResponseWriter ResponseWriter
	Handler        UnaryHandler
	Options        *InvokerOptions
}

// OnewayInvokeRequest encapsulates minimum arguments to invoke a unary handler
type OnewayInvokeRequest struct {
	Context context.Context
	Request *Request
	Handler OnewayHandler
	Options *InvokerOptions
}

// StreamInvokeRequest encapsulates minimum arguments to invoke a unary handler
type StreamInvokeRequest struct {
	Stream  *ServerStream
	Handler StreamHandler
	Options *InvokerOptions
}

// InvokeUnaryHandler calls the handler h, recovering panics and timeout errors,
// converting them to yarpc errors. All other errors are passed through.
func InvokeUnaryHandler(
	i UnaryInvokeRequest,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(Unary, i.Options, r, i.Request.ToRequestMeta())
		}
	}()

	err = i.Handler.Handle(i.Context, i.Request, i.ResponseWriter)

	// The handler stopped work on context deadline.
	if err == context.DeadlineExceeded && err == i.Context.Err() {
		deadline, _ := i.Context.Deadline()
		err = yarpcerrors.Newf(
			yarpcerrors.CodeDeadlineExceeded,
			"call to procedure %q of service %q from caller %q timed out after %v",
			i.Request.Procedure, i.Request.Service, i.Request.Caller, deadline.Sub(i.StartTime))
	}
	return err
}

// InvokeOnewayHandler calls the oneway handler, recovering from panics as
// errors
func InvokeOnewayHandler(
	i OnewayInvokeRequest,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(Oneway, i.Options, r, i.Request.ToRequestMeta())
		}
	}()

	return i.Handler.HandleOneway(i.Context, i.Request)
}

// InvokeStreamHandler calls the stream handler, recovering from panics as
// errors.
func InvokeStreamHandler(
	i StreamInvokeRequest,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(Streaming, i.Options, r, i.Stream.Request().Meta)
		}
	}()

	return i.Handler.HandleStream(i.Stream)
}

func handlePanic(rpcType Type, options *InvokerOptions, recovered interface{}, req *RequestMeta) error {
	err := fmt.Errorf("panic: %v", recovered)
	if options != nil {
		if options.logger != nil {
			logPanic(rpcType, options.logger, err, req)
			return err
		}
	}
	log.Printf("%s handler panicked: %v\n%s", rpcType, recovered, debug.Stack())
	return err
}

func logPanic(rpcType Type, logger *zap.Logger, err error, req *RequestMeta) error {
	logger.Error(fmt.Sprintf("%s handler panicked", rpcType),
		zap.String("service", req.Service),
		zap.String("transport", req.Transport),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("caller", req.Caller),
		zap.Error(err),
		zap.Stack("stack"),
	)
	return err
}
