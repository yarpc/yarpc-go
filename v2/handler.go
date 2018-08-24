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
	"time"

	"go.uber.org/zap/zapcore"
)

// Type is an enum of RPC types
type Type int

const (
	// Unary types are traditional request/response RPCs
	Unary Type = iota + 1
	// Streaming types are Stream based RPCs (bidirectional messages over long
	// lived connections)
	Streaming
)

// HandlerSpec holds a handler and its Type
// one handler will be set, the other nil
type HandlerSpec struct {
	t Type

	unaryHandler  UnaryHandler
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

// Stream returns the Stream Handler or nil
func (h HandlerSpec) Stream() StreamHandler { return h.streamHandler }

// NewUnaryHandlerSpec returns an new HandlerSpec with a UnaryHandler
func NewUnaryHandlerSpec(handler UnaryHandler) HandlerSpec {
	return HandlerSpec{t: Unary, unaryHandler: handler}
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
//
// Deprecated: Use InvokeUnaryHandler instead.
func DispatchUnaryHandler(
	ctx context.Context,
	h UnaryHandler,
	start time.Time,
	req *Request,
	resq ResponseWriter,
) (err error) {
	return InvokeUnaryHandler(UnaryInvokeRequest{
		Context:        ctx,
		StartTime:      start,
		Request:        req,
		ResponseWriter: resq,
		Handler:        h,
	})
}

// DispatchStreamHandler calls the stream handler, recovering from panics as
// errors.
//
// Deprecated: Use InvokeStreamHandler instead.
func DispatchStreamHandler(
	h StreamHandler,
	stream *ServerStream,
) (err error) {
	return InvokeStreamHandler(StreamInvokeRequest{
		Stream:  stream,
		Handler: h,
	})
}
