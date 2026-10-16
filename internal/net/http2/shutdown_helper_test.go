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
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhttp2 "golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func TestNewShutdownHelper(t *testing.T) {
	h := NewShutdownHelper()
	require.IsType(t, &gracefulShutdownHelper{}, h)
	assert.Empty(t, h.(*gracefulShutdownHelper).trackedConns())
	assert.Equal(t, shutdownNone, h.(*gracefulShutdownHelper).shutdownState())
	assert.NotSame(t, h, NewShutdownHelper(), "must not share state between servers")
}

func TestNewNoopShutdownHelper(t *testing.T) {
	assert.IsType(t, &noopHelper{}, NewNoopShutdownHelper())
}

func TestNoopHelper(t *testing.T) {
	h := NewNoopShutdownHelper()

	t.Run("complete before shutdown", func(t *testing.T) {
		assert.True(t, h.IsShutdownReady(context.Background()))
		assert.True(t, h.IsShutdownComplete(context.Background()))
	})

	t.Run("shutdown", func(t *testing.T) {
		conn := &stubConn{}
		assert.NotPanics(t, func() {
			assert.Nil(t, h.TrackConnection(conn), "must not track")
			assert.Nil(t, h.TrackConnection(nil), "must not track")
			h.BeginShutdown()
			h.ShutdownCallback()
			h.BeginShutdown()
		})
		assert.Empty(t, conn.calls, "must not touch tracked connections")
		assert.True(t, h.IsShutdownReady(context.Background()))
		assert.True(t, h.IsShutdownComplete(context.Background()))
	})

	t.Run("complete with ended context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.True(t, h.IsShutdownReady(ctx), "must not block nor fail")
		assert.True(t, h.IsShutdownComplete(ctx), "must not block nor fail")
	})
}

func (h *gracefulShutdownHelper) trackedConns() []*TrackedConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := make([]*TrackedConn, 0, len(h.conns))
	for tc := range h.conns {
		conns = append(conns, tc)
	}
	return conns
}

func (h *gracefulShutdownHelper) shutdownState() shutdownState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

// newHelperConn returns a [TrackedConn] that h tracks: as the server never
// writes its preface, shutting it down writes nothing.
func newHelperConn(t *testing.T, h *gracefulShutdownHelper) (*TrackedConn, *fakeConn) {
	tc, conn := newFakeConnTrackedBy(t, h.TrackConnection)
	t.Cleanup(func() { _ = tc.Close() })
	return tc, conn
}

func TestGracefulShutdownHelperTrackConnection(t *testing.T) {
	t.Run("not a TrackedConn", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		assert.Nil(t, h.TrackConnection(nil))
		assert.Nil(t, h.TrackConnection(&stubConn{}))
		assert.Empty(t, h.trackedConns())
	})

	t.Run("track and untrack", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		tc := newTestConn(&stubConn{})

		untrack := h.TrackConnection(tc)
		require.NotNil(t, untrack)
		assert.Equal(t, []*TrackedConn{tc}, h.trackedConns())
		assert.False(t, tc.shuttingDown.Load(), "must not shut down before BeginShutdown")

		untrack()
		untrack()
		assert.Empty(t, h.trackedConns())
	})

	t.Run("closed connections", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		tc, _ := newHelperConn(t, h)
		require.Equal(t, []*TrackedConn{tc}, h.trackedConns())

		require.NoError(t, tc.Close())
		assert.Empty(t, h.trackedConns(), "must untrack closed connections")
	})

	t.Run("during shutdown", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		h.BeginShutdown()

		tc, _ := newHelperConn(t, h)
		assert.True(t, tc.shuttingDown.Load(), "must shut down a connection tracked during shutdown")
		assert.Equal(t, []*TrackedConn{tc}, h.trackedConns())
	})

	t.Run("once waiting", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		h.BeginShutdown()
		require.True(t, h.IsShutdownComplete(context.Background()))

		tc := newTrackedConn(&fakeConn{reads: [][]byte{append([]byte(xhttp2.ClientPreface), headersFrame(1)...)}},
			h.TrackConnection).(*TrackedConn)
		n, err := tc.Read(make([]byte, 64))
		assert.ErrorIs(t, err, net.ErrClosed, "the server must not serve a refused connection")
		assert.Zero(t, n)
		assert.True(t, isClosed(tc.done()), "must close a refused connection")
		assert.Empty(t, h.trackedConns(), "must not track connections once waiting")
	})
}

func TestGracefulShutdownHelperBeginShutdown(t *testing.T) {
	for _, begin := range []string{"BeginShutdown", "ShutdownCallback"} {
		t.Run(begin, func(t *testing.T) {
			h := newGracefulShutdownHelper()
			a, _ := newHelperConn(t, h)
			b, _ := newHelperConn(t, h)
			closed := newTestConn(&stubConn{}) // closed before BeginShutdown, but not untracked yet
			h.TrackConnection(closed)
			require.NoError(t, closed.Close())

			if begin == "BeginShutdown" {
				h.BeginShutdown()
			} else {
				h.ShutdownCallback()
			}
			assert.Equal(t, shutdownStarted, h.shutdownState())
			assert.True(t, a.shuttingDown.Load(), "must shut down every connection")
			assert.True(t, b.shuttingDown.Load(), "must shut down every connection")
			assert.False(t, closed.shuttingDown.Load(), "must not shut down an untracked connection")

			// Later calls, such as net/http's to ShutdownCallback, are no-ops.
			h.BeginShutdown()
			h.ShutdownCallback()
			assert.Equal(t, shutdownStarted, h.shutdownState())
		})
	}
}

func TestGracefulShutdownHelperIsShutdownComplete(t *testing.T) {
	t.Run("not shutting down", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		newHelperConn(t, h)

		assert.True(t, h.IsShutdownComplete(context.Background()), "must not wait before BeginShutdown")
		assert.Equal(t, shutdownNone, h.shutdownState(), "must not refuse connections before BeginShutdown")
	})

	t.Run("no connections", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		h.BeginShutdown()

		assert.True(t, h.IsShutdownComplete(context.Background()))
		assert.Equal(t, shutdownComplete, h.shutdownState())
		assert.True(t, h.IsShutdownComplete(context.Background()), "must remain complete")
	})

	t.Run("connections close", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		a, _ := newHelperConn(t, h)
		h.BeginShutdown()
		b, _ := newHelperConn(t, h) // sent its preface after BeginShutdown

		result := make(chan bool, 1)
		go func() { result <- h.IsShutdownComplete(context.Background()) }()

		require.NoError(t, a.Close())
		assert.Never(t, func() bool { return len(result) > 0 }, 50*time.Millisecond, time.Millisecond,
			"must wait for every connection")

		require.NoError(t, b.Close())
		assert.True(t, <-result)
		assert.Equal(t, shutdownComplete, h.shutdownState())
	})

	t.Run("timeout", func(t *testing.T) {
		h := newGracefulShutdownHelper()
		a, conn := newHelperConn(t, h)
		b, _ := newHelperConn(t, h)
		h.BeginShutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		assert.False(t, h.IsShutdownComplete(ctx), "must report the connections left open")
		assert.True(t, isClosed(a.done()), "must close the connections left open")
		assert.True(t, isClosed(b.done()), "must close the connections left open")
		assert.Contains(t, conn.calls, "Close")
		assert.Empty(t, h.trackedConns())

		assert.Equal(t, shutdownComplete, h.shutdownState())
		assert.True(t, h.IsShutdownComplete(context.Background()), "every connection is closed now")
	})
}

// startHTTP2Server serves HTTP/2 with prior knowledge through a listener
// wrapped for h. Its handler waits until release is closed.
func startHTTP2Server(t *testing.T, h ShutdownHelper) (addr string, release chan struct{}) {
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	l := WrapListener(inner, h)
	t.Cleanup(func() { _ = l.Close() })

	release = make(chan struct{})
	opts := &xhttp2.ServeConnOpts{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		_, _ = io.WriteString(w, "ok")
	})}
	srv := &xhttp2.Server{}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go srv.ServeConn(conn, opts)
		}
	}()
	return l.Addr().String(), release
}

// h2Client is a raw HTTP/2 client with prior knowledge.
type h2Client struct {
	t    *testing.T
	conn net.Conn
	fr   *xhttp2.Framer
}

func dialHTTP2(t *testing.T, addr string) *h2Client {
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

	c := &h2Client{t: t, conn: conn, fr: xhttp2.NewFramer(conn, conn)}
	_, err = io.WriteString(conn, xhttp2.ClientPreface)
	require.NoError(t, err)
	require.NoError(t, c.fr.WriteSettings())

	// The server preface: a SETTINGS frame that comes first.
	f, err := c.fr.ReadFrame()
	require.NoError(t, err)
	settings, ok := f.(*xhttp2.SettingsFrame)
	require.True(t, ok, "the server must write SETTINGS first, got %v", f)
	require.False(t, settings.IsAck())
	require.NoError(t, c.fr.WriteSettingsAck())
	return c
}

func (c *h2Client) get(streamID uint32) {
	var block []byte
	enc := hpack.NewEncoder(&bytesWriter{&block})
	for _, f := range []hpack.HeaderField{
		{Name: ":method", Value: "GET"},
		{Name: ":scheme", Value: "http"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/"},
	} {
		require.NoError(c.t, enc.WriteField(f))
	}
	require.NoError(c.t, c.fr.WriteHeaders(xhttp2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: block,
		EndStream:     true,
		EndHeaders:    true,
	}))
}

// next returns the next frame that is not about flow control nor settings.
func (c *h2Client) next() xhttp2.Frame {
	for {
		f, err := c.fr.ReadFrame()
		require.NoError(c.t, err)
		switch f.(type) {
		case *xhttp2.SettingsFrame, *xhttp2.WindowUpdateFrame:
			continue
		}
		return f
	}
}

func (c *h2Client) nextGoAway() uint32 {
	f := c.next()
	ga, ok := f.(*xhttp2.GoAwayFrame)
	require.True(c.t, ok, "want a GOAWAY frame, got %v", f)
	assert.Equal(c.t, xhttp2.ErrCodeNo, ga.ErrCode)
	return ga.LastStreamID
}

// shutdownSequence checks the two-phase graceful shutdown, opening a stream
// streamID between both GOAWAY frames when not zero.
func (c *h2Client) shutdownSequence(streamID, wantLastStreamID uint32) {
	assert.Equal(c.t, uint32(maxStreamID), c.nextGoAway(), "first GOAWAY")
	f := c.next()
	ping, ok := f.(*xhttp2.PingFrame)
	require.True(c.t, ok, "want a PING frame right after the first GOAWAY, got %v", f)
	require.False(c.t, ping.IsAck())

	if streamID != 0 {
		c.get(streamID) // in flight before the client read the first GOAWAY
	}
	require.NoError(c.t, c.fr.WritePing(true, ping.Data))
	assert.Equal(c.t, wantLastStreamID, c.nextGoAway(), "second GOAWAY")
}

// responses reads the responses to streams, which may interleave, until
// each ends.
func (c *h2Client) responses(streamIDs ...uint32) {
	open := make(map[uint32]bool)
	for _, id := range streamIDs {
		open[id] = true
	}
	for len(open) > 0 {
		f := c.next()
		id := f.Header().StreamID
		require.True(c.t, open[id], "unexpected frame %v", f)
		if _, ok := f.(*xhttp2.DataFrame); ok && f.Header().Flags.Has(xhttp2.FlagDataEndStream) {
			delete(open, id)
		}
	}
}

type bytesWriter struct{ b *[]byte }

func (w *bytesWriter) Write(p []byte) (int, error) {
	*w.b = append(*w.b, p...)
	return len(p), nil
}

func TestGracefulShutdownHelperWithHTTP2Server(t *testing.T) {
	t.Run("streams in flight", func(t *testing.T) {
		h := NewShutdownHelper()
		addr, release := startHTTP2Server(t, h)
		c := dialHTTP2(t, addr)
		c.get(1)
		require.Eventually(t, func() bool {
			return len(h.(*gracefulShutdownHelper).trackedConns()) == 1
		}, time.Second, time.Millisecond)

		h.BeginShutdown()
		c.shutdownSequence(3, 3)

		close(release)
		c.responses(1, 3)
		require.NoError(t, c.conn.Close())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.True(t, h.IsShutdownComplete(ctx), "must complete once the client closed")
	})

	t.Run("connection opened during shutdown", func(t *testing.T) {
		h := NewShutdownHelper()
		addr, _ := startHTTP2Server(t, h)
		h.BeginShutdown()

		c := dialHTTP2(t, addr) // checks the server SETTINGS come first
		c.shutdownSequence(0, 0)
		require.NoError(t, c.conn.Close())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.True(t, h.IsShutdownComplete(ctx), "must complete once the client closed")
	})

	t.Run("connection opened once waiting", func(t *testing.T) {
		h := NewShutdownHelper()
		addr, _ := startHTTP2Server(t, h)
		h.BeginShutdown()
		require.True(t, h.IsShutdownComplete(context.Background()))

		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		defer conn.Close()
		require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
		c := &h2Client{t: t, conn: conn, fr: xhttp2.NewFramer(conn, conn)}
		_, _ = io.WriteString(conn, xhttp2.ClientPreface)
		_ = c.fr.WriteSettings()
		_ = c.fr.WriteHeaders(xhttp2.HeadersFrameParam{StreamID: 1, BlockFragment: []byte{0x82}, EndHeaders: true, EndStream: true})

		for {
			f, err := c.fr.ReadFrame()
			if err != nil {
				assert.False(t, errors.Is(err, os.ErrDeadlineExceeded), "must close a refused connection")
				break
			}
			assert.IsType(t, &xhttp2.SettingsFrame{}, f, "must not serve a refused connection")
		}
	})

	t.Run("client never closes", func(t *testing.T) {
		h := NewShutdownHelper()
		addr, _ := startHTTP2Server(t, h)
		c := dialHTTP2(t, addr)
		require.Eventually(t, func() bool {
			return len(h.(*gracefulShutdownHelper).trackedConns()) == 1
		}, time.Second, time.Millisecond)

		h.BeginShutdown()
		c.shutdownSequence(0, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		assert.False(t, h.IsShutdownComplete(ctx), "must report the connection left open")
		_, err := c.fr.ReadFrame()
		assert.ErrorIs(t, err, io.EOF, "must close the connection left open")
	})
}
