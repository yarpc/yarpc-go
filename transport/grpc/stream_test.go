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
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

// TestStreamReceiveMessage verifies bytesBody wiring: both server and client
// receive paths wrap payloads in bytesBody (exposes Bytes() for zero-copy).
// The actual fast-path dispatch is tested in TestUnmarshalFastPath.
func TestStreamReceiveMessage(t *testing.T) {
	t.Run("server stream wraps body in bytesBody", func(t *testing.T) {
		payload := []byte("server receive test")

		mockStream := &mockGRPCServerStream{
			recvMsgFunc: func(m any) error {
				ptr := m.(*[]byte)
				*ptr = payload
				return nil
			},
		}

		ss := &serverStream{
			ctx:    context.Background(),
			stream: mockStream,
			req:    &transport.StreamRequest{Meta: &transport.RequestMeta{}},
		}

		msg, err := ss.ReceiveMessage(context.Background())
		assert.NoError(t, err)

		br, ok := msg.Body.(interface{ Bytes() []byte })
		assert.True(t, ok, "Body should implement Bytes() []byte")
		assert.Equal(t, payload, br.Bytes())
		assert.Equal(t, len(payload), msg.BodySize)
	})

	t.Run("client stream wraps body in bytesBody", func(t *testing.T) {
		payload := []byte("client receive test")

		mockStream := &mockGRPCClientStream{
			recvMsgFunc: func(m any) error {
				ptr := m.(*[]byte)
				*ptr = payload
				return nil
			},
		}

		cs := &clientStream{
			ctx:    context.Background(),
			stream: mockStream,
			req:    &transport.StreamRequest{Meta: &transport.RequestMeta{}},
		}

		msg, err := cs.ReceiveMessage(context.Background())
		assert.NoError(t, err)

		br, ok := msg.Body.(interface{ Bytes() []byte })
		assert.True(t, ok, "Body should implement Bytes() []byte")
		assert.Equal(t, payload, br.Bytes())
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
	recvMsgFunc func(m any) error
}

func (m *mockGRPCServerStream) SendMsg(msg any) error {
	if m.sendMsgFunc != nil {
		return m.sendMsgFunc(msg)
	}
	return nil
}

func (m *mockGRPCServerStream) RecvMsg(msg any) error {
	if m.recvMsgFunc != nil {
		return m.recvMsgFunc(msg)
	}
	return nil
}

type mockGRPCClientStream struct {
	grpc.ClientStream
	sendMsgFunc func(m any) error
	recvMsgFunc func(m any) error
	closeSendFn func() error
}

func (m *mockGRPCClientStream) SendMsg(msg any) error {
	if m.sendMsgFunc != nil {
		return m.sendMsgFunc(msg)
	}
	return nil
}

func (m *mockGRPCClientStream) RecvMsg(msg any) error {
	if m.recvMsgFunc != nil {
		return m.recvMsgFunc(msg)
	}
	return io.EOF
}

func (m *mockGRPCClientStream) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
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

// newTestClientStream builds a clientStream backed by mock with a tracked release callback.
// releaseCalls is incremented each time the release (decStreamCount) callback fires.
func newTestClientStream(mock *mockGRPCClientStream) (cs *clientStream, releaseCalls *int32) {
	var calls int32
	span := opentracing.GlobalTracer().StartSpan("test")
	cs = newClientStream(
		context.Background(),
		&transport.StreamRequest{Meta: &transport.RequestMeta{}},
		mock,
		span,
		func(error) { atomic.AddInt32(&calls, 1) },
	)
	return cs, &calls
}

// TestStreamTerminationDecrementsStreamCount verifies that the release callback
// (which decrements grpcClientConnWrapper.streamCount) is called exactly once
// for every way a client stream can terminate.
func TestStreamTerminationDecrementsStreamCount(t *testing.T) {
	t.Parallel()

	t.Run("normal close via Close()", func(t *testing.T) {
		t.Parallel()
		cs, calls := newTestClientStream(&mockGRPCClientStream{})
		_ = cs.Close(context.Background())
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must be called exactly once on normal close")
	})

	t.Run("caller context cancellation causes ReceiveMessage error", func(t *testing.T) {
		t.Parallel()
		cs, calls := newTestClientStream(&mockGRPCClientStream{
			recvMsgFunc: func(any) error { return context.Canceled },
		})
		_, err := cs.ReceiveMessage(context.Background())
		require.Error(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must be called when context is cancelled")
	})

	t.Run("server-side abort via gRPC status error", func(t *testing.T) {
		t.Parallel()
		serverErr := status.Error(codes.Aborted, "server aborted")
		cs, calls := newTestClientStream(&mockGRPCClientStream{
			recvMsgFunc: func(any) error { return serverErr },
		})
		_, err := cs.ReceiveMessage(context.Background())
		require.Error(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must be called on server-side abort")
	})

	t.Run("TCP connection dropped - io.ErrUnexpectedEOF", func(t *testing.T) {
		t.Parallel()
		cs, calls := newTestClientStream(&mockGRPCClientStream{
			recvMsgFunc: func(any) error { return io.ErrUnexpectedEOF },
		})
		_, err := cs.ReceiveMessage(context.Background())
		require.Error(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must be called on TCP drop")
	})

	t.Run("send error also triggers release", func(t *testing.T) {
		t.Parallel()
		cs, calls := newTestClientStream(&mockGRPCClientStream{
			sendMsgFunc: func(any) error { return io.ErrUnexpectedEOF },
		})
		err := cs.SendMessage(context.Background(), &transport.StreamMessage{
			Body: io.NopCloser(bytes.NewReader([]byte("msg"))),
		})
		require.Error(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must be called on send error")
	})

	t.Run("release called at most once even if multiple paths fire", func(t *testing.T) {
		t.Parallel()
		cs, calls := newTestClientStream(&mockGRPCClientStream{
			recvMsgFunc: func(any) error { return context.Canceled },
		})
		_, _ = cs.ReceiveMessage(context.Background())
		_ = cs.Close(context.Background()) // second termination — must be a no-op
		assert.Equal(t, int32(1), atomic.LoadInt32(calls),
			"release must fire exactly once regardless of how many close paths are triggered")
	})

	t.Run("GC leak - release is NOT called if stream is never read, written, or closed", func(t *testing.T) {
		t.Parallel()
		// This test documents the known limitation: if a caller obtains a stream
		// and then drops the reference without calling Close or reading/writing,
		// decStreamCount is never called because there is no finalizer on clientStream.
		// The goroutine goroutine leak detector (goleak) will surface this if there
		// are active goroutines, but the streamCount itself will not self-correct.
		_, calls := newTestClientStream(&mockGRPCClientStream{})
		// Deliberately do nothing with the stream — simulate a caller that leaks it.
		assert.Equal(t, int32(0), atomic.LoadInt32(calls),
			"streamCount is not decremented when a stream is leaked without Close/read/write — "+
				"this is a known limitation; adding a runtime.SetFinalizer would address it")
	})
}
