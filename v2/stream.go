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

	"go.uber.org/yarpc/v2/yarpcerror"
)

// ServerStreamOption are options to configure a ServerStream.
type ServerStreamOption interface {
	unimplemented()
}

// NewServerStream will create a new ServerStream.
func NewServerStream(s Stream, _ ...ServerStreamOption) (*ServerStream, error) {
	if s == nil {
		return nil, yarpcerror.InvalidArgumentErrorf("non-nil Stream is required")
	}
	return &ServerStream{stream: s}, nil
}

// ServerStream represents the Server API of interacting with a Stream.
type ServerStream struct {
	stream Stream
}

// Context returns the context for the stream.
func (s *ServerStream) Context() context.Context {
	return s.stream.Context()
}

// Request contains all the metadata about the request.
func (s *ServerStream) Request() *Request {
	return s.stream.Request()
}

// SendMessage sends a request over the stream. It blocks until the message
// has been sent.  In certain implementations, the timeout on the context
// will be used to timeout the request.
func (s *ServerStream) SendMessage(ctx context.Context, msg *Buffer) error {
	return s.stream.SendMessage(ctx, msg)
}

// ReceiveMessage blocks until a message is received from the connection. It
// returns an io.Reader with the contents of the message.
func (s *ServerStream) ReceiveMessage(ctx context.Context) (*Buffer, error) {
	return s.stream.ReceiveMessage(ctx)
}

// ClientStreamOption are options for configuring a ClientStream.
type ClientStreamOption interface {
	unimplemented()
}

// NewClientStream will create a new ClientStream.
func NewClientStream(s StreamCloser, _ ...ClientStreamOption) (*ClientStream, error) {
	if s == nil {
		return nil, yarpcerror.InvalidArgumentErrorf("non-nil StreamCloser is required")
	}
	return &ClientStream{stream: s}, nil
}

// ClientStream represents the Client API of interacting with a Stream.
type ClientStream struct {
	stream StreamCloser
}

// Context returns the context for the stream.
func (s *ClientStream) Context() context.Context {
	return s.stream.Context()
}

// Request contains all the metadata about the request.
func (s *ClientStream) Request() *Request {
	return s.stream.Request()
}

// SendMessage sends a request over the stream. It blocks until the message
// has been sent.  In certain implementations, the timeout on the context
// will be used to timeout the request.
func (s *ClientStream) SendMessage(ctx context.Context, msg *Buffer) error {
	return s.stream.SendMessage(ctx, msg)
}

// ReceiveMessage blocks until a message is received from the connection. It
// returns an io.Reader with the contents of the message.
func (s *ClientStream) ReceiveMessage(ctx context.Context) (*Buffer, error) {
	return s.stream.ReceiveMessage(ctx)
}

// Close will close the connection. It blocks until the server has
// acknowledged the close. In certain implementations, the timeout on the
// context will be used to timeout the request. If the server timed out the
// connection will be forced closed by the client.
func (s *ClientStream) Close(ctx context.Context) error {
	return s.stream.Close(ctx)
}

// StreamOption is an option that may be passed in at
// streaming function call sites.
type StreamOption interface {
	unimplemented()
}

// Stream is an interface for interacting with a stream.
type Stream interface {
	// Context returns the context for the stream.
	Context() context.Context

	// Request contains all the metadata about the request.
	Request() *Request

	// SendMessage sends a request over the stream. It blocks until the message
	// has been sent.  In certain implementations, the timeout on the context
	// will be used to timeout the request.
	SendMessage(context.Context, *Buffer) error

	// ReceiveMessage blocks until a message is received from the connection. It
	// returns a yarpc.Buffer with the contents of the message.
	ReceiveMessage(context.Context) (*Buffer, error)
}

// StreamCloser represents an API of interacting with a Stream that is
// closable.
type StreamCloser interface {
	Stream

	// Close will close the connection. It blocks until the server has
	// acknowledged the close. The provided context controls the timeout for
	// this operation if the implementation supports it. If the server timed out
	// the connection will be forced closed by the client.
	Close(context.Context) error
}