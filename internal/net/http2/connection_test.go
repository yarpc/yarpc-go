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
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAddr is a [net.Addr] for tests.
type stubAddr string

func (a stubAddr) Network() string { return "stub" }
func (a stubAddr) String() string  { return string(a) }

// stubConn is a [net.Conn] that records the calls it receives, and fails
// each of them with err.
type stubConn struct {
	err      error
	calls    []string
	deadline time.Time
}

var _ net.Conn = (*stubConn)(nil)

func (c *stubConn) record(call string) { c.calls = append(c.calls, call) }

func (c *stubConn) Read(b []byte) (int, error) {
	c.record("Read")
	return copy(b, "read"), c.err
}

func (c *stubConn) Write(b []byte) (int, error) {
	c.record("Write")
	return len(b), c.err
}

func (c *stubConn) Close() error {
	c.record("Close")
	return c.err
}

func (c *stubConn) LocalAddr() net.Addr  { return stubAddr("local") }
func (c *stubConn) RemoteAddr() net.Addr { return stubAddr("remote") }

func (c *stubConn) SetDeadline(t time.Time) error {
	c.record("SetDeadline")
	c.deadline = t
	return c.err
}

func (c *stubConn) SetReadDeadline(t time.Time) error {
	c.record("SetReadDeadline")
	c.deadline = t
	return c.err
}

func (c *stubConn) SetWriteDeadline(t time.Time) error {
	c.record("SetWriteDeadline")
	c.deadline = t
	return c.err
}

func TestNewTrackedConn(t *testing.T) {
	conn := &stubConn{}
	var tracked []net.Conn
	got := newTrackedConn(conn, func(c net.Conn) { tracked = append(tracked, c) })

	tc, ok := got.(*TrackedConn)
	require.True(t, ok, "must return a *TrackedConn, got %T", got)
	assert.Same(t, conn, tc.conn, "must wrap the given connection")
	require.NotNil(t, tc.trackFn, "must keep the given tracker func")

	tc.trackFn(tc)
	assert.Equal(t, []net.Conn{tc}, tracked, "trackFn must be the given tracker func")
	assert.Empty(t, conn.calls, "construction must not touch the connection")
}

func TestNoopTrackerFunc(t *testing.T) {
	conn := &stubConn{}
	assert.NotPanics(t, func() {
		noopTrackerFunc(nil)
		noopTrackerFunc(conn)
	})
	assert.Empty(t, conn.calls, "noopTrackerFunc must not touch the connection")
}

func TestTrackedConnForwards(t *testing.T) {
	deadline := time.Unix(1700000000, 0)

	tests := []struct {
		desc string
		call func(net.Conn) (int, error) // n is ignored when not meaningful
		n    int
	}{
		{
			desc: "Read",
			call: func(c net.Conn) (int, error) { return c.Read(make([]byte, 8)) },
			n:    len("read"),
		},
		{
			desc: "Write",
			call: func(c net.Conn) (int, error) { return c.Write([]byte("hello")) },
			n:    len("hello"),
		},
		{
			desc: "Close",
			call: func(c net.Conn) (int, error) { return 0, c.Close() },
		},
		{
			desc: "SetDeadline",
			call: func(c net.Conn) (int, error) { return 0, c.SetDeadline(deadline) },
		},
		{
			desc: "SetReadDeadline",
			call: func(c net.Conn) (int, error) { return 0, c.SetReadDeadline(deadline) },
		},
		{
			desc: "SetWriteDeadline",
			call: func(c net.Conn) (int, error) { return 0, c.SetWriteDeadline(deadline) },
		},
	}

	for _, tt := range tests {
		for _, wantErr := range []error{nil, errors.New("great sadness")} {
			name := tt.desc + "/success"
			if wantErr != nil {
				name = tt.desc + "/error"
			}
			t.Run(name, func(t *testing.T) {
				conn := &stubConn{err: wantErr}
				tc := newTrackedConn(conn, noopTrackerFunc)

				n, err := tt.call(tc)
				assert.Equal(t, wantErr, err, "must return the connection's error")
				assert.Equal(t, tt.n, n, "must return the connection's byte count")
				assert.Equal(t, []string{tt.desc}, conn.calls, "must forward exactly one call")
				if tt.desc[:3] == "Set" {
					assert.Equal(t, deadline, conn.deadline, "must forward the deadline")
				}
			})
		}
	}
}

func TestTrackedConnAddrs(t *testing.T) {
	conn := &stubConn{}
	tc := newTrackedConn(conn, noopTrackerFunc)

	assert.Equal(t, stubAddr("local"), tc.LocalAddr())
	assert.Equal(t, stubAddr("remote"), tc.RemoteAddr())
}

func TestTrackedConnOverPipe(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	var tracked []net.Conn
	tc := newTrackedConn(server, func(c net.Conn) { tracked = append(tracked, c) })

	t.Run("write", func(t *testing.T) {
		go func() { _, _ = tc.Write([]byte("PING")) }()
		buf := make([]byte, 4)
		_, err := io.ReadFull(client, buf)
		require.NoError(t, err)
		assert.Equal(t, "PING", string(buf))
	})

	t.Run("read", func(t *testing.T) {
		go func() { _, _ = client.Write([]byte("PONG")) }()
		buf := make([]byte, 4)
		_, err := io.ReadFull(tc, buf)
		require.NoError(t, err)
		assert.Equal(t, "PONG", string(buf))
	})

	t.Run("read deadline", func(t *testing.T) {
		require.NoError(t, tc.SetReadDeadline(time.Now().Add(-time.Second)))
		_, err := tc.Read(make([]byte, 1))
		assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
		require.NoError(t, tc.SetDeadline(time.Time{}))
	})

	t.Run("close", func(t *testing.T) {
		require.NoError(t, tc.Close())
		_, err := tc.Write([]byte("x"))
		assert.ErrorIs(t, err, io.ErrClosedPipe, "writes must fail once closed")
		_, err = client.Read(make([]byte, 1))
		assert.ErrorIs(t, err, io.EOF, "the peer must see the connection close")
	})

	assert.Empty(t, tracked, "reading and writing must not track the connection yet")
}
