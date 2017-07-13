// Copyright (c) 2017 Uber Technologies, Inc.
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

package ioutil

import (
	"bytes"
	"io"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/internal/bufferpool"
)

// NewRereader returns a new re-reader and closer that wraps the given reader.
// The re-reader consumes the given reader on demand, recording the entire
// stream so it can be replayed. The re-reader is suitable for retries. The
// re-reader does not support concurrent consumers, as would be necessary
// for speculative retries or fanout requests.
func NewRereader(src io.Reader) (*Rereader, func()) {
	// If the src is already a rereader, the api should be safe to reuse. as
	// long as there will be a single request at a time.
	if rr, ok := src.(*Rereader); ok {
		return rr, func() {}
	}

	// If the src is a *bytes.Buffer, we don't need to copy the src into a buffer
	// and can use the buffer directly.
	if bb, ok := src.(*bytes.Buffer); ok {
		return &Rereader{
			buf: bb,
			bufReader: bytes.NewReader(bb.Bytes()),
			useBuffer: atomic.NewBool(true),
			hasReadFromSrc: atomic.NewBool(false),
		}, func () {}
	}

	buf := bufferpool.Get()
	return &Rereader{
		src:       src,
		buf:       buf,
		useBuffer: atomic.NewBool(false),
		hasReadFromSrc: atomic.NewBool(false),
	}, func() { bufferpool.Put(buf) }
}

// Rereader has the ability to read the same io.Reader multiple times.
// There are currently some limitations to the design:
//   - Reset MUST be called in order to Re-Read the reader.
//   - Reset MUST be called after the ReReader is exausted (Read will return
//     an io.EOF error).
//   - Concurrent reads are not supported.
type Rereader struct {
	// src is the source io.Reader.  When we read from it, we will tee it's
	// data into a bytes buffer for reuse. (If the src was originally a bytes
	// buffer, we will skip the copy step and use the buffer immediately).
	src io.Reader

	// buf will be filled with the contents from the src reader as we read
	// from it.  After the initial read, it will be the source of truth.
	buf *bytes.Buffer

	// Unfortunately we can't reset a bytes.Buffer to reread from it, so after
	// our initial read into the buffer, we need to use a reader to read the
	// bytes out of the buffer.
	bufReader *bytes.Reader

	// hasReadFromSrc indicates whether anything has been read from the src
	// io.Reader.
	hasReadFromSrc *atomic.Bool

	// useBuffer indicates whether reads should go to the teeReader or the
	// bufReader.
	useBuffer *atomic.Bool
}

// Read implements the io.Reader interface.  On the first read, we will record
// the entire contents of the source io.Reader for subsequent Reads.
func (rr *Rereader) Read(p []byte) (n int, err error) {
	if rr.useBuffer.Load() {
		return rr.bufReader.Read(p)
	}

	rr.hasReadFromSrc.Store(true)
	n, err = rr.src.Read(p)
	if n > 0 {
		if n, err := rr.buf.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

// Reset resets the rereader to read from the beginning of the source data.
// If the src has not finished reading, we will read the rest of the src data
// into the buffer.
func (rr *Rereader) Reset() error {
	// If we're reading from a buffer, we can reset the buffer reader
	// immediately.
	if rr.useBuffer.Load() {
		_, err := rr.bufReader.Seek(0, io.SeekStart)
		return err
	}

	// If we haven't read a single byte from the src reader, we don't need to
	// reset anything and can reuse the reader.
	if !rr.hasReadFromSrc.Load() {
		return nil
	}

	// Ensure we've filled the buffer by reading the rest of the src.
	if _, err := rr.buf.ReadFrom(rr.src); err != nil {
		return err
	}

	rr.bufReader = bytes.NewReader(rr.buf.Bytes())
	rr.useBuffer.Store(true)
	return nil
}