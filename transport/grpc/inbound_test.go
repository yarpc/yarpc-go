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
	"encoding/binary"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/http2test"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
)

func TestInboundMechanics(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	inbound := NewTransport().NewInbound(listener)

	assert.False(t, inbound.IsRunning())
	assert.Equal(t, errRouterNotSet, inbound.Start())

	inbound = NewTransport().NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))
	assert.Nil(t, inbound.Addr())
	assert.NoError(t, inbound.Start())
	assert.True(t, inbound.IsRunning())
	assert.NotNil(t, inbound.Addr())
	assert.NoError(t, inbound.Stop())
	assert.Nil(t, inbound.Addr())
}

func TestInboundIntrospection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	inbound := NewTransport().NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))

	assert.Equal(t, TransportName, inbound.Introspect().Transport, "unexpected transport name")
	assert.Equal(t, "Stopped", inbound.Introspect().State, "expected 'Stopped' state")
	assert.Empty(t, inbound.Introspect().Endpoint, "unexpected endpoint")

	require.NoError(t, inbound.Start())
	assert.Equal(t, "Started", inbound.Introspect().State, "expected 'Started' state")
	assert.NotEmpty(t, inbound.Introspect().Endpoint)
	assert.Equal(t, inbound.Addr().String(), inbound.Introspect().Endpoint, "unexpected endpoint")

	assert.NoError(t, inbound.Stop())
	assert.Equal(t, "Stopped", inbound.Introspect().State, "expected 'Stopped' state")
	assert.Empty(t, inbound.Introspect().Endpoint, "unexpected endpoint")
}

func TestInboundStartWithNumStreamWorkers(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	transport := NewTransport(NumStreamWorkers(100))
	inbound := transport.NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))

	require.NoError(t, inbound.Start())
	assert.True(t, inbound.IsRunning())
	require.NoError(t, inbound.Stop())
}

func TestInboundStartWithServerHeaderTableSize(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	transport := NewTransport(ServerHeaderTableSize(8192))
	inbound := transport.NewInbound(listener)
	inbound.SetRouter(newTestRouter(nil))

	require.NoError(t, inbound.Start())
	assert.True(t, inbound.IsRunning())
	require.NoError(t, inbound.Stop())
}

// TestInboundGracefulShutdownDeliversGOAWAYAndDrains pins the graceful
// shutdown behavior gRPC inbounds get from grpc.Server.GracefulStop, which the
// HTTP inbound's equivalent test holds HTTP/2 cleartext connections to: Stop
// delivers GOAWAY and lets in-flight streams complete before it returns.
//
// GracefulStop also implements the two-phase shutdown of RFC 9113 §6.8: a
// first GOAWAY reserving the whole stream ID space, then, after a PING round
// trip, a second one carrying the last stream actually processed.
func TestInboundGracefulShutdownDeliversGOAWAYAndDrains(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	h := newBlockingHandler()
	inbound := NewTransport().NewInbound(listener)
	inbound.SetRouter(newTestRouter([]transport.Procedure{{
		Name:        "block",
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
	}}))
	require.NoError(t, inbound.Start())
	t.Cleanup(func() { _ = inbound.Stop() })
	// Registered after Stop so it runs first: a pending call must not hold
	// GracefulStop forever when a test fails midway.
	t.Cleanup(h.releaseAll)
	addr := inbound.Addr().String()

	c := http2test.DialRawH2C(t, addr)
	c.OpenStream(t, 1, "/"+defaultServiceName+"/block", map[string]string{
		"content-type": baseContentType,
		"te":           "trailers",
		"grpc-timeout": "10S",
		CallerHeader:   "caller",
		ServiceHeader:  "service",
		EncodingHeader: string(raw.Encoding),
	}, grpcMessage("payload"))
	h.awaitEntered(t)

	var seq, stopSeq atomic.Int64
	stopErr := make(chan error, 1)
	go func() {
		err := inbound.Stop()
		stopSeq.Store(seq.Add(1))
		stopErr <- err
	}()

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	first := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, first.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(math.MaxInt32), first.LastStreamID,
		"first GOAWAY must reserve the whole stream ID space")

	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)...)
	goAwaySeq := seq.Add(1)
	second := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, second.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), second.LastStreamID, "second GOAWAY must carry the last processed stream")

	releaseSeq := seq.Add(1)
	h.releaseAll()
	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.StreamDone(1))...)
	res := http2test.ResponseOn(frames, 1)
	require.False(t, res.Reset, "in-flight stream 1 was reset with %v instead of completing", res.ResetCode)
	assert.Equal(t, "200", res.Headers[":status"], "unexpected status for in-flight stream 1")
	assert.Equal(t, "0", res.Trailers["grpc-status"], "unexpected gRPC status for in-flight stream 1")
	assert.Equal(t, grpcMessage("payload"), res.Body, "unexpected body for in-flight stream 1")

	require.NoError(t, c.Close())

	select {
	case err := <-stopErr:
		assert.NoError(t, err, "Stop must succeed once connections have drained")
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after the connection drained")
	}
	assert.Less(t, goAwaySeq, stopSeq.Load(), "Stop returned before the client observed a GOAWAY")
	assert.Less(t, releaseSeq, stopSeq.Load(), "Stop returned while stream 1 was still in flight")

	_, err = net.Dial("tcp", addr)
	assert.Error(t, err, "new connections must be refused once Stop returns")
}

// grpcMessage frames payload as an uncompressed gRPC length-prefixed message.
func grpcMessage(payload string) []byte {
	msg := make([]byte, 5, 5+len(payload))
	binary.BigEndian.PutUint32(msg[1:], uint32(len(payload)))
	return append(msg, payload...)
}

// blockingHandler is a unary handler that blocks every call until released,
// then echoes the request body back.
type blockingHandler struct {
	entered     chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

func newBlockingHandler() *blockingHandler {
	return &blockingHandler{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
}

func (h *blockingHandler) Handle(_ context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	h.entered <- struct{}{}
	<-h.release
	_, err := io.Copy(resw, req.Body)
	return err
}

func (h *blockingHandler) releaseAll() { h.releaseOnce.Do(func() { close(h.release) }) }

func (h *blockingHandler) awaitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-h.entered:
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "handler was not called")
	}
}
