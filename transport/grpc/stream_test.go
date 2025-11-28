// Copyright (c) 2026 Uber Technologies, Inc.
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

package grpc

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/status"
)

func TestToYARPCStreamErrorWithInvalidGrpcCode(t *testing.T) {
	st := status.New(100, "test")
	err := toYARPCStreamError(st.Err())
	yarpcstatus := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeUnknown, yarpcstatus.Code())
	assert.Equal(t, yarpcstatus.Message(), "test")
	assert.Nil(t, yarpcstatus.Details())
}

func TestServerStreamSendMessage(t *testing.T) {
	t.Run("passes mem.Buffer to gRPC without conversion", func(t *testing.T) {
		data := []byte("test data")
		var receivedMemBuffer bool
		buffer := newTestMemBuffer(&data, func() {})

		mockStream := &mockGRPCServerStream{
			sendMsgFunc: func(m any) error {
				_, receivedMemBuffer = m.(mem.Buffer)
				return nil
			},
		}

		ss := &serverStream{
			ctx:    context.Background(),
			stream: mockStream,
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{},
			},
		}

		msg := &transport.StreamMessage{
			Body:     buffer,
			BodySize: len(data),
		}

		err := ss.SendMessage(context.Background(), msg)
		assert.NoError(t, err)
		assert.True(t, receivedMemBuffer, "gRPC should receive mem.Buffer, not []byte")
	})

	t.Run("calls Close on body after SendMsg completes", func(t *testing.T) {
		data := []byte("test data")
		var closeCalledBeforeSend bool
		var closeCalled bool

		buffer := newTestMemBuffer(&data, func() { closeCalled = true })

		mockStream := &mockGRPCServerStream{
			sendMsgFunc: func(m any) error {
				closeCalledBeforeSend = closeCalled
				return nil
			},
		}

		ss := &serverStream{
			ctx:    context.Background(),
			stream: mockStream,
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{},
			},
		}

		msg := &transport.StreamMessage{
			Body:     buffer,
			BodySize: len(data),
		}

		err := ss.SendMessage(context.Background(), msg)
		assert.NoError(t, err)
		assert.False(t, closeCalledBeforeSend, "Close() must not be called before SendMsg")
		assert.True(t, closeCalled, "Close() must be called after SendMsg")
	})

	t.Run("handles empty buffer", func(t *testing.T) {
		emptyData := []byte{}
		var closeCalled bool
		buffer := newTestMemBuffer(&emptyData, func() { closeCalled = true })

		mockStream := &mockGRPCServerStream{
			sendMsgFunc: func(m any) error {
				buf, ok := m.(mem.Buffer)
				assert.True(t, ok, "Should receive mem.Buffer")
				assert.Equal(t, 0, buf.Len(), "Empty buffer should have zero length")
				return nil
			},
		}

		ss := &serverStream{
			ctx:    context.Background(),
			stream: mockStream,
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{},
			},
		}

		msg := &transport.StreamMessage{
			Body:     buffer,
			BodySize: 0,
		}

		err := ss.SendMessage(context.Background(), msg)
		assert.NoError(t, err)
		assert.True(t, closeCalled, "Close() should be called even for empty buffer")
	})
}

func TestClientStreamSendMessage(t *testing.T) {
	t.Run("passes mem.Buffer to gRPC without conversion", func(t *testing.T) {
		data := []byte("client test")
		var receivedMemBuffer bool
		buffer := newTestMemBuffer(&data, func() {})

		mockStream := &mockGRPCClientStream{
			sendMsgFunc: func(m any) error {
				_, receivedMemBuffer = m.(mem.Buffer)
				return nil
			},
		}

		cs := &clientStream{
			ctx:    context.Background(),
			stream: mockStream,
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{},
			},
		}

		msg := &transport.StreamMessage{
			Body:     buffer,
			BodySize: len(data),
		}

		err := cs.SendMessage(context.Background(), msg)
		assert.NoError(t, err)
		assert.True(t, receivedMemBuffer, "gRPC should receive mem.Buffer, not []byte")
	})

	t.Run("calls Close on body after SendMsg completes", func(t *testing.T) {
		data := []byte("client test")
		var closeCalledBeforeSend bool
		var closeCalled bool

		buffer := newTestMemBuffer(&data, func() { closeCalled = true })

		mockStream := &mockGRPCClientStream{
			sendMsgFunc: func(m any) error {
				closeCalledBeforeSend = closeCalled
				return nil
			},
		}

		cs := &clientStream{
			ctx:    context.Background(),
			stream: mockStream,
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{},
			},
		}

		msg := &transport.StreamMessage{
			Body:     buffer,
			BodySize: len(data),
		}

		err := cs.SendMessage(context.Background(), msg)
		assert.NoError(t, err)
		assert.False(t, closeCalledBeforeSend, "Close() must not be called before SendMsg")
		assert.True(t, closeCalled, "Close() must be called after SendMsg")
	})
}

type mockGRPCServerStream struct {
	grpc.ServerStream
	sendMsgFunc func(m any) error
}

func (m *mockGRPCServerStream) SendMsg(msg any) error {
	if m.sendMsgFunc != nil {
		return m.sendMsgFunc(msg)
	}
	return nil
}

type mockGRPCClientStream struct {
	grpc.ClientStream
	sendMsgFunc func(m any) error
}

func (m *mockGRPCClientStream) SendMsg(msg any) error {
	if m.sendMsgFunc != nil {
		return m.sendMsgFunc(msg)
	}
	return nil
}

func newTestMemBuffer(data *[]byte, onClose func()) *testMemBuffer {
	buffer := mem.NewBuffer(data, &testMemBufferPool{cleanup: func() {}})
	return &testMemBuffer{buffer: buffer, onClose: onClose}
}

type testMemBuffer struct {
	buffer  mem.Buffer
	onClose func()
}

func (t *testMemBuffer) Buffer() mem.Buffer {
	return t.buffer
}

func (t *testMemBuffer) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (t *testMemBuffer) Close() error {
	if t.onClose != nil {
		t.onClose()
	}
	t.buffer.Free()
	return nil
}

type testMemBufferPool struct {
	cleanup func()
}

func (m *testMemBufferPool) Get(length int) *[]byte { return nil }
func (m *testMemBufferPool) Put(buf *[]byte) {
	if m.cleanup != nil {
		m.cleanup()
	}
}
