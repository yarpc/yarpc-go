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

package transport

import "context"

// Outbound is the common interface for all outbounds
type Outbound interface {
	Lifecycle

	// Transports returns the transports that used by this outbound, so they
	// can be collected for lifecycle management, typically by a Dispatcher.
	//
	// Though most outbounds only use a single transport, composite outbounds
	// may use multiple transport protocols, particularly for shadowing traffic
	// across multiple transport protocols during a transport protocol
	// migration.
	Transports() []Transport
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

// StreamOutbound is a transport that knows how to send stream requests for
// procedure calls.
type StreamOutbound interface {
	Outbound

	// CallStream creates a stream connection based on the metadata in the
	// request passed in.  If there is a timeout on the context, this timeout
	// is for establishing a connection, and not for the lifetime of the stream.
	CallStream(ctx context.Context, request *StreamRequest) (*ClientStream, error)
}

// Outbounds encapsulates the outbound specification for a service.
//
// This includes the service name that will be used for outbound requests as
// well as the Outbound that will be used to transport the request.  The
// outbound will be one of Unary and Oneway.
type Outbounds struct {
	ServiceName string

	// If set, this is the unary outbound which sends a request and waits for
	// the response.
	Unary UnaryOutbound

	// If set, this is the oneway outbound which sends the request and
	// continues once the message has been delivered.
	Oneway OnewayOutbound

	// If set, this is the stream outbound which creates a ClientStream that can
	// be used to continuously send/recv requests over the connection.
	Stream StreamOutbound
}
