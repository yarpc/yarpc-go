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

package yarpctransport

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/zap"
)

// UnaryInvokeRequest encapsulates arguments to invoke a unary handler.
type UnaryInvokeRequest struct {
	Context   context.Context
	StartTime time.Time
	Request   *yarpc.Request
	Handler   yarpc.UnaryHandler
	Logger    *zap.Logger // optional
}

// StreamInvokeRequest encapsulates arguments to invoke a unary handler.
type StreamInvokeRequest struct {
	Stream  *yarpc.ServerStream
	Handler yarpc.StreamHandler
	Logger  *zap.Logger // optional
}

// InvokeUnaryHandler calls the handler h, recovering panics and timeout errors,
// converting them to YARPC errors. All other errors are passed through.
func InvokeUnaryHandler(
	i UnaryInvokeRequest,
) (res *yarpc.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(yarpc.Unary, i.Logger, r, i.Request.ToRequestMeta())
		}
	}()

	res, err = i.Handler.Handle(i.Context, i.Request)

	// The handler stopped work on context deadline.
	if err == context.DeadlineExceeded && err == i.Context.Err() {
		deadline, _ := i.Context.Deadline()
		err = yarpcerror.Newf(
			yarpcerror.CodeDeadlineExceeded,
			"call to procedure %q of service %q from caller %q timed out after %v",
			i.Request.Procedure, i.Request.Service, i.Request.Caller, deadline.Sub(i.StartTime))
	}
	return res, err
}

// InvokeStreamHandler calls the stream handler, recovering from panics as
// errors.
func InvokeStreamHandler(
	i StreamInvokeRequest,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(yarpc.Streaming, i.Logger, r, i.Stream.Request().Meta)
		}
	}()

	return i.Handler.HandleStream(i.Stream)
}

func handlePanic(rpcType yarpc.Type, logger *zap.Logger, recovered interface{}, reqMeta *yarpc.RequestMeta) error {
	err := fmt.Errorf("panic: %v", recovered)
	if logger != nil {
		logPanic(rpcType, logger, err, reqMeta)
		return err
	}
	log.Printf("%s handler panicked: %v\n%s", rpcType, recovered, debug.Stack())
	return err
}

func logPanic(rpcType yarpc.Type, logger *zap.Logger, err error, reqMeta *yarpc.RequestMeta) {
	logger.Error(fmt.Sprintf("%s handler panicked", rpcType),
		zap.String("service", reqMeta.Service),
		zap.String("transport", reqMeta.Transport),
		zap.String("procedure", reqMeta.Procedure),
		zap.String("encoding", string(reqMeta.Encoding)),
		zap.String("caller", reqMeta.Caller),
		zap.Error(err),
		zap.Stack("stack"),
	)
}
