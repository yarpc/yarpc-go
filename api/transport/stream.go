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

import (
	"context"
	"io"
)

// StreamRequest is used to establish a streaming request.  It contains basic
// Metadata information, and any Stream-specfic configuration for creating a
// stream.
type StreamRequest struct {
	Meta *RequestMeta
}

// StreamResponse is the information returned at the end of a stream connection.
// It contains basic Metadata information, and any Stream-specfic configuration
// for ending a stream.
type StreamResponse struct {
	Meta *ResponseMeta
}

// ServerStream represents the Server API of interacting with a Stream.
type ServerStream interface {
	Stream

	// SetResponse sets the stream response metadata for the stream before
	// the stream has been stopped.  Depending on the implementation this will
	// be propagated back to the client.
	SetResponse(*StreamResponse)
}

// ClientStream represents the Client API of interacting with a Stream.
type ClientStream interface {
	Stream

	// Response returns the StreamResponse that was set by the server when the
	// stream was closed.  It will return nil if it was called before the Close
	// was called, or before one of the SendMessage/ReceiveMessage functions
	// returned an error.
	Response() *StreamResponse

	// Close will close the connection. It blocks until the server has
	// acknowledged the close. In certain implementations, the timeout on the
	// context will be used to timeout the request. If the server timed out the
	// connection will be forced closed by the client.
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
