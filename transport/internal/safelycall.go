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

package internal

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"golang.org/x/net/context"
)

// SafelyCallHandler calls the handler h, recovering panics and timeout errors,
// converting them to yarpc errors. All other errors are passed trough.
func SafelyCallHandler(
	h transport.Handler,
	start time.Time,
	ctx context.Context,
	opts transport.Options,
	req *transport.Request,
	resq transport.ResponseWriter,
) (err error) {
	// We recover panics from now on.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Handler panicked: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	err = h.Handle(ctx, opts, req, resq)

	// The handler stopped work on context deadline.
	if err == context.DeadlineExceeded && err == ctx.Err() {
		deadline, _ := ctx.Deadline()
		err = errors.HandlerTimeoutError(req.Caller, req.Service,
			req.Procedure, deadline.Sub(start))
	}

	return err
}

// SafelyCallOnewayHandler calls the handler h, recovering panics and timeout errors,
// converting them to yarpc errors. All other errors are passed trough.
func SafelyCallOnewayHandler(
	h transport.OnewayHandler,
	start time.Time,
	ctx context.Context,
	opts transport.Options,
	req *transport.Request,
) (err error) {

	// We recover panics from now on.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Handler panicked: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	err = h.HandleOneway(ctx, opts, req)

	// The handler stopped work on context deadline.
	if err == context.DeadlineExceeded && err == ctx.Err() {
		deadline, _ := ctx.Deadline()
		err = errors.HandlerTimeoutError(req.Caller, req.Service,
			req.Procedure, deadline.Sub(start))
	}

	return err
}
