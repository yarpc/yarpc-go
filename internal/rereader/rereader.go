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

package rereader

import (
	"bytes"
	"errors"
	"io"

	"go.uber.org/yarpc/internal/buffer"

	"github.com/uber-go/atomic"
)

// TODO Optimizations:
// - Attempt to cast the source reader to a *bytes.Buffer and use it directly
//   instead of needing to copy into a new buffer.
// - In cases where the first reader was not completely read when we call
//   reset we should flush the rest of the reader into the buffer, or
//   combine the buffer and source reader together dynamically.

// New will return a new ReReader and a closer to cleanup the ReReader once
// the callsite is done with it.
func New(source io.Reader) (*ReReader, func()) {
	buf := buffer.Get()
	closer := func() {
		buffer.Put(buf)
	}
	return &ReReader{
		src: source,
		buf: buf,
	}, closer
}

// ReReader has the ability to read the same io.Reader multiple times.
// There are currently some limitations to the design:
//   - Reset MUST be called in order to Re-Read the reader.
//   - Reset MUST be called after the ReReader is exausted (Read will return
//     an io.EOF error).
type ReReader struct {
	src    io.Reader
	buf    *bytes.Buffer
	reader *bytes.Reader

	// Small buffer required to test whether the reader is exausted (EOF).
	testbuf [1]byte

	useBuffer atomic.Bool
}

// Read implements the io.Reader interface.
func (rr *ReReader) Read(p []byte) (n int, err error) {
	if rr.useBuffer.Load() {
		return rr.reader.Read(p)
	}

	n, err = rr.src.Read(p)
	if n > 0 {
		if n, err := rr.buf.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

// Reset validates that the current Reader is finished (EOF) before resetting
// the Reader to read from the beginning of the source reader again.
func (rr *ReReader) Reset() error {
	if _, err := rr.Read(rr.testbuf[:]); err != io.EOF {
		return errors.New("cannot reset the rereader until we've finished reading the current reader")
	}

	if !rr.useBuffer.Load() {
		rr.reader = bytes.NewReader(rr.buf.Bytes())
		rr.useBuffer.Store(true)
		return nil
	}

	_, err := rr.reader.Seek(0, io.SeekStart)
	return err
}
