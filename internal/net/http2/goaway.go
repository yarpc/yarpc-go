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

// Package http2 provides functionality useful for HTTP/2 transport.
// This source file contains details for sending HTTP/2 GOAWAY frames correctly.
package http2

import (
	"bytes"
	"encoding/binary"
	"time"

	xhttp2 "golang.org/x/net/http2"
)

const (
	frameHeaderLen = 9
	maxStreamID    = 1<<31 - 1

	// pingAckTimeout bounds how long the second GOAWAY waits for the client
	// to acknowledge the PING sent with the first one.
	pingAckTimeout = time.Second

	// closeWriteTimeout bounds writing the GOAWAY frames still owed on Close.
	closeWriteTimeout = 100 * time.Millisecond
)

// goAwayState encodes a connection's state during its two-phase graceful shutdown.
type goAwayState int

const (
	goAwayNone   goAwayState = iota // Not in shutdown yet.
	goAwayFirst                     // GOAWAY with last stream ID 2^31-1, then a PING, sent.
	goAwaySecond                    // GOAWAY with the last stream ID read sent; final state.
)

// pingData tells the ACK of the PING sent with the first GOAWAY apart from
// the ACKs of PINGs the HTTP/2 server sends itself.
var pingData = [8]byte{'y', 'a', 'r', 'p', 'c', 'g', 'o', 'a'}

// goAwayFrame encodes a GOAWAY frame with NO_ERROR (RFC 9113 §6.8), with the
// x/net/http2 framer that net/http's HTTP/2 implementation is copied from.
func goAwayFrame(lastStreamID uint32) []byte {
	var buf bytes.Buffer
	_ = xhttp2.NewFramer(&buf, nil).WriteGoAway(lastStreamID, xhttp2.ErrCodeNo, nil)
	return buf.Bytes()
}

// pingFrame encodes the PING frame sent with the first GOAWAY.
func pingFrame() []byte {
	var buf bytes.Buffer
	_ = xhttp2.NewFramer(&buf, nil).WritePing(false, pingData)
	return buf.Bytes()
}

// frameScanner follows HTTP/2 frame boundaries in a byte stream.
// It only keeps frame headers, and the payload of PING frames.
type frameScanner struct {
	buf  [frameHeaderLen + 8]byte
	n    int // bytes of buf filled
	skip int // payload bytes left in the current frame
}

// scan calls fn with every frame header completed in p, followed by its
// payload for PING frames, and with the offset in p where the frame started:
// negative if it started in an earlier call.
func (s *frameScanner) scan(p []byte, fn func(start int, frame []byte)) {
	for off := 0; off < len(p); {
		if s.skip > 0 {
			k := min(s.skip, len(p)-off)
			s.skip -= k
			off += k
			continue
		}
		want := frameHeaderLen
		if s.n >= frameHeaderLen {
			want = len(s.buf)
		}
		k := copy(s.buf[s.n:want], p[off:])
		s.n += k
		off += k
		if s.n < frameHeaderLen {
			return
		}
		length := int(s.buf[0])<<16 | int(s.buf[1])<<8 | int(s.buf[2])
		if xhttp2.FrameType(s.buf[3]) == xhttp2.FramePing && length == len(pingData) && s.n < len(s.buf) {
			continue
		}
		fn(off-s.n, s.buf[:s.n])
		s.skip = length - (s.n - frameHeaderLen)
		s.n = 0
	}
}

// atBoundary reports whether the scanner is at a frame boundary,
// i.e., not in the middle of a frame.
func (s *frameScanner) atBoundary() bool {
	return s.n == 0 && s.skip == 0
}

// frameRead handles a fully read HTTP/2 frame.
// It updates the last stream ID.
func (tc *TrackedConn) frameRead(_ int, frame []byte) {
	switch xhttp2.FrameType(frame[3]) {
	case xhttp2.FrameHeaders:
		if id := binary.BigEndian.Uint32(frame[5:]) & maxStreamID; id%2 == 1 {
			tc.idMu.Lock()
			tc.lastStreamID = max(id, tc.lastStreamID)
			tc.idMu.Unlock()
		}
	case xhttp2.FramePing:
		if xhttp2.Flags(frame[4]).Has(xhttp2.FlagPingAck) &&
			len(frame) == frameHeaderLen+len(pingData) && [8]byte(frame[frameHeaderLen:]) == pingData {
			tc.ackOnce.Do(func() { close(tc.pingAcked) })
		}
	}
}

// frameWrite handles an HTTP/2 frame about to be written, the first of which
// starts the server's connection preface.
// A GOAWAY frame of the server's own takes the place of the second GOAWAY,
// whether shutting down or not: the server may start its own shutdown first.
// So it sets where Write splits to write the first GOAWAY ahead of it.
// Note: only thread-safe if tc.wmu is held.
func (tc *TrackedConn) frameWrite(start int, frame []byte) {
	tc.prefaceSent = true
	if xhttp2.FrameType(frame[3]) != xhttp2.FrameGoAway || tc.split >= 0 {
		return
	}
	if start < 0 {
		// Started in an earlier Write: nothing can go ahead of it.
		tc.pending = nil
	} else {
		tc.split = start
		tc.pending = append(tc.pending, tc.goAwayFramesLocked(goAwayFirst)...)
	}
	tc.goAway = goAwaySecond
}

// drain runs the two-phase graceful shutdown of RFC 9113 §6.8: a GOAWAY with
// last stream ID 2^31-1 and a PING, then, once the client acknowledges the
// PING, a GOAWAY with the last stream ID read by then.
func (tc *TrackedConn) drain() {
	if tc.advance(goAwayFirst) != nil {
		return
	}
	select {
	case <-tc.pingAcked:
	case <-tc.closed:
		return
	case <-time.After(pingAckTimeout):
	}
	_ = tc.advance(goAwaySecond)
}

// advance writes the GOAWAY frames that move tc's shutdown to target.
func (tc *TrackedConn) advance(target goAwayState) error {
	tc.wmu.Lock()
	defer tc.wmu.Unlock()
	return tc.writeLocked(tc.goAwayFramesLocked(target))
}

// writeOwedFrames writes the GOAWAY frames tc still owes, so that both are
// sent before it closes. It does not wait on a write in progress.
func (tc *TrackedConn) writeOwedFrames() {
	if !tc.wmu.TryLock() {
		return
	}
	defer tc.wmu.Unlock()
	_ = tc.conn.SetWriteDeadline(time.Now().Add(closeWriteTimeout))
	_ = tc.writeLocked(tc.goAwayFramesLocked(goAwaySecond))
}

// goAwayFramesLocked moves tc's shutdown to target, and returns the frames
// that takes, so that each GOAWAY is written at most once, and in order.
// Note: only thread-safe if tc.wmu is held.
func (tc *TrackedConn) goAwayFramesLocked(target goAwayState) []byte {
	var frames []byte
	for ; tc.goAway < target; tc.goAway++ {
		switch tc.goAway {
		case goAwayNone:
			frames = append(frames, goAwayFrame(maxStreamID)...)
			frames = append(frames, pingFrame()...)
		case goAwayFirst:
			tc.idMu.Lock()
			lastStreamID := tc.lastStreamID
			tc.idMu.Unlock()
			frames = append(frames, goAwayFrame(lastStreamID)...)
		}
	}
	return frames
}

// atBoundaryLocked reports whether frames can be written now: between two
// frames, once the server started its connection preface, as its SETTINGS
// frame MUST come first (RFC 9113 §3.4).
// Note: only thread-safe if tc.wmu is held.
func (tc *TrackedConn) atBoundaryLocked() bool {
	return tc.prefaceSent && tc.wscan.atBoundary()
}

// writeLocked writes frames now if between two frames, or at the next frame
// boundary otherwise.
func (tc *TrackedConn) writeLocked(frames []byte) error {
	tc.pending = append(tc.pending, frames...)
	if !tc.atBoundaryLocked() {
		return nil
	}
	return tc.flushLocked()
}

// flushLocked writes any pending frames to the underlying connection.
// Note: only thread-safe if tc.wmu is held.
func (tc *TrackedConn) flushLocked() error {
	if len(tc.pending) == 0 {
		return nil
	}
	_, err := tc.conn.Write(tc.pending)
	tc.pending = nil
	return err
}
