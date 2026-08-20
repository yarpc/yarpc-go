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

package tracinginterceptor

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

type testContextKey struct{}

func newTestStreamRequest() *transport.StreamRequest {
	return &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    "test-caller",
			Service:   "test-service",
			Procedure: "test-procedure",
		},
	}
}

// newSuccessClientStream returns a mock whose every operation succeeds.
func newSuccessClientStream(ctx context.Context) *mockClientStream {
	return &mockClientStream{
		ctx: ctx,
		req: newTestStreamRequest(),
		sendMsg: func(context.Context, *transport.StreamMessage) error {
			return nil
		},
		receiveMsg: func(context.Context) (*transport.StreamMessage, error) {
			return &transport.StreamMessage{}, nil
		},
		close: func(context.Context) error {
			return nil
		},
		headers: func() (transport.Headers, error) {
			return transport.Headers{}, nil
		},
	}
}

// newSuccessServerStream returns a mock whose every operation succeeds.
func newSuccessServerStream(ctx context.Context) *mockServerStream {
	return &mockServerStream{
		ctx: ctx,
		req: newTestStreamRequest(),
		sendMsg: func(context.Context, *transport.StreamMessage) error {
			return nil
		},
		receiveMsg: func(context.Context) (*transport.StreamMessage, error) {
			return &transport.StreamMessage{}, nil
		},
		sendHeaders: func(transport.Headers) error {
			return nil
		},
	}
}

func newTracedOTelClientStream(t *testing.T, m *mockClientStream) (*tracedOTelClientStream, *tracetest.SpanRecorder) {
	t.Helper()
	tp, sr := newTestProvider(t)
	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	wrapped, err := transport.NewClientStream(m)
	require.NoError(t, err)
	return &tracedOTelClientStream{clientStream: wrapped, span: span}, sr
}

func newTracedOTelServerStream(t *testing.T, m *mockServerStream) (*tracedOTelServerStream, *tracetest.SpanRecorder) {
	t.Helper()
	tp, sr := newTestProvider(t)
	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	wrapped, err := transport.NewServerStream(m)
	require.NoError(t, err)
	return &tracedOTelServerStream{serverStream: wrapped, span: span}, sr
}

func spanAttrValue(t *testing.T, s sdktrace.ReadOnlySpan, key string) attribute.Value {
	t.Helper()
	for _, a := range s.Attributes() {
		if string(a.Key) == key {
			return a.Value
		}
	}
	t.Fatalf("attribute %q not present on span", key)
	return attribute.Value{}
}

func TestTracedOTelClientStreamDelegatesAndKeepsSpanOpen(t *testing.T) {
	ctx := context.Background()
	mock := newSuccessClientStream(ctx)
	tracedStream, sr := newTracedOTelClientStream(t, mock)

	assert.Equal(t, ctx, tracedStream.Context())
	assert.Equal(t, mock.req, tracedStream.Request())

	require.NoError(t, tracedStream.SendMessage(ctx, &transport.StreamMessage{}))

	msg, err := tracedStream.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.NotNil(t, msg)

	_, err = tracedStream.Headers()
	require.NoError(t, err)

	assert.Empty(t, sr.Ended(), "span must stay open while the stream is healthy")
}

func TestTracedOTelClientStreamEndsSpan(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name       string
		setup      func(*mockClientStream)
		call       func(*tracedOTelClientStream) error
		wantErr    error
		wantStatus codes.Code
	}{
		{
			name: "send message error",
			setup: func(m *mockClientStream) {
				m.sendMsg = func(context.Context, *transport.StreamMessage) error { return boom }
			},
			call: func(s *tracedOTelClientStream) error {
				return s.SendMessage(context.Background(), &transport.StreamMessage{})
			},
			wantErr:    boom,
			wantStatus: codes.Error,
		},
		{
			name: "receive message error",
			setup: func(m *mockClientStream) {
				m.receiveMsg = func(context.Context) (*transport.StreamMessage, error) { return nil, boom }
			},
			call: func(s *tracedOTelClientStream) error {
				_, err := s.ReceiveMessage(context.Background())
				return err
			},
			wantErr:    boom,
			wantStatus: codes.Error,
		},
		{
			name: "close error",
			setup: func(m *mockClientStream) {
				m.close = func(context.Context) error { return boom }
			},
			call: func(s *tracedOTelClientStream) error {
				return s.Close(context.Background())
			},
			wantErr:    boom,
			wantStatus: codes.Error,
		},
		{
			name: "headers error",
			setup: func(m *mockClientStream) {
				m.headers = func() (transport.Headers, error) { return transport.Headers{}, boom }
			},
			call: func(s *tracedOTelClientStream) error {
				_, err := s.Headers()
				return err
			},
			wantErr:    boom,
			wantStatus: codes.Error,
		},
		{
			name: "clean close",
			call: func(s *tracedOTelClientStream) error {
				return s.Close(context.Background())
			},
			wantStatus: codes.Unset,
		},
		{
			name: "end of stream is not a failure",
			setup: func(m *mockClientStream) {
				m.receiveMsg = func(context.Context) (*transport.StreamMessage, error) { return nil, io.EOF }
			},
			call: func(s *tracedOTelClientStream) error {
				_, err := s.ReceiveMessage(context.Background())
				return err
			},
			wantErr:    io.EOF,
			wantStatus: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newSuccessClientStream(context.Background())
			if tt.setup != nil {
				tt.setup(mock)
			}
			tracedStream, sr := newTracedOTelClientStream(t, mock)

			err := tt.call(tracedStream)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1, "the span must be ended exactly once")
			assert.Equal(t, tt.wantStatus, spans[0].Status().Code)
		})
	}
}

func TestTracedOTelClientStreamEndsSpanOnlyOnce(t *testing.T) {
	ctx := context.Background()
	mock := newSuccessClientStream(ctx)
	mock.receiveMsg = func(context.Context) (*transport.StreamMessage, error) { return nil, io.EOF }
	tracedStream, sr := newTracedOTelClientStream(t, mock)

	_, err := tracedStream.ReceiveMessage(ctx)
	assert.ErrorIs(t, err, io.EOF)

	// A Close following end-of-stream is the normal shutdown path, and a late
	// failure must not reopen or re-end an already finished span.
	require.NoError(t, tracedStream.Close(ctx))
	mock.sendMsg = func(context.Context, *transport.StreamMessage) error { return errors.New("late failure") }
	assert.Error(t, tracedStream.SendMessage(ctx, &transport.StreamMessage{}))

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code,
		"a failure after the span ended must not change its status")
}

func TestTracedOTelServerStreamDelegatesAndKeepsSpanOpen(t *testing.T) {
	ctx := context.Background()
	mock := newSuccessServerStream(ctx)
	tracedStream, sr := newTracedOTelServerStream(t, mock)

	assert.Equal(t, mock.req, tracedStream.Request())

	require.NoError(t, tracedStream.SendMessage(ctx, &transport.StreamMessage{}))

	msg, err := tracedStream.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.NotNil(t, msg)

	require.NoError(t, tracedStream.SendHeaders(transport.Headers{}))

	assert.Empty(t, sr.Ended(), "span must stay open while the stream is healthy")
}

func TestTracedOTelServerStreamContext(t *testing.T) {
	underlying := context.Background()
	enriched := context.WithValue(underlying, testContextKey{}, "v")

	t.Run("falls back to the stream context", func(t *testing.T) {
		tracedStream, _ := newTracedOTelServerStream(t, newSuccessServerStream(underlying))
		assert.Equal(t, underlying, tracedStream.Context())
	})

	t.Run("prefers the enriched context", func(t *testing.T) {
		tracedStream, _ := newTracedOTelServerStream(t, newSuccessServerStream(underlying))
		tracedStream.ctx = enriched
		assert.Equal(t, enriched, tracedStream.Context())
	})
}

func TestTracedOTelServerStreamEndsSpan(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(*mockServerStream)
		call       func(*tracedOTelServerStream) error
		wantStatus codes.Code
	}{
		{
			name: "send message error",
			setup: func(m *mockServerStream) {
				m.sendMsg = func(context.Context, *transport.StreamMessage) error {
					return yarpcerrors.Newf(yarpcerrors.CodeInternal, "send failed")
				}
			},
			call: func(s *tracedOTelServerStream) error {
				return s.SendMessage(ctx, &transport.StreamMessage{})
			},
			wantStatus: codes.Error,
		},
		{
			name: "receive message error",
			setup: func(m *mockServerStream) {
				m.receiveMsg = func(context.Context) (*transport.StreamMessage, error) {
					return nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "receive failed")
				}
			},
			call: func(s *tracedOTelServerStream) error {
				_, err := s.ReceiveMessage(ctx)
				return err
			},
			wantStatus: codes.Error,
		},
		{
			name: "send headers error",
			setup: func(m *mockServerStream) {
				m.sendHeaders = func(transport.Headers) error {
					return yarpcerrors.Newf(yarpcerrors.CodeInternal, "headers failed")
				}
			},
			call: func(s *tracedOTelServerStream) error {
				return s.SendHeaders(transport.Headers{})
			},
			wantStatus: codes.Error,
		},
		{
			name: "end of stream is not a failure",
			setup: func(m *mockServerStream) {
				m.receiveMsg = func(context.Context) (*transport.StreamMessage, error) { return nil, io.EOF }
			},
			call: func(s *tracedOTelServerStream) error {
				_, err := s.ReceiveMessage(ctx)
				return err
			},
			wantStatus: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newSuccessServerStream(ctx)
			tt.setup(mock)
			tracedStream, sr := newTracedOTelServerStream(t, mock)

			assert.Error(t, tt.call(tracedStream))

			spans := sr.Ended()
			require.Len(t, spans, 1, "the span must be ended exactly once")
			assert.Equal(t, tt.wantStatus, spans[0].Status().Code)
			if tt.wantStatus == codes.Error {
				assert.Equal(t, int64(yarpcerrors.CodeInternal),
					spanAttrValue(t, spans[0], rpcStatusCodeTag).AsInt64())
			}
		})
	}
}

func TestTracedOTelServerStreamEndsSpanOnlyOnce(t *testing.T) {
	ctx := context.Background()

	// End of stream on receive is routine and ends the span, so a later
	// failure is not recorded. This matches the OpenTracing server stream,
	// where HandleStream and the stream itself race to finish the same span
	// and the first one wins.
	mock := newSuccessServerStream(ctx)
	mock.receiveMsg = func(context.Context) (*transport.StreamMessage, error) { return nil, io.EOF }
	mock.sendMsg = func(context.Context, *transport.StreamMessage) error {
		return yarpcerrors.Newf(yarpcerrors.CodeInternal, "send failed")
	}
	tracedStream, sr := newTracedOTelServerStream(t, mock)

	_, err := tracedStream.ReceiveMessage(ctx)
	require.ErrorIs(t, err, io.EOF)
	assert.Error(t, tracedStream.SendMessage(ctx, &transport.StreamMessage{}))

	// HandleStream ends the same span; the duplicate End is a no-op.
	tracedStream.span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
}
