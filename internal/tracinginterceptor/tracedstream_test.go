// Copyright (c) 2025 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software are
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF TORT, TORT OR OTHERWISE, ARISING FROM,
// THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package tracinginterceptor

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

// serverStub implements transport.Stream and transport.StreamHeadersSender
type serverStub struct {
	sent transport.Headers
}

var _ transport.Stream = (*serverStub)(nil)
var _ transport.StreamHeadersSender = (*serverStub)(nil)

func (s *serverStub) Context() context.Context {
	return context.Background()
}
func (s *serverStub) Request() *transport.StreamRequest {
	return &transport.StreamRequest{}
}
func (s *serverStub) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	return nil
}
func (s *serverStub) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return nil, nil
}
func (s *serverStub) SendHeaders(h transport.Headers) error {
	s.sent = h
	return nil
}

func TestTracedServerStream_SendHeaders(t *testing.T) {
	tracer := mocktracer.New()
	span := tracer.StartSpan("test-span")

	stub := &serverStub{}
	raw, err := transport.NewServerStream(stub)
	require.NoError(t, err)

	tracedRaw := &tracedServerStream{
		serverStream: raw,
		span:         span,
	}
	wrapped, err := transport.NewServerStream(tracedRaw)
	require.NoError(t, err)

	want := transport.Headers{}.With("foo", "bar")
	err = wrapped.SendHeaders(want)
	require.NoError(t, err)

	assert.Equal(t, want, stub.sent)
}
