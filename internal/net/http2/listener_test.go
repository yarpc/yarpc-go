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

package http2

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhttp2 "golang.org/x/net/http2"
)

// recordingHelper is a [ShutdownHelper] that records the connections it is
// asked to track, and those it stops tracking.
type recordingHelper struct {
	noopHelper
	refuse bool // does not track connections

	mu        sync.Mutex
	tracked   []net.Conn
	untracked []net.Conn
}

func (h *recordingHelper) TrackConnection(conn net.Conn) func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tracked = append(h.tracked, conn)
	if h.refuse {
		return nil
	}
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.untracked = append(h.untracked, conn)
	}
}

func (h *recordingHelper) trackedConns() []net.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]net.Conn(nil), h.tracked...)
}

func (h *recordingHelper) untrackedConns() []net.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]net.Conn(nil), h.untracked...)
}

// stubListener is a [net.Listener] that accepts conn, or fails with err.
type stubListener struct {
	conn   net.Conn
	err    error
	closed bool
}

var _ net.Listener = (*stubListener)(nil)

func (l *stubListener) Accept() (net.Conn, error) { return l.conn, l.err }
func (l *stubListener) Addr() net.Addr            { return stubAddr("listener") }

func (l *stubListener) Close() error {
	l.closed = true
	return l.err
}

func isNoopTrackerFunc(fn ConnectionTrackerFunc) bool {
	return reflect.ValueOf(fn).Pointer() == reflect.ValueOf(noopTrackerFunc).Pointer()
}

func TestWrapListener(t *testing.T) {
	l := &stubListener{}
	helper := &recordingHelper{}

	t.Run("nil listener", func(t *testing.T) {
		assert.Nil(t, WrapListener(nil, helper))
	})

	t.Run("nil helper", func(t *testing.T) {
		assert.Same(t, l, WrapListener(l, nil), "must not wrap without a helper")
	})

	t.Run("noop helper", func(t *testing.T) {
		assert.Same(t, l, WrapListener(l, NewNoopShutdownHelper()), "must not wrap for a no-op helper")
	})

	t.Run("graceful shutdown helper", func(t *testing.T) {
		assert.IsType(t, &TrackingListener{}, WrapListener(l, NewShutdownHelper()))
	})

	t.Run("helper", func(t *testing.T) {
		got := WrapListener(l, helper)
		tl, ok := got.(*TrackingListener)
		require.True(t, ok, "must return a *TrackingListener, got %T", got)
		assert.Same(t, l, tl.listener, "must wrap the given listener")
		assert.Same(t, helper, tl.helper, "must use the given helper")
	})
}

func TestTrackingListenerGetTrackerFunc(t *testing.T) {
	t.Run("nil helper", func(t *testing.T) {
		tl := &TrackingListener{listener: &stubListener{}}
		assert.True(t, isNoopTrackerFunc(tl.getTrackerFunc()), "must not track without a helper")
	})

	t.Run("helper", func(t *testing.T) {
		helper := &recordingHelper{}
		tl := &TrackingListener{listener: &stubListener{}, helper: helper}
		fn := tl.getTrackerFunc()

		assert.Nil(t, fn(nil), "must not track a nil connection")
		assert.Empty(t, helper.trackedConns(), "must not track a nil connection")

		conn := &stubConn{}
		untrack := fn(conn)
		assert.Equal(t, []net.Conn{conn}, helper.trackedConns(), "must track through the helper")
		require.NotNil(t, untrack, "must return the helper's untrack func")
		untrack()
		assert.Equal(t, []net.Conn{conn}, helper.untrackedConns(), "must untrack through the helper")
		assert.Empty(t, conn.calls, "tracking must not touch the connection")
	})
}

func TestTrackingListenerAccept(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn := &stubConn{}
		helper := &recordingHelper{}
		l := newTrackingListener(&stubListener{conn: conn}, helper)

		got, err := l.Accept()
		require.NoError(t, err)
		tc, ok := got.(*TrackedConn)
		require.True(t, ok, "must return a *TrackedConn, got %T", got)
		assert.Same(t, conn, tc.conn, "must wrap the accepted connection")
		assert.Empty(t, helper.trackedConns(), "must not track on Accept")

		assert.NotNil(t, tc.trackFn(tc), "trackFn must track through the helper")
		assert.Equal(t, []net.Conn{tc}, helper.trackedConns(), "trackFn must track through the helper")
	})

	t.Run("nil helper", func(t *testing.T) {
		l := newTrackingListener(&stubListener{conn: &stubConn{}}, nil)

		got, err := l.Accept()
		require.NoError(t, err)
		assert.True(t, isNoopTrackerFunc(got.(*TrackedConn).trackFn), "must not track without a helper")
	})

	t.Run("error", func(t *testing.T) {
		wantErr := errors.New("great sadness")
		l := newTrackingListener(&stubListener{conn: &stubConn{}, err: wantErr}, &recordingHelper{})

		got, err := l.Accept()
		assert.Equal(t, wantErr, err, "must return the listener's error")
		assert.Nil(t, got, "must not return a connection on error")
	})
}

func TestTrackingListenerCloseAndAddr(t *testing.T) {
	for _, wantErr := range []error{nil, errors.New("great sadness")} {
		inner := &stubListener{err: wantErr}
		l := newTrackingListener(inner, &recordingHelper{})

		assert.Equal(t, stubAddr("listener"), l.Addr(), "must return the listener's address")
		assert.Equal(t, wantErr, l.Close(), "must return the listener's error")
		assert.True(t, inner.closed, "must close the listener")
	}
}

func TestTrackingListenerOverTCP(t *testing.T) {
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	helper := &recordingHelper{}
	l := WrapListener(inner, helper)
	defer l.Close()
	require.IsType(t, &TrackingListener{}, l)
	assert.Equal(t, inner.Addr(), l.Addr())

	client, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	defer client.Close()

	server, err := l.Accept()
	require.NoError(t, err)
	require.IsType(t, &TrackedConn{}, server)
	assert.Equal(t, client.LocalAddr().String(), server.RemoteAddr().String())
	assert.Equal(t, client.RemoteAddr().String(), server.LocalAddr().String())

	_, err = client.Write([]byte("hello"))
	require.NoError(t, err)
	buf := make([]byte, 5)
	_, err = io.ReadFull(server, buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf))

	_, err = server.Write([]byte("world"))
	require.NoError(t, err)
	_, err = io.ReadFull(client, buf)
	require.NoError(t, err)
	assert.Equal(t, "world", string(buf))

	require.NoError(t, server.Close())
	_, err = client.Read(buf)
	assert.ErrorIs(t, err, io.EOF, "the client must see the connection close")

	require.NoError(t, l.Close())
	_, err = l.Accept()
	assert.ErrorIs(t, err, net.ErrClosed, "Accept must fail once the listener closed")

	assert.Empty(t, helper.trackedConns(), "must not track a connection that is not HTTP/2")
	assert.True(t, helper.IsShutdownComplete(context.Background()))
}

func TestTrackingListenerOverTCPWithHTTP2(t *testing.T) {
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	helper := &recordingHelper{}
	l := WrapListener(inner, helper)
	defer l.Close()

	client, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	defer client.Close()

	server, err := l.Accept()
	require.NoError(t, err)

	_, err = client.Write([]byte(xhttp2.ClientPreface))
	require.NoError(t, err)
	_, err = io.ReadFull(server, make([]byte, len(xhttp2.ClientPreface)))
	require.NoError(t, err)
	assert.Equal(t, []net.Conn{server}, helper.trackedConns(), "must track an HTTP/2 connection")

	require.NoError(t, server.Close())
	assert.Equal(t, []net.Conn{server}, helper.untrackedConns(), "must untrack a closed connection")
}
