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
	"errors"
	"io"

	"go.uber.org/yarpc/internal/buffer"

	"go.uber.org/atomic"
)

// TODO Optimizations:
// - Attempt to cast the source reader to a *bytes.Buffer and use it directly
//   instead of needing to copy into a new buffer.
// - In cases where the first reader was not completely read when we call
//   reset we should flush the rest of the reader into the buffer, or
//   combine the buffer and source reader together dynamically.

// NewRereader returns a new re-reader and closer that wraps the given reader.
// The re-reader consumes the given reader on demand, recording the entire
// stream so it can be replayed. The re-reader is suitable for retries. The
// re-reader does not support concurrent consumers, as would be necessary
// for speculative retries.
func NewRereader(src io.Reader) (*Rereader, func()) {
	buf := buffer.Get()
	return &Rereader{
		teeReader: io.TeeReader(src, buf),
		buf:       buf,
	}, func() { buffer.Put(buf) }
}

// Rereader has the ability to read the same io.Reader multiple times.
// There are currently some limitations to the design:
//   - Reset MUST be called in order to Re-Read the reader.
//   - Reset MUST be called after the ReReader is exausted (Read will return
//     an io.EOF error).
//   - Concurrent reads are not supported.
type Rereader struct {
	// teeReader is an io.TeeReader that will read from the source io.Reader
	// and simultaneously return the result and write all the data to the
	// buf attribute to be replayed.
	teeReader io.Reader

	// buf will be filled with the contents from the teeReader as we read
	// from the tee reader.
	buf *bytes.Buffer

	// Unfortunately we can't reset a bytes.Buffer to reread from it, so after
	// our initial read into the buffer, we need to use a reader to read the
	// bytes out of the buffer.
	bufReader *bytes.Reader

	// In order to test whether the reader is finished, we need to call `Read`
	// and validate that it returns an io.EOF error.  This buffer is used so
	// we don't need to reallocate a slice for every `Read` check.
	testbuf [1]byte

	// useBuffer indicates whether reads should go to the teeReader or the
	// bufReader.
	useBuffer atomic.Bool
}

// Read implements the io.Reader interface.  On the first read, we will record
// the entire contents of the source io.Reader for subsequent Reads.
func (rr *Rereader) Read(p []byte) (n int, err error) {
	if rr.useBuffer.Load() {
		return rr.bufReader.Read(p)
	}
	return rr.teeReader.Read(p)
}

// Reset resets the rereader to read from the beginning of the source data.
//   - If the current Read is not finished (a call to `Read` does not
//     return an io.EOF error), Reset will return an error.
func (rr *Rereader) Reset() error {
	if _, err := rr.Read(rr.testbuf[:]); err != io.EOF {
		return errors.New("cannot reset the rereader until we've finished reading the current reader")
	}

	if !rr.useBuffer.Load() {
		rr.bufReader = bytes.NewReader(rr.buf.Bytes())
		rr.useBuffer.Store(true)
		return nil
	}

	_, err := rr.bufReader.Seek(0, io.SeekStart)
	return err
}
