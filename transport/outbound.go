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

//go:generate mockgen -destination=transporttest/outbound.go -package=transporttest go.uber.org/yarpc/transport UnaryOutbound,OnewayOutbound

// Outbound is the common interface for all outbounds
type Outbound interface {
	// Sets up the outbound to start making calls.
	//
	// This MUST block until the outbound is ready to start sending requests.
	// This MUST be idempotent and thread-safe. If called multiple times, only
	// the first call's dependencies are used
	Start() error

	// Stops the outbound, cleaning up any resources held by the Outbound.
	//
	// This MUST be idempotent and thread-safe. This MAY be called more than once
	Stop() error
}

// UnaryOutbound is a transport that knows how to send unary requests for procedure
// calls.
type UnaryOutbound interface {
	Outbound

	// Call sends the given request through this transport and returns its
	// response.
	//
	// This MUST NOT be called before Start() has been called successfully. This
	// MAY panic if called without calling Start(). This MUST be safe to call
	// concurrently.
	Call(ctx context.Context, request *Request) (*Response, error)
}

// OnewayOutbound is a transport that knows how to send oneway requests for
// procedure calls.
type OnewayOutbound interface {
	Outbound

	// CallOneway sends the given request through this transport and returns an
	// ack.
	//
	// This MUST NOT be called before Start() has been called successfully. This
	// MAY panic if called without calling Start(). This MUST be safe to call
	// concurrently.
	CallOneway(ctx context.Context, request *Request) (Ack, error)
}

// Outbounds encapsulates outbound types for a service
type Outbounds struct {
	Unary  UnaryOutbound
	Oneway OnewayOutbound
}
