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

// EncodingHandlerSpec holds either UnaryEncodingHandler or StreamEncodingHandler.
type EncodingHandlerSpec struct {
	t Type

	unaryHandler  UnaryEncodingHandler
	streamHandler StreamEncodingHandler
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (h EncodingHandlerSpec) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("rpcType", h.t.String())
	return nil
}

// Type returns the associated handler's type.
func (h EncodingHandlerSpec) Type() Type { return h.t }

// Unary returns the Unary UnaryEncodingHandler or nil.
func (h EncodingHandlerSpec) Unary() UnaryEncodingHandler { return h.unaryHandler }

// NewUnaryEncodingHandlerSpec returns a new EncodingHandlerSpec with a UnaryEncodingHandler.
func NewUnaryEncodingHandlerSpec(handler UnaryEncodingHandler) EncodingHandlerSpec {
	return EncodingHandlerSpec{t: Unary, unaryHandler: handler}
}

// NewStreamEncodingHandlerSpec returns a new EncodingHandlerSpec with a StreamEncodingHandler.
func NewStreamEncodingHandlerSpec(handler StreamEncodingHandler) EncodingHandlerSpec {
	return EncodingHandlerSpec{t: Streaming, streamHandler: handler}
}

// UnaryEncodingHandler handles a single, encoding-level, unary request.
// An encoding handler handles a request after the request has been decoded into a concrete
// instance specific to the procedure.
type UnaryEncodingHandler interface {
	// Handle the given request.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	Handle(ctx context.Context, reqBody interface{}) (interface{}, error)
}

// StreamEncodingHandler handles a stream connection request in the encoding layer.
type StreamEncodingHandler interface {
	HandleStream(stream *ServerStream) error
}

// UnaryEncodingHandlerFunc is a utility for defining a UnaryEncodingHandler with just a
// function.
type UnaryEncodingHandlerFunc func(context.Context, interface{}) (interface{}, error)

// Handle handles an inbound unary request.
func (f UnaryEncodingHandlerFunc) Handle(ctx context.Context, reqBody interface{}) (interface{}, error) {
	return f(ctx, reqBody)
}
