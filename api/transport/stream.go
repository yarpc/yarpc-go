// Copyright (c) 2020 Uber Technologies, Inc.
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
	"io"

	"go.uber.org/yarpc/yarpcerrors"
)

// StreamRequest represents a streaming request.  It contains basic stream
// metadata.
type StreamRequest struct {
	Meta *RequestMeta
}

// ServerStreamOption are options to configure a ServerStream.
// There are no current ServerStreamOptions implemented.
type ServerStreamOption interface {
	unimplemented()
}

// NewServerStream will create a new ServerStream.
// The Stream can implement StreamHeadersWriter if the underlying transport
// supports stream headers.
func NewServerStream(s Stream, options ...ServerStreamOption) (*ServerStream, error) {
	if s == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("non-nil stream is required")
	}
	return &ServerStream{stream: s}, nil
}

// ServerStream represents the Server API of interacting with a Stream.
type ServerStream struct {
	stream Stream
}

// StreamHeadersWriter is the interface for sending stream headers.
type StreamHeadersWriter interface {
	SendHeaders(headers Headers) error
}

// Context returns the context for the stream.
func (s *ServerStream) Context() context.Context {
	return s.stream.Context()
}

// Request contains all the metadata about the request.
func (s *ServerStream) Request() *StreamRequest {
	return s.stream.Request()
}

// SendMessage sends a request over the stream. It blocks until the message
// has been sent.  In certain implementations, the timeout on the context
// will be used to timeout the request.
func (s *ServerStream) SendMessage(ctx context.Context, msg *StreamMessage) error {
	return s.stream.SendMessage(ctx, msg)
}

// ReceiveMessage blocks until a message is received from the connection. It
// returns an io.Reader with the contents of the message.
func (s *ServerStream) ReceiveMessage(ctx context.Context) (*StreamMessage, error) {
	return s.stream.ReceiveMessage(ctx)
}

// SendHeaders sends the one-time response headers to an initial stream connect.
// It fails if called multiple times.
func (s *ServerStream) SendHeaders(headers Headers) error {
	if w, ok := s.stream.(StreamHeadersWriter); ok {
		return w.SendHeaders(headers)
	}
	return yarpcerrors.UnimplementedErrorf("transport does not support stream headers")
}

// ClientStreamOption is an option for configuring a client stream.
// There are no current ClientStreamOptions implemented.
type ClientStreamOption interface {
	unimplemented()
}

// NewClientStream will create a new ClientStream.
// The StreamCloser can implement StreamMessageReader if the underlying transport
// supports stream headers.
func NewClientStream(s StreamCloser, options ...ClientStreamOption) (*ClientStream, error) {
	if s == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("non-nil stream with close is required")
	}
	return &ClientStream{stream: s}, nil
}

// ClientStream represents the Client API of interacting with a Stream.
type ClientStream struct {
	stream StreamCloser
}

// StreamHeadersReader is the interface for reading stream headers.
type StreamHeadersReader interface {
	Headers() (Headers, error)
}

// Context returns the context for the stream.
func (s *ClientStream) Context() context.Context {
	return s.stream.Context()
}

// Request contains all the metadata about the request.
func (s *ClientStream) Request() *StreamRequest {
	return s.stream.Request()
}

// SendMessage sends a request over the stream. It blocks until the message
// has been sent.  In certain implementations, the timeout on the context
// will be used to timeout the request.
func (s *ClientStream) SendMessage(ctx context.Context, msg *StreamMessage) error {
	return s.stream.SendMessage(ctx, msg)
}

// ReceiveMessage blocks until a message is received from the connection. It
// returns an io.Reader with the contents of the message.
func (s *ClientStream) ReceiveMessage(ctx context.Context) (*StreamMessage, error) {
	return s.stream.ReceiveMessage(ctx)
}

// Close will close the connection. It blocks until the server has
// acknowledged the close. In certain implementations, the timeout on the
// context will be used to timeout the request. If the server timed out the
// connection will be forced closed by the client.
func (s *ClientStream) Close(ctx context.Context) error {
	return s.stream.Close(ctx)
}

// Headers returns the initial stream response headers received from the server if there
// are any. It blocks if the headers are not available.
func (s *ClientStream) Headers() (Headers, error) {
	if r, ok := s.stream.(StreamHeadersReader); ok {
		return r.Headers()
	}
	return NewHeaders(), yarpcerrors.UnimplementedErrorf("transport does not support stream headers")
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

// Stream is an interface for interacting with a stream.
type Stream interface {
	// Context returns the context for the stream.
	Context() context.Context

	// Request contains all the metadata about the request.
	Request() *StreamRequest

	// SendMessage sends a request over the stream. It blocks until the message
	// has been sent.  In certain implementations, the timeout on the context
	// will be used to timeout the request.
	SendMessage(context.Context, *StreamMessage) error

	// ReceiveMessage blocks until a message is received from the connection. It
	// returns an io.Reader with the contents of the message.
	ReceiveMessage(context.Context) (*StreamMessage, error)
}

// StreamMessage represents information that can be read off of an individual
// message in the stream.
type StreamMessage struct {
	Body io.ReadCloser
}
