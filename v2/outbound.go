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

import "context"

// UnaryTransportOutbound is a transport that knows how to send unary requests
// for procedure calls.
type UnaryTransportOutbound interface {
	// Call sends the given request through this transport and returns its
	// response.
	//
	// This MUST NOT be called before Start() has been called successfully. This
	// MAY panic if called without calling Start(). This MUST be safe to call
	// concurrently.
	Call(context.Context, *Request, *Buffer) (*Response, *Buffer, error)
}

// StreamTransportOutbound is a transport that knows how to send stream requests
// for procedure calls.
type StreamTransportOutbound interface {
	// CallStream creates a stream connection based on the metadata in the
	// request passed in.  If there is a timeout on the context, this timeout
	// is for establishing a connection, and not for the lifetime of the stream.
	CallStream(context.Context, *Request) (*ClientStream, error)
}
