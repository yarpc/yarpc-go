// Copyright (c) 2019 Uber Technologies, Inc.
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

package observability

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

var _ transport.StreamCloser = (*streamWrapper)(nil)

type streamWrapper struct {
	call   call
	stream transport.StreamCloser
}

func newClientStreamWrapper(call call, stream transport.StreamCloser) transport.StreamCloser {
	return &streamWrapper{
		call:   call,
		stream: stream,
	}
}

func newServerStreamWrapper(call call, stream transport.Stream) transport.Stream {
	return &streamWrapper{
		call:   call,
		stream: contextCloser{stream},
	}
}

func (s *streamWrapper) Context() context.Context {
	return s.stream.Context()
}

func (s *streamWrapper) Request() *transport.StreamRequest {
	return s.stream.Request()
}

func (s *streamWrapper) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	return s.stream.SendMessage(ctx, msg)
}

func (s *streamWrapper) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return s.stream.ReceiveMessage(ctx)
}

func (s *streamWrapper) Close(ctx context.Context) error {
	return s.stream.Close(ctx)
}

// This is a light wrapper so that we can re-use the same methods for
// instrumenting observaiblity. The transport.ServerStream does not have a
// Close(ctx) method, unlike the transport.ClientStream.
type contextCloser struct {
	transport.Stream
}

func (c contextCloser) Close(ctx context.Context) error {
	return nil
}
