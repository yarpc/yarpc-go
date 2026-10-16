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
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhttp2 "golang.org/x/net/http2"
)

var errWrite = errors.New("great sadness")

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

// fakeConn is a [stubConn] that reads the given chunks, at most one per Read,
// and keeps what is written to it. Its failWrite-th Write fails, counting
// from 1.
type fakeConn struct {
	stubConn
	reads     [][]byte
	written   bytes.Buffer
	writes    int
	failWrite int
}

func (c *fakeConn) Read(b []byte) (int, error) {
	c.record("Read")
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	n := copy(b, c.reads[0])
	if c.reads[0] = c.reads[0][n:]; len(c.reads[0]) == 0 {
		c.reads = c.reads[1:]
	}
	return n, nil
}

func (c *fakeConn) Write(b []byte) (int, error) {
	c.record("Write")
	if c.writes++; c.writes == c.failWrite {
		return 0, errWrite
	}
	return c.written.Write(b)
}

// closeWriteConn is a [stubConn] that can be half-closed.
type closeWriteConn struct{ stubConn }

func (c *closeWriteConn) CloseWrite() error {
	c.record("CloseWrite")
	return c.err
}

// newTestConn returns an untracked [TrackedConn] over conn.
func newTestConn(conn net.Conn) *TrackedConn {
	return newTrackedConn(conn, nil).(*TrackedConn)
}

// newTrackedFakeConn returns a [TrackedConn] that the helper tracks, once it
// read the client preface followed by frames.
func newTrackedFakeConn(t *testing.T, helper *recordingHelper, frames ...[]byte) (*TrackedConn, *fakeConn) {
	conn := &fakeConn{reads: [][]byte{append([]byte(xhttp2.ClientPreface), bytes.Join(frames, nil)...)}}
	tc := newTrackedConn(conn, helper.TrackConnection).(*TrackedConn)
	_, err := tc.Read(make([]byte, 1024))
	require.NoError(t, err)
	require.True(t, tc.tracked.Load(), "must be tracked")
	conn.calls = nil
	return tc, conn
}

func headersFrame(streamID uint32) []byte {
	return rawFrame(xhttp2.FrameHeaders, xhttp2.FlagHeadersEndHeaders, streamID, []byte{0x82})
}

// firstGoAway is what the first phase of the graceful shutdown writes.
func firstGoAway() []byte {
	return append(goAwayFrame(maxStreamID), pingFrame()...)
}

func TestNewTrackedConn(t *testing.T) {
	conn := &stubConn{}
	helper := &recordingHelper{}
	got := newTrackedConn(conn, helper.TrackConnection)

	tc, ok := got.(*TrackedConn)
	require.True(t, ok, "must return a *TrackedConn, got %T", got)
	assert.Same(t, conn, tc.conn, "must wrap the given connection")
	require.NotNil(t, tc.trackFn, "must keep the given tracker func")
	assert.NotNil(t, tc.pingAcked)
	assert.False(t, isClosed(tc.done()), "must not be done before Close")
	assert.False(t, tc.tracked.Load(), "must not be tracked before reading the client preface")

	tc.trackFn(tc)
	assert.Equal(t, []net.Conn{tc}, helper.trackedConns(), "trackFn must be the given tracker func")
	assert.Empty(t, conn.calls, "construction must not touch the connection")
}

func TestNoopTrackerFunc(t *testing.T) {
	conn := &stubConn{}
	assert.Nil(t, noopTrackerFunc(nil), "must not track")
	assert.Nil(t, noopTrackerFunc(conn), "must not track")
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
		for _, wantErr := range []error{nil, errWrite} {
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

func TestTrackedConnCloseWrite(t *testing.T) {
	t.Run("unsupported", func(t *testing.T) {
		conn := &stubConn{err: errWrite}
		assert.NoError(t, newTestConn(conn).CloseWrite())
		assert.Empty(t, conn.calls)
	})

	for _, wantErr := range []error{nil, errWrite} {
		conn := &closeWriteConn{stubConn{err: wantErr}}
		assert.Equal(t, wantErr, newTestConn(conn).CloseWrite(), "must return the connection's error")
		assert.Equal(t, []string{"CloseWrite"}, conn.calls, "must forward exactly one call")
	}

	t.Run("TCP", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer l.Close()
		client, err := net.Dial("tcp", l.Addr().String())
		require.NoError(t, err)
		defer client.Close()
		server, err := l.Accept()
		require.NoError(t, err)
		tc := newTestConn(server)
		defer tc.Close()

		require.NoError(t, tc.CloseWrite())
		_, err = client.Read(make([]byte, 1))
		assert.ErrorIs(t, err, io.EOF, "the client must see the write side close")
		_, err = client.Write([]byte("x"))
		require.NoError(t, err)
		_, err = io.ReadFull(tc, make([]byte, 1))
		assert.NoError(t, err, "the read side must remain open")
	})
}

func TestTrackedConnOverPipe(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	helper := &recordingHelper{}
	tc := newTrackedConn(server, helper.TrackConnection)

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

	assert.Empty(t, helper.trackedConns(), "must not track a connection that is not HTTP/2")
}

func TestTrackedConnTracking(t *testing.T) {
	preface := []byte(xhttp2.ClientPreface)
	withFrames := func(frames ...[]byte) []byte {
		return append(append([]byte(nil), preface...), bytes.Join(frames, nil)...)
	}
	byteByByte := func(b []byte) [][]byte {
		chunks := make([][]byte, len(b))
		for i := range b {
			chunks[i] = b[i : i+1]
		}
		return chunks
	}

	tests := []struct {
		desc      string
		reads     [][]byte
		refuse    bool // the helper does not track
		noTrackFn bool

		wantAsked        bool // trackFn called
		wantTracked      bool
		wantLastStreamID uint32
	}{
		{
			desc:  "HTTP/1.1",
			reads: [][]byte{[]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")},
		},
		{
			desc:  "HTTP/1.1 sharing the method prefix",
			reads: [][]byte{[]byte("PRI * HTTP/1.1\r\n\r\n")},
		},
		{
			desc:  "client preface diverging past the first read",
			reads: [][]byte{preface[:10], []byte("HTTP/1.1\r\n\r\n")},
		},
		{
			desc:        "client preface",
			reads:       [][]byte{preface},
			wantAsked:   true,
			wantTracked: true,
		},
		{
			desc:        "client preface split",
			reads:       [][]byte{preface[:3], preface[3:20], preface[20:]},
			wantAsked:   true,
			wantTracked: true,
		},
		{
			desc:        "client preface byte by byte",
			reads:       byteByByte(preface),
			wantAsked:   true,
			wantTracked: true,
		},
		{
			desc:             "client preface and frames",
			reads:            [][]byte{withFrames(headersFrame(1), headersFrame(3), rawFrame(xhttp2.FrameData, 0, 3, []byte("hi")))},
			wantAsked:        true,
			wantTracked:      true,
			wantLastStreamID: 3,
		},
		{
			desc:             "frames in later reads",
			reads:            append([][]byte{preface}, byteByByte(bytes.Join([][]byte{headersFrame(1), headersFrame(5)}, nil))...),
			wantAsked:        true,
			wantTracked:      true,
			wantLastStreamID: 5,
		},
		{
			desc:      "refused by the helper",
			reads:     [][]byte{withFrames(headersFrame(1))},
			refuse:    true,
			wantAsked: true,
		},
		{
			desc:      "no tracker func",
			reads:     [][]byte{withFrames(headersFrame(1))},
			noTrackFn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			want := bytes.Join(tt.reads, nil)
			conn := &fakeConn{reads: tt.reads}
			helper := &recordingHelper{refuse: tt.refuse}
			var fn ConnectionTrackerFunc = helper.TrackConnection
			if tt.noTrackFn {
				fn = nil
			}
			tc := newTrackedConn(conn, fn).(*TrackedConn)

			var got []byte
			buf := make([]byte, 64)
			for {
				n, err := tc.Read(buf)
				got = append(got, buf[:n]...)
				if err != nil {
					require.ErrorIs(t, err, io.EOF)
					break
				}
			}
			assert.Equal(t, want, got, "must read everything unchanged")

			if tt.wantAsked {
				assert.Equal(t, []net.Conn{tc}, helper.trackedConns(), "must ask to be tracked once")
			} else {
				assert.Empty(t, helper.trackedConns(), "must not ask to be tracked")
			}
			assert.Equal(t, tt.wantTracked, tc.tracked.Load())
			assert.Equal(t, tt.wantLastStreamID, tc.lastStreamID)
		})
	}
}

func TestTrackedConnTrackAfterClose(t *testing.T) {
	conn := &fakeConn{reads: [][]byte{[]byte(xhttp2.ClientPreface)}}
	helper := &recordingHelper{}
	tc := newTrackedConn(conn, helper.TrackConnection).(*TrackedConn)

	require.NoError(t, tc.Close())
	_, err := tc.Read(make([]byte, 64))
	require.NoError(t, err)
	assert.Empty(t, helper.trackedConns(), "must not track a closed connection")
	assert.False(t, tc.tracked.Load())
}

func TestTrackedConnClose(t *testing.T) {
	t.Run("untracked", func(t *testing.T) {
		conn := &fakeConn{}
		tc := newTestConn(conn)
		tc.shutdown()

		require.NoError(t, tc.Close())
		require.NoError(t, tc.Close(), "closing again must be a no-op")
		assert.Equal(t, []string{"Close"}, conn.calls, "must close the connection once")
		assert.True(t, isClosed(tc.done()))
	})

	t.Run("error", func(t *testing.T) {
		conn := &stubConn{err: errWrite}
		tc := newTestConn(conn)
		assert.Equal(t, errWrite, tc.Close(), "must return the connection's error")
		assert.True(t, isClosed(tc.done()))
	})

	t.Run("tracked", func(t *testing.T) {
		helper := &recordingHelper{}
		tc, conn := newTrackedFakeConn(t, helper, headersFrame(7))

		require.NoError(t, tc.Close())
		require.NoError(t, tc.Close())
		assert.Equal(t, []string{"Close"}, conn.calls, "must not write when not shutting down")
		assert.Equal(t, []net.Conn{tc}, helper.untrackedConns(), "must untrack once")
		assert.True(t, isClosed(tc.done()))
	})

	t.Run("shutting down", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{}, headersFrame(7))
		tc.shuttingDown.Store(true) // as shutdown does, before draining

		require.NoError(t, tc.Close())
		assert.Equal(t, append(firstGoAway(), goAwayFrame(7)...), conn.written.Bytes(),
			"must write both GOAWAY frames before closing")
		assert.Equal(t, []string{"SetWriteDeadline", "Write", "Close"}, conn.calls)
		assert.WithinDuration(t, time.Now().Add(closeWriteTimeout), conn.deadline, closeWriteTimeout)
	})

	t.Run("shutting down after the first GOAWAY", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{}, headersFrame(7))
		tc.shuttingDown.Store(true)
		require.NoError(t, tc.advance(goAwayFirst))

		require.NoError(t, tc.Close())
		assert.Equal(t, append(firstGoAway(), goAwayFrame(7)...), conn.written.Bytes(),
			"must only write the second GOAWAY before closing")
	})

	t.Run("shutting down during a write", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{})
		tc.shuttingDown.Store(true)

		tc.wmu.Lock()
		require.NoError(t, tc.Close(), "must not wait on the write in progress")
		tc.wmu.Unlock()
		assert.Equal(t, []string{"Close"}, conn.calls)
	})
}

func TestTrackedConnWrite(t *testing.T) {
	data := rawFrame(xhttp2.FrameData, 0, 1, []byte("hello"))
	serverGoAway := goAwayFrame(3)
	cat := func(frames ...[]byte) []byte { return bytes.Join(frames, nil) }

	t.Run("untracked", func(t *testing.T) {
		conn := &fakeConn{}
		tc := newTestConn(conn)

		n, err := tc.Write(cat(data, serverGoAway))
		require.NoError(t, err)
		assert.Equal(t, len(data)+len(serverGoAway), n)
		assert.Equal(t, cat(data, serverGoAway), conn.written.Bytes(), "must write unchanged")
		assert.Equal(t, goAwayNone, tc.goAway, "must not follow frames")
	})

	t.Run("frames", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{})
		b := cat(rawFrame(xhttp2.FrameSettings, 0, 0, nil), headersFrame(1), data)

		n, err := tc.Write(b)
		require.NoError(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, b, conn.written.Bytes(), "must write unchanged")
		assert.Equal(t, []string{"Write"}, conn.calls, "must write at once")
	})

	tests := []struct {
		desc   string
		before func(*TrackedConn)
		writes [][]byte
		want   []byte
		state  goAwayState // zero for goAwaySecond: the server GOAWAY takes the place of the second
	}{
		{
			desc:   "server GOAWAY",
			writes: [][]byte{cat(data, serverGoAway, data)},
			want:   cat(data, firstGoAway(), serverGoAway, data),
		},
		{
			desc:   "server GOAWAY first",
			writes: [][]byte{serverGoAway},
			want:   cat(firstGoAway(), serverGoAway),
		},
		{
			desc:   "server GOAWAY twice",
			writes: [][]byte{cat(serverGoAway, goAwayFrame(1))},
			want:   cat(firstGoAway(), serverGoAway, goAwayFrame(1)),
		},
		{
			desc:   "server GOAWAY in a later write",
			writes: [][]byte{data, serverGoAway, goAwayFrame(1)},
			want:   cat(data, firstGoAway(), serverGoAway, goAwayFrame(1)),
		},
		{
			desc:   "server GOAWAY after the first GOAWAY",
			before: func(tc *TrackedConn) { require.NoError(t, tc.advance(goAwayFirst)) },
			writes: [][]byte{serverGoAway},
			want:   cat(firstGoAway(), serverGoAway),
		},
		{
			desc:   "server GOAWAY header split",
			writes: [][]byte{serverGoAway[:4], serverGoAway[4:]},
			want:   serverGoAway,
		},
		{
			desc: "server GOAWAY header split, first GOAWAY pending",
			before: func(tc *TrackedConn) {
				_, err := tc.Write(data[:5])
				require.NoError(t, err)
				require.NoError(t, tc.advance(goAwayFirst))
			},
			writes: [][]byte{cat(data[5:], serverGoAway[:4]), serverGoAway[4:]},
			want:   cat(data, serverGoAway), // a first GOAWAY after it would raise the last stream ID
		},
		{
			desc: "first GOAWAY pending",
			before: func(tc *TrackedConn) {
				_, err := tc.Write(data[:5])
				require.NoError(t, err)
				require.NoError(t, tc.advance(goAwayFirst))
			},
			writes: [][]byte{data[5 : frameHeaderLen+2], data[frameHeaderLen+2:]},
			want:   cat(data, firstGoAway()),
			state:  goAwayFirst,
		},
		{
			desc: "first GOAWAY pending, then server GOAWAY",
			before: func(tc *TrackedConn) {
				_, err := tc.Write(data[:5])
				require.NoError(t, err)
				require.NoError(t, tc.advance(goAwayFirst))
			},
			writes: [][]byte{cat(data[5:], serverGoAway)},
			want:   cat(data, firstGoAway(), serverGoAway),
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tc, conn := newTrackedFakeConn(t, &recordingHelper{})
			if tt.before != nil {
				tt.before(tc)
			}
			for _, b := range tt.writes {
				n, err := tc.Write(b)
				require.NoError(t, err)
				assert.Equal(t, len(b), n, "must report the bytes of b written")
			}
			assert.Equal(t, tt.want, conn.written.Bytes())
			if tt.state == goAwayNone {
				tt.state = goAwaySecond
			}
			assert.Equal(t, tt.state, tc.goAway)
			assert.Empty(t, tc.pending)
		})
	}

	t.Run("errors", func(t *testing.T) {
		b := cat(data, serverGoAway)
		for failWrite, wantN := range map[int]int{1: 0, 2: len(data), 3: len(data)} {
			tc, conn := newTrackedFakeConn(t, &recordingHelper{})
			conn.failWrite = failWrite

			n, err := tc.Write(b)
			assert.Equal(t, errWrite, err, "write %d must fail", failWrite)
			assert.Equal(t, wantN, n, "write %d must count the bytes of b written", failWrite)
			assert.Equal(t, failWrite, conn.writes, "must stop writing on error")
		}
	})
}

// h2Pipe connects a [TrackedConn], as the server side, to an HTTP/2 client.
type h2Pipe struct {
	tc     *TrackedConn
	helper *recordingHelper
	client net.Conn
	fr     *xhttp2.Framer
}

func newH2Pipe(t *testing.T, frames ...[]byte) *h2Pipe {
	server, client := net.Pipe()
	p := &h2Pipe{helper: &recordingHelper{}, client: client, fr: xhttp2.NewFramer(nil, client)}
	p.tc = newTrackedConn(server, p.helper.TrackConnection).(*TrackedConn)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_, _ = io.Copy(io.Discard, p.tc) // reads as the HTTP/2 server would
	}()
	t.Cleanup(func() {
		_ = client.Close()
		_ = p.tc.Close()
		<-readDone
	})

	p.send(t, append([][]byte{[]byte(xhttp2.ClientPreface)}, frames...)...)
	require.Eventually(t, p.tc.tracked.Load, time.Second, time.Millisecond, "must track")
	return p
}

func (p *h2Pipe) send(t *testing.T, frames ...[]byte) {
	for _, f := range frames {
		_, err := p.client.Write(f)
		require.NoError(t, err)
	}
}

func (p *h2Pipe) readGoAway(t *testing.T) uint32 {
	f, err := p.fr.ReadFrame()
	require.NoError(t, err)
	ga, ok := f.(*xhttp2.GoAwayFrame)
	require.True(t, ok, "want a GOAWAY frame, got %v", f)
	assert.Equal(t, xhttp2.ErrCodeNo, ga.ErrCode)
	return ga.LastStreamID
}

func (p *h2Pipe) readPing(t *testing.T) {
	f, err := p.fr.ReadFrame()
	require.NoError(t, err)
	ping, ok := f.(*xhttp2.PingFrame)
	require.True(t, ok, "want a PING frame, got %v", f)
	assert.False(t, ping.IsAck())
	assert.Equal(t, pingData, ping.Data)
}

func (p *h2Pipe) readEOF(t *testing.T) {
	_, err := p.fr.ReadFrame()
	assert.ErrorIs(t, err, io.EOF, "the client must see the connection close")
}

func TestTrackedConnShutdown(t *testing.T) {
	pingAck := rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, pingData[:])

	t.Run("untracked", func(t *testing.T) {
		conn := &fakeConn{}
		tc := newTestConn(conn)
		tc.shutdown()
		assert.False(t, tc.shuttingDown.Load(), "must not shut down an untracked connection")
		assert.Empty(t, conn.calls)
	})

	t.Run("client acknowledges", func(t *testing.T) {
		p := newH2Pipe(t, headersFrame(1), headersFrame(3))

		p.tc.shutdown()
		p.tc.shutdown() // no-op
		assert.Equal(t, uint32(maxStreamID), p.readGoAway(t))
		p.readPing(t)

		// Opened before the client read the first GOAWAY: still processed.
		p.send(t, headersFrame(5), pingAck)
		assert.Equal(t, uint32(5), p.readGoAway(t), "must count streams opened until the PING ACK")

		require.NoError(t, p.tc.Close())
		p.readEOF(t)
		assert.True(t, isClosed(p.tc.done()))
		assert.Equal(t, []net.Conn{p.tc}, p.helper.untrackedConns())
	})

	t.Run("PING ACK timeout", func(t *testing.T) {
		p := newH2Pipe(t, headersFrame(1))

		start := time.Now()
		p.tc.shutdown()
		assert.Equal(t, uint32(maxStreamID), p.readGoAway(t))
		p.readPing(t)
		assert.Equal(t, uint32(1), p.readGoAway(t))
		assert.GreaterOrEqual(t, time.Since(start), pingAckTimeout, "must wait for the PING ACK")
	})

	t.Run("closed before the PING ACK", func(t *testing.T) {
		p := newH2Pipe(t, headersFrame(3))

		p.tc.shutdown()
		assert.Equal(t, uint32(maxStreamID), p.readGoAway(t))
		p.readPing(t)

		// Close does not wait on a write in progress: let drain finish writing.
		require.Eventually(t, func() bool {
			if !p.tc.wmu.TryLock() {
				return false
			}
			p.tc.wmu.Unlock()
			return true
		}, time.Second, time.Millisecond)

		closed := make(chan error, 1)
		go func() { closed <- p.tc.Close() }()
		assert.Equal(t, uint32(3), p.readGoAway(t), "Close must write the second GOAWAY")
		p.readEOF(t)
		require.NoError(t, <-closed)
	})
}
