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
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhttp2 "golang.org/x/net/http2"
)

// rawFrame encodes an HTTP/2 frame byte by byte, so that tests can build
// frames the x/net/http2 framer refuses to write.
func rawFrame(typ xhttp2.FrameType, flags xhttp2.Flags, streamID uint32, payload []byte) []byte {
	f := make([]byte, frameHeaderLen, frameHeaderLen+len(payload))
	f[0], f[1], f[2] = byte(len(payload)>>16), byte(len(payload)>>8), byte(len(payload))
	f[3], f[4] = byte(typ), byte(flags)
	binary.BigEndian.PutUint32(f[5:], streamID)
	return append(f, payload...)
}

// readFrame decodes the only frame in b with the x/net/http2 framer.
func readFrame(t *testing.T, b []byte) xhttp2.Frame {
	fr := xhttp2.NewFramer(nil, bytes.NewReader(b))
	f, err := fr.ReadFrame()
	require.NoError(t, err)
	_, err = fr.ReadFrame()
	require.ErrorIs(t, err, io.EOF, "must hold exactly one frame")
	return f
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestGoAwayFrame(t *testing.T) {
	for _, id := range []uint32{0, 1, 7, maxStreamID} {
		b := goAwayFrame(id)
		assert.Len(t, b, frameHeaderLen+8, "GOAWAY without debug data")

		f, ok := readFrame(t, b).(*xhttp2.GoAwayFrame)
		require.True(t, ok, "must be a GOAWAY frame")
		assert.Zero(t, f.StreamID, "GOAWAY applies to the connection")
		assert.Equal(t, id, f.LastStreamID)
		assert.Equal(t, xhttp2.ErrCodeNo, f.ErrCode)
		assert.Equal(t, uint32(8), f.Length, "must carry no debug data")
	}
}

func TestPingFrame(t *testing.T) {
	b := pingFrame()
	assert.Len(t, b, frameHeaderLen+len(pingData))

	f, ok := readFrame(t, b).(*xhttp2.PingFrame)
	require.True(t, ok, "must be a PING frame")
	assert.Zero(t, f.StreamID, "PING applies to the connection")
	assert.False(t, f.IsAck(), "must not be an ACK")
	assert.Equal(t, pingData, f.Data)
}

func TestFrameScanner(t *testing.T) {
	type scanned struct {
		start int    // offset of the frame in the stream
		frame []byte // what scan reports
	}

	var (
		stream []byte
		want   []scanned
	)
	add := func(frame []byte, reported int) {
		want = append(want, scanned{start: len(stream), frame: frame[:reported]})
		stream = append(stream, frame...)
	}
	add(rawFrame(xhttp2.FrameSettings, 0, 0, nil), frameHeaderLen)
	add(rawFrame(xhttp2.FrameHeaders, xhttp2.FlagHeadersEndHeaders, 1, []byte{0x82, 0x86}), frameHeaderLen)
	add(rawFrame(xhttp2.FrameData, 0, 1, make([]byte, 300)), frameHeaderLen)
	add(pingFrame(), frameHeaderLen+len(pingData))
	add(rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, pingData[:]), frameHeaderLen+len(pingData))
	add(rawFrame(xhttp2.FramePing, 0, 0, []byte{1, 2, 3}), frameHeaderLen) // malformed: payload not kept
	add(goAwayFrame(3), frameHeaderLen)

	boundaries := map[int]bool{len(stream): true}
	for _, w := range want {
		boundaries[w.start] = true
	}

	t.Run("zero value at boundary", func(t *testing.T) {
		var s frameScanner
		assert.True(t, s.atBoundary())
	})

	t.Run("whole stream", func(t *testing.T) {
		var (
			s   frameScanner
			got []scanned
		)
		s.scan(stream, func(start int, frame []byte) {
			got = append(got, scanned{start: start, frame: append([]byte(nil), frame...)})
		})
		assert.Equal(t, want, got)
		assert.True(t, s.atBoundary())
	})

	// Writes and reads may split frames anywhere: every chunk size must
	// report the same frames, with start relative to the chunk.
	for _, size := range []int{1, 2, 5, frameHeaderLen, frameHeaderLen + 1, 16, 17, 100, len(stream) - 1} {
		var (
			s   frameScanner
			got []scanned
		)
		for off := 0; off < len(stream); off += size {
			s.scan(stream[off:min(off+size, len(stream))], func(start int, frame []byte) {
				got = append(got, scanned{start: off + start, frame: append([]byte(nil), frame...)})
			})
			end := min(off+size, len(stream))
			if !assert.Equal(t, boundaries[end], s.atBoundary(), "chunk size %d: atBoundary after %d bytes", size, end) {
				break
			}
		}
		assert.Equal(t, want, got, "chunk size %d", size)
	}

	t.Run("empty input", func(t *testing.T) {
		var s frameScanner
		s.scan(nil, func(int, []byte) { t.Fatal("no frame to report") })
		assert.True(t, s.atBoundary())
	})
}

func TestTrackedConnFrameRead(t *testing.T) {
	t.Run("last stream ID", func(t *testing.T) {
		tc := newTestConn(&stubConn{})
		var s frameScanner

		steps := []struct {
			desc  string
			frame []byte
			want  uint32
		}{
			{"first stream", rawFrame(xhttp2.FrameHeaders, xhttp2.FlagHeadersEndHeaders, 1, []byte{0x82}), 1},
			{"higher stream", rawFrame(xhttp2.FrameHeaders, 0, 5, []byte{0x82}), 5},
			{"lower stream", rawFrame(xhttp2.FrameHeaders, 0, 3, []byte{0x82}), 5},
			{"server stream", rawFrame(xhttp2.FrameHeaders, 0, 8, []byte{0x82}), 5},
			{"not HEADERS", rawFrame(xhttp2.FrameData, 0, 9, []byte("hello")), 5},
			{"PRIORITY", rawFrame(xhttp2.FramePriority, 0, 11, make([]byte, 5)), 5},
			{"reserved bit", rawFrame(xhttp2.FrameHeaders, 0, 1<<31|7, []byte{0x82}), 7},
			{"max stream", rawFrame(xhttp2.FrameHeaders, 0, maxStreamID, nil), maxStreamID},
		}
		for _, step := range steps {
			s.scan(step.frame, tc.frameRead)
			assert.Equal(t, step.want, tc.lastStreamID, step.desc)
		}
	})

	t.Run("PING ACK", func(t *testing.T) {
		otherData := []byte("12345678")
		tests := []struct {
			desc  string
			frame []byte
			acked bool
		}{
			{"PING", rawFrame(xhttp2.FramePing, 0, 0, pingData[:]), false},
			{"ACK of another PING", rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, otherData), false},
			{"ACK header only", rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, pingData[:])[:frameHeaderLen], false},
			{"ACK", rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, pingData[:]), true},
		}
		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				tc := newTestConn(&stubConn{})
				tc.frameRead(0, tt.frame)
				assert.Equal(t, tt.acked, isClosed(tc.pingAcked))
				assert.Zero(t, tc.lastStreamID, "PING must not change the last stream ID")
			})
		}
	})

	t.Run("repeated PING ACK", func(t *testing.T) {
		tc := newTestConn(&stubConn{})
		ack := rawFrame(xhttp2.FramePing, xhttp2.FlagPingAck, 0, pingData[:])

		var s frameScanner
		assert.NotPanics(t, func() {
			s.scan(append(append([]byte(nil), ack...), ack...), tc.frameRead)
		})
		assert.True(t, isClosed(tc.pingAcked))
	})
}

// bufferConn is a [stubConn] that keeps what is written to it.
type bufferConn struct {
	stubConn
	buf bytes.Buffer
}

func (c *bufferConn) Write(b []byte) (int, error) {
	c.record("Write")
	if c.err != nil {
		return 0, c.err
	}
	return c.buf.Write(b)
}

func TestTrackedConnWriteLocked(t *testing.T) {
	noFrame := func(int, []byte) {}

	t.Run("at frame boundary", func(t *testing.T) {
		conn := &bufferConn{}
		tc := newTestConn(conn)
		tc.prefaceSent = true
		tc.wmu.Lock()
		defer tc.wmu.Unlock()

		require.NoError(t, tc.writeLocked(goAwayFrame(1)))
		assert.Equal(t, goAwayFrame(1), conn.buf.Bytes(), "must write right away")
		assert.Empty(t, tc.pending)
	})

	t.Run("within a frame", func(t *testing.T) {
		conn := &bufferConn{}
		tc := newTestConn(conn)
		tc.prefaceSent = true
		tc.wmu.Lock()
		defer tc.wmu.Unlock()

		data := rawFrame(xhttp2.FrameData, 0, 1, []byte("hello"))
		tc.wscan.scan(data[:frameHeaderLen+2], noFrame)
		require.NoError(t, tc.writeLocked(goAwayFrame(maxStreamID)))
		require.NoError(t, tc.writeLocked(pingFrame()))
		assert.Empty(t, conn.calls, "must not write within a frame")

		tc.wscan.scan(data[frameHeaderLen+2:], noFrame)
		require.True(t, tc.wscan.atBoundary())
		require.NoError(t, tc.flushLocked())
		assert.Equal(t, append(goAwayFrame(maxStreamID), pingFrame()...), conn.buf.Bytes(),
			"must write the pending frames in order, at once")
		assert.Equal(t, []string{"Write"}, conn.calls)
		assert.Empty(t, tc.pending)
	})

	t.Run("before the server preface", func(t *testing.T) {
		conn := &bufferConn{}
		tc := newTestConn(conn)
		tc.wmu.Lock()
		defer tc.wmu.Unlock()

		require.NoError(t, tc.writeLocked(goAwayFrame(1)))
		assert.Empty(t, conn.calls, "must not write ahead of the server SETTINGS")
		assert.Equal(t, goAwayFrame(1), tc.pending)
	})

	t.Run("nothing pending", func(t *testing.T) {
		conn := &bufferConn{}
		tc := newTestConn(conn)
		tc.prefaceSent = true
		tc.wmu.Lock()
		defer tc.wmu.Unlock()

		require.NoError(t, tc.flushLocked())
		require.NoError(t, tc.writeLocked(nil))
		assert.Empty(t, conn.calls, "must not write without pending frames")
	})

	t.Run("write error", func(t *testing.T) {
		wantErr := errors.New("great sadness")
		conn := &bufferConn{stubConn: stubConn{err: wantErr}}
		tc := newTestConn(conn)
		tc.prefaceSent = true
		tc.wmu.Lock()
		defer tc.wmu.Unlock()

		assert.Equal(t, wantErr, tc.writeLocked(goAwayFrame(1)), "must return the connection's error")
		assert.Empty(t, tc.pending, "must not retry failed frames")
	})
}

func TestTrackedConnGoAwayFramesLocked(t *testing.T) {
	tests := []struct {
		from, to goAwayState
		want     []byte
	}{
		{goAwayNone, goAwayNone, nil},
		{goAwayNone, goAwayFirst, firstGoAway()},
		{goAwayNone, goAwaySecond, append(firstGoAway(), goAwayFrame(9)...)},
		{goAwayFirst, goAwayFirst, nil},
		{goAwayFirst, goAwaySecond, goAwayFrame(9)},
		{goAwaySecond, goAwayFirst, nil},
		{goAwaySecond, goAwaySecond, nil},
	}
	for _, tt := range tests {
		tc := newTestConn(&stubConn{})
		tc.lastStreamID = 9
		tc.goAway = tt.from

		assert.Equal(t, tt.want, tc.goAwayFramesLocked(tt.to), "from %v to %v", tt.from, tt.to)
		assert.Equal(t, max(tt.from, tt.to), tc.goAway, "from %v to %v: must never move back", tt.from, tt.to)
		assert.Empty(t, tc.goAwayFramesLocked(tt.to), "from %v to %v: must write each GOAWAY once", tt.from, tt.to)
	}
}

func TestTrackedConnDrain(t *testing.T) {
	t.Run("PING acknowledged", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{}, headersFrame(3))
		close(tc.pingAcked)

		tc.drain()
		assert.Equal(t, append(firstGoAway(), goAwayFrame(3)...), conn.written.Bytes())
	})

	t.Run("closed", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{}, headersFrame(3))
		require.NoError(t, tc.Close())
		conn.written.Reset()

		start := time.Now()
		tc.drain()
		assert.Less(t, time.Since(start), pingAckTimeout, "must not wait for a PING ACK once closed")
		assert.Equal(t, firstGoAway(), conn.written.Bytes(), "must not write the second GOAWAY once closed")
	})

	t.Run("write error", func(t *testing.T) {
		tc, conn := newTrackedFakeConn(t, &recordingHelper{}, headersFrame(3))
		conn.failWrite = 1

		start := time.Now()
		tc.drain()
		assert.Less(t, time.Since(start), pingAckTimeout, "must not wait for a PING ACK")
		assert.Equal(t, 1, conn.writes, "must give up on error")
		assert.Equal(t, goAwayFirst, tc.goAway, "must not write the first GOAWAY again")
	})
}
