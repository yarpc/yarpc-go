// Copyright (c) 2017 Uber Technologies, Inc.
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

package protobuf

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
)

// UnaryHandler represents a protobuf unary request handler.
//
// Users should use the server code generated rather than using this directly.
type UnaryHandler interface {
	// response message, application error, metadata, yarpc error
	Handle(ctx context.Context, requestMessage proto.Message) (proto.Message, error)
	NewRequest() proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
) UnaryHandler {
	return newUnaryHandler(handle, newRequest)
}

type unaryHandler struct {
	handle     func(context.Context, proto.Message) (proto.Message, error)
	newRequest func() proto.Message
}

func newUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
) UnaryHandler {
	return &unaryHandler{handle, newRequest}
}

func (u *unaryHandler) Handle(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	return u.handle(ctx, requestMessage)
}

func (u *unaryHandler) NewRequest() proto.Message {
	return u.newRequest()
}

type transportUnaryHandler struct {
	unaryHandler UnaryHandler
}

func newTransportUnaryHandler(unaryHandler UnaryHandler) *transportUnaryHandler {
	return &transportUnaryHandler{unaryHandler}
}

func (t *transportUnaryHandler) Handle(ctx context.Context, transportRequest *transport.Request, responseWriter transport.ResponseWriter) error {
	return nil
}
