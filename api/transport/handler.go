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
	"go.uber.org/zap/zapcore"
)

// Type is an enum of RPC types
type Type int

const (
	// Unary types are traditional request/response RPCs
	Unary Type = iota + 1
	// Oneway types are fire and forget RPCs (no response)
	Oneway
	// Streaming types are Stream based RPCs (bidirectional messages over long
	// lived connections)
	Streaming
)

// HandlerSpec holds a handler and its Type
// one handler will be set, the other nil
type HandlerSpec struct {
	t Type

	unaryHandler  UnaryHandler
	onewayHandler OnewayHandler
	streamHandler StreamHandler
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (h HandlerSpec) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("rpcType", h.t.String())
	return nil
}

// Type returns the associated handler's type
func (h HandlerSpec) Type() Type { return h.t }

// Unary returns the Unary Handler or nil
func (h HandlerSpec) Unary() UnaryHandler { return h.unaryHandler }

// Oneway returns the Oneway Handler or nil
func (h HandlerSpec) Oneway() OnewayHandler { return h.onewayHandler }

// Stream returns the Stream Handler or nil
func (h HandlerSpec) Stream() StreamHandler { return h.streamHandler }

// NewUnaryHandlerSpec returns an new HandlerSpec with a UnaryHandler
func NewUnaryHandlerSpec(handler UnaryHandler) HandlerSpec {
	return HandlerSpec{t: Unary, unaryHandler: handler}
}

// NewOnewayHandlerSpec returns an new HandlerSpec with a OnewayHandler
func NewOnewayHandlerSpec(handler OnewayHandler) HandlerSpec {
	return HandlerSpec{t: Oneway, onewayHandler: handler}
}

// NewStreamHandlerSpec returns an new HandlerSpec with a StreamHandler
func NewStreamHandlerSpec(handler StreamHandler) HandlerSpec {
	return HandlerSpec{t: Streaming, streamHandler: handler}
}

// UnaryHandler handles a single, transport-level, unary request.
type UnaryHandler interface {
	// Handle the given request, writing the response to the given
	// ResponseWriter.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	//
	// Handlers MUST NOT retain references to the ResponseWriter.
	Handle(ctx context.Context, req *Request, resw ResponseWriter) error
}

// OnewayHandler handles a single, transport-level, oneway request.
type OnewayHandler interface {
	// Handle the given oneway request
	//
	// An error may be returned in case of failures.
	HandleOneway(ctx context.Context, req *Request) error
}

// StreamHandler handles a stream connection request.
type StreamHandler interface {
	// Handle the given stream connection.  The stream will close when the
	// function returns.
	//
	// An error may be returned in case of failures.
	HandleStream(stream *ServerStream) error
}

// DispatchUnaryHandler calls the handler h, recovering panics and timeout errors,
// converting them to yarpc errors. All other errors are passed trough.
func DispatchUnaryHandler(
	ctx context.Context,
	h UnaryHandler,
	start time.Time,
	req *Request,
	resq ResponseWriter,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Unary handler panicked: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
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

// DispatchOnewayHandler calls the oneway handler, recovering from panics as
// errors
func DispatchOnewayHandler(
	ctx context.Context,
	h OnewayHandler,
	req *Request,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Oneway handler panicked: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return h.HandleOneway(ctx, req)
}

// DispatchStreamHandler calls the stream handler, recovering from panics as
// errors.
func DispatchStreamHandler(
	h StreamHandler,
	stream *ServerStream,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Stream handler panicked: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return h.HandleStream(stream)
}
