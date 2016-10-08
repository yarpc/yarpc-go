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

import "golang.org/x/net/context"

// Mode is an enum of RPC types
type Mode int

const (
	// Unknown RPC type
	Unknown = iota
	// Unary RPC type
	Unary = iota
	// Oneway RPC type
	Oneway
)

func (m Mode) String() string {
	switch m {
	case Unary:
		return "Unary"
	case Oneway:
		return "Oneway"
	default:
		return "Unknown"
	}
}

// HandlerSpec holds a handler and its mode
type HandlerSpec struct {
	Mode Mode

	Handler       Handler
	OnewayHandler OnewayHandler
}

// Handler handles a single, transport-level, unary request.
type Handler interface {
	// Handle the given request, writing the response to the given
	// ResponseWriter.
	//
	// An error may be returned in case of failures. BadRequestError must be
	// returned for invalid requests. All other failures are treated as
	// UnexpectedErrors.
	Handle(
		ctx context.Context,
		opts Options,
		req *Request,
		resw ResponseWriter,
	) error
}

// OnewayHandler handles a single, transport-level, oneway request.
type OnewayHandler interface {
	// Handle the given oneway request
	//
	// An error may be returned in case of failures.
	// TODO: determine oneway errors and how to deal with them
	HandleOneway(
		ctx context.Context,
		opts Options,
		req *Request,
	) error
}
