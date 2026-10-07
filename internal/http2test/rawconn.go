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

// Package http2test provides a frame-level HTTP/2 cleartext (h2c) client for
// tests that need to observe the frames a server writes on the wire.
//
// It exists because http2.Transport surfaces a GOAWAY only as an opaque error,
// which cannot express assertions about frame ordering or LastStreamID.
package http2test

import (
	"bytes"
	"net"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// Frame is a copy of a frame read from the server, decoupled from the
// http2.Framer buffers so that it stays valid after subsequent reads.
type Frame struct {
	Type     http2.FrameType
	StreamID uint32

	// EndStream is set on HEADERS and DATA frames that close the stream.
	EndStream bool
	// Headers holds the decoded header block of a HEADERS frame.
	Headers map[string]string
	// Data holds the payload of a DATA frame.
	Data []byte

	// LastStreamID is set on GOAWAY frames.
	LastStreamID uint32
	// ErrCode is set on GOAWAY and RST_STREAM frames.
	ErrCode http2.ErrCode
}

// RawH2CConn is a frame-level h2c client. A background goroutine reads every
// frame the server writes, acknowledges SETTINGS and PING on the client's
// behalf, and publishes the rest on Frames.
type RawH2CConn struct {
	addr   string
	conn   net.Conn
	framer *http2.Framer

	writeMu sync.Mutex // guards writes on framer
	encBuf  bytes.Buffer
	enc     *hpack.Encoder

	frames     chan Frame
	stop       chan struct{} // closed when the test ends, unblocking readLoop
	readerDone chan struct{}
	readErr    error // set before frames is closed
}

// DialRawH2C opens an h2c connection with prior knowledge to addr: it writes
// the client preface and an empty SETTINGS frame, and starts reading frames.
// The connection is closed when the test ends.
func DialRawH2C(t testing.TB, addr string) *RawH2CConn {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err, "failed to dial %v", addr)

	c := &RawH2CConn{
		addr:       addr,
		conn:       conn,
		framer:     http2.NewFramer(conn, conn),
		frames:     make(chan Frame, 64),
		stop:       make(chan struct{}),
		readerDone: make(chan struct{}),
	}
	c.enc = hpack.NewEncoder(&c.encBuf)
	c.framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	t.Cleanup(func() {
		close(c.stop)
		_ = c.conn.Close()
		<-c.readerDone
	})

	_, err = conn.Write([]byte(http2.ClientPreface))
	require.NoError(t, err, "failed to write client preface")
	require.NoError(t, c.write(func(f *http2.Framer) error { return f.WriteSettings() }),
		"failed to write client SETTINGS")

	go c.readLoop()
	return c
}

// OpenStream writes a request on a new stream: a HEADERS frame carrying the
// POST pseudo-headers for path plus headers, followed by body in a DATA frame
// that ends the stream.
func (c *RawH2CConn) OpenStream(t testing.TB, streamID uint32, path string, headers map[string]string, body []byte) {
	t.Helper()

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	err := c.write(func(f *http2.Framer) error {
		c.encBuf.Reset()
		fields := []hpack.HeaderField{
			{Name: ":method", Value: "POST"},
			{Name: ":scheme", Value: "http"},
			{Name: ":authority", Value: c.addr},
			{Name: ":path", Value: path},
		}
		for _, k := range keys {
			fields = append(fields, hpack.HeaderField{Name: k, Value: headers[k]})
		}
		for _, hf := range fields {
			if err := c.enc.WriteField(hf); err != nil {
				return err
			}
		}
		if err := f.WriteHeaders(http2.HeadersFrameParam{
			StreamID:      streamID,
			BlockFragment: c.encBuf.Bytes(),
			EndHeaders:    true,
		}); err != nil {
			return err
		}
		return f.WriteData(streamID, true, body)
	})
	require.NoError(t, err, "failed to open stream %d", streamID)
}

// Frames returns the frames read from the server, in order. SETTINGS and
// PING frames are acknowledged and not published; PING acknowledgements are.
// The channel is closed once the connection ends.
func (c *RawH2CConn) Frames() <-chan Frame { return c.frames }

// ReadUntil reads frames until one satisfies match and returns every frame
// read, ending with the matching one. The test fails if the connection ends
// first, or if nothing matches within timeout.
func (c *RawH2CConn) ReadUntil(t testing.TB, timeout time.Duration, match func(Frame) bool) []Frame {
	t.Helper()

	var frames []Frame
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case f, ok := <-c.frames:
			if !ok {
				require.FailNow(t, "connection ended before the expected frame",
					"read error: %v, frames read: %+v", c.readErr, frames)
			}
			frames = append(frames, f)
			if match(f) {
				return frames
			}
		case <-deadline.C:
			require.FailNow(t, "timed out waiting for the expected frame", "frames read: %+v", frames)
		}
	}
}

// ReadUntilClosed reads frames until the connection ends and returns them.
// The test fails if the connection is still open after timeout.
func (c *RawH2CConn) ReadUntilClosed(t testing.TB, timeout time.Duration) []Frame {
	t.Helper()

	var frames []Frame
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case f, ok := <-c.frames:
			if !ok {
				return frames
			}
			frames = append(frames, f)
		case <-deadline.C:
			require.FailNow(t, "timed out waiting for the connection to end", "frames read: %+v", frames)
		}
	}
}

// Ping writes a PING frame; the server's acknowledgement shows up on Frames
// as a PING frame, which IsPingAck matches.
func (c *RawH2CConn) Ping(t testing.TB) {
	t.Helper()
	require.NoError(t, c.write(func(f *http2.Framer) error { return f.WritePing(false, [8]byte{}) }),
		"failed to write PING")
}

// Close closes the client side of the connection.
func (c *RawH2CConn) Close() error { return c.conn.Close() }

func (c *RawH2CConn) write(fn func(*http2.Framer) error) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return fn(c.framer)
}

func (c *RawH2CConn) readLoop() {
	defer close(c.readerDone)
	defer close(c.frames)

	for {
		f, err := c.framer.ReadFrame()
		if err != nil {
			c.readErr = err
			return
		}

		var out Frame
		switch f := f.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				_ = c.write(func(fr *http2.Framer) error { return fr.WriteSettingsAck() })
			}
			continue
		case *http2.PingFrame:
			if !f.IsAck() {
				_ = c.write(func(fr *http2.Framer) error { return fr.WritePing(true, f.Data) })
				continue
			}
		case *http2.MetaHeadersFrame:
			out.EndStream = f.StreamEnded()
			out.Headers = make(map[string]string, len(f.Fields))
			for _, hf := range f.Fields {
				out.Headers[hf.Name] = hf.Value
			}
		case *http2.DataFrame:
			out.EndStream = f.StreamEnded()
			out.Data = append([]byte(nil), f.Data()...)
		case *http2.GoAwayFrame:
			out.LastStreamID = f.LastStreamID
			out.ErrCode = f.ErrCode
		case *http2.RSTStreamFrame:
			out.ErrCode = f.ErrCode
		}
		out.Type = f.Header().Type
		out.StreamID = f.Header().StreamID
		select {
		case c.frames <- out:
		case <-c.stop:
			return
		}
	}
}

// IsGoAway reports whether f is a GOAWAY frame.
func IsGoAway(f Frame) bool { return f.Type == http2.FrameGoAway }

// IsPingAck reports whether f acknowledges a PING sent with Ping.
func IsPingAck(f Frame) bool { return f.Type == http2.FramePing }

// StreamDone returns a matcher for the frame that ends or resets the given
// stream.
func StreamDone(streamID uint32) func(Frame) bool {
	return func(f Frame) bool {
		return f.StreamID == streamID && (f.EndStream || f.Type == http2.FrameRSTStream)
	}
}

// Response is the server's response on one stream, assembled from frames.
type Response struct {
	// Headers holds the first header block; Trailers the second, if any.
	Headers  map[string]string
	Trailers map[string]string
	Body     []byte
	// Complete is set if the server ended the stream.
	Complete bool
	// Reset is set if the server reset the stream, with ResetCode.
	Reset     bool
	ResetCode http2.ErrCode
}

// ResponseOn assembles the response on streamID from frames.
func ResponseOn(frames []Frame, streamID uint32) Response {
	var r Response
	for _, f := range frames {
		if f.StreamID != streamID {
			continue
		}
		switch f.Type {
		case http2.FrameHeaders:
			if r.Headers == nil {
				r.Headers = f.Headers
			} else {
				r.Trailers = f.Headers
			}
		case http2.FrameData:
			r.Body = append(r.Body, f.Data...)
		case http2.FrameRSTStream:
			r.Reset = true
			r.ResetCode = f.ErrCode
		}
		if f.EndStream {
			r.Complete = true
		}
	}
	return r
}
