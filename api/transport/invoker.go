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

// InvokeUnaryHandler calls the handler h, recovering panics and timeout errors,
// converting them to yarpc errors. All other errors are passed through.
func InvokeUnaryHandler(
	ctx context.Context,
	h UnaryHandler,
	start time.Time,
	req *Request,
	resq ResponseWriter,
	logger *zap.Logger,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = logPanic(Unary, logger, r, req.ToRequestMeta())
		}
	}()

	err = h.Handle(ctx, req, resq)

	// The handler stopped work on context deadline.
	if err == context.DeadlineExceeded && err == ctx.Err() {
		deadline, _ := ctx.Deadline()
		err = yarpcerrors.Newf(
			yarpcerrors.CodeDeadlineExceeded,
			"call to procedure %q of service %q from caller %q timed out after %v",
			req.Procedure, req.Service, req.Caller, deadline.Sub(start))
	}
	return err
}

// InvokeOnewayHandler calls the oneway handler, recovering from panics as
// errors
func InvokeOnewayHandler(
	ctx context.Context,
	h OnewayHandler,
	req *Request,
	logger *zap.Logger,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = logPanic(Oneway, logger, r, req.ToRequestMeta())
		}
	}()

	return h.HandleOneway(ctx, req)
}

// InvokeStreamHandler calls the stream handler, recovering from panics as
// errors.
func InvokeStreamHandler(
	h StreamHandler,
	stream *ServerStream,
	logger *zap.Logger,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = logPanic(Streaming, logger, r, stream.Request().Meta)
		}
	}()

	return h.HandleStream(stream)
}

func logPanic(rpcType Type, logger *zap.Logger, recovered interface{}, req *RequestMeta) error {
	err := fmt.Errorf("panic: %v", recovered)
	if logger != nil {
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
	log.Printf("%s handler panicked: %v\n%s", rpcType, recovered, debug.Stack())
	return err
}
