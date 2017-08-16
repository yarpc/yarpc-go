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

// ServerStream represents the Server API of interacting with a Stream.
type ServerStream interface {
	BaseStream
}

// ClientStream represents the Client API of interacting with a Stream.
type ClientStream interface {
	// Close implements the io.Closer interface.  It closes the stream
	// connection from the client side.
	Close() error

	BaseStream
}

// BaseStream is an interface for interacting with a stream.
// TODO Should the Send/Recv functions pass around bytes/interface{} instead of
// io.Readers?
type BaseStream interface {
	// Context returns the context for the stream.
	Context() context.Context

	// Request contains all the metadata about the request (without the body)
	// TODO Don't use the transport request here, we should have another piece
	// of info here to hold the connection metadata (basically the headers)
	Request() *Request

	// SendMsg sends a request over the stream. It blocks until the message
	// has been sent.
	SendMsg(m io.Reader) error

	// RecvMsg blocks until a message is received from the connection. It
	// returns an io.Reader with the contents of the message.
	// TODO, should this return a ReadCloser so we can use bufferpooling?
	RecvMsg() (io.Reader, error)
}
