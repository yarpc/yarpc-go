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

package transport

import "context"

//go:generate mockgen -destination=transporttest/handler.go -package=transporttest go.uber.org/yarpc/api/transport UnaryHandler,OnewayHandler

// Type is an enum of RPC types
type Type int

//go:generate stringer -type=Type

const (
	// Unary types are traditional request/response RPCs
	Unary Type = iota + 1
	// Oneway types are fire and forget RPCs (no response)
	Oneway
)

// HandlerSpec holds a handler and its Type
// one handler will be set, the other nil
type HandlerSpec struct {
	t Type

	unaryHandler  UnaryHandler
	onewayHandler OnewayHandler
}

// Type returns the associated handler's type
func (h HandlerSpec) Type() Type { return h.t }

// Unary returns the Unary Handler or nil
func (h HandlerSpec) Unary() UnaryHandler { return h.unaryHandler }

// Oneway returns the Oneway Handler or nil
func (h HandlerSpec) Oneway() OnewayHandler { return h.onewayHandler }

// NewUnaryHandlerSpec returns an new HandlerSpec with a UnaryHandler
func NewUnaryHandlerSpec(handler UnaryHandler) HandlerSpec {
	return HandlerSpec{t: Unary, unaryHandler: handler}
}

// NewOnewayHandlerSpec returns an new HandlerSpec with a OnewayHandler
func NewOnewayHandlerSpec(handler OnewayHandler) HandlerSpec {
	return HandlerSpec{t: Oneway, onewayHandler: handler}
}

// UnaryHandler handles a single, transport-level, unary request.
type UnaryHandler interface {
	// Handle the given request, writing the response to the given
	// ResponseWriter.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	Handle(ctx context.Context, req *Request, resw ResponseWriter) error
}

// OnewayHandler handles a single, transport-level, oneway request.
type OnewayHandler interface {
	// Handle the given oneway request
	//
	// An error may be returned in case of failures.
	HandleOneway(ctx context.Context, req *Request) error
}
