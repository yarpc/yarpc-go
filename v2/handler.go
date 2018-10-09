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

package yarpc

import (
	"context"

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

// HandlerSpec holds either a UnaryTransportHandler or StreamTransportHandler.
type HandlerSpec struct {
	t Type

	unaryHandler  UnaryTransportHandler
	streamHandler StreamTransportHandler
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (h HandlerSpec) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("rpcType", h.t.String())
	return nil
}

// Type returns the associated handler's type
func (h HandlerSpec) Type() Type { return h.t }

// Unary returns the Unary UnaryTransportHandler or nil
func (h HandlerSpec) Unary() UnaryTransportHandler { return h.unaryHandler }

// Stream returns the Stream StreamTransportHandler or nil
func (h HandlerSpec) Stream() StreamTransportHandler { return h.streamHandler }

// NewUnaryHandlerSpec returns an new HandlerSpec with a UnaryTransportHandler
func NewUnaryHandlerSpec(handler UnaryTransportHandler) HandlerSpec {
	return HandlerSpec{t: Unary, unaryHandler: handler}
}

// NewStreamHandlerSpec returns an new HandlerSpec with a StreamTransportHandler
func NewStreamHandlerSpec(handler StreamTransportHandler) HandlerSpec {
	return HandlerSpec{t: Streaming, streamHandler: handler}
}

// UnaryTransportHandler handles a single, transport-level, unary request.
type UnaryTransportHandler interface {
	// Handle the given request.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	Handle(context.Context, *Request, *Buffer) (*Response, *Buffer, error)
}

// StreamTransportHandler handles a stream connection request.
type StreamTransportHandler interface {
	// Handle the given stream connection. The stream will close when the function
	// returns.
	//
	// An error may be returned in case of failures.
	HandleStream(stream *ServerStream) error
}

// UnaryTransportHandlerFunc is a utility for defining a UnaryTransportHandler with just a
// function.
type UnaryTransportHandlerFunc func(context.Context, *Request, *Buffer) (*Response, *Buffer, error)

// StreamTransportHandlerFunc is a utility for defining a StreamTransportHandler with just a
// function.
type StreamTransportHandlerFunc func(*ServerStream) error

// Handle handles an inbound unary request.
func (f UnaryTransportHandlerFunc) Handle(ctx context.Context, r *Request, b *Buffer) (*Response, *Buffer, error) {
	return f(ctx, r, b)
}

// HandleStream handles an inbound streaming request.
func (f StreamTransportHandlerFunc) HandleStream(stream *ServerStream) error {
	return f(stream)
}
