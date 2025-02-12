// Copyright (c) 2025 Uber Technologies, Inc.
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

package bufferpool

import (
	"bytes"
	"io"
)

// Buffer represents a poolable buffer. It wraps an underlying
// *bytes.Buffer with lightweight detection of races.
type Buffer struct {
	pool *Pool

	// version is an ever-incrementing integer on every operation.
	// it ensures that we don't perform multiple overlapping operations.
	version uint

	// released tracks whether the buffer has been released.
	released bool

	buf *bytes.Buffer
}

func newBuffer(pool *Pool) *Buffer {
	return &Buffer{
		pool: pool,
		buf:  &bytes.Buffer{},
	}
}

func (b *Buffer) checkUseAfterFree() {
	if b.released || b.buf == nil {
		panic("use-after-free of pooled buffer")
	}
}

func (b *Buffer) preOp() uint {
	b.checkUseAfterFree()
	b.version++
	return b.version
}

func (b *Buffer) postOp(v uint) {
	b.checkUseAfterFree()
	if v != b.version || b.released {
		panic("concurrent use of pooled buffer")
	}
	b.version++
}

// Read is the same as bytes.Buffer.Read.
func (b *Buffer) Read(p []byte) (int, error) {
	version := b.preOp()
	n, err := b.buf.Read(p)
	b.postOp(version)
	return n, err
}

// ReadFrom is the same as bytes.Buffer.ReadFrom.
func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	version := b.preOp()
	n, err := b.buf.ReadFrom(r)
	b.postOp(version)
	return n, err
}

// Write is the same as bytes.Buffer.Write.
func (b *Buffer) Write(p []byte) (int, error) {
	version := b.preOp()
	n, err := b.buf.Write(p)
	b.postOp(version)
	return n, err
}

// WriteTo is the same as bytes.Buffer.WriteTo.
func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	version := b.preOp()
	n, err := b.buf.WriteTo(w)
	b.postOp(version)
	return n, err
}

// Bytes returns the bytes in the underlying buffer, as well as a
// function to call when the caller is done using the bytes.
// This is easy to mis-use and lead to a use-after-free that
// cannot be detected, so it is strongly recommended that this method
// is NOT used.
func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}

// Len is the same as bytes.Buffer.Len.
func (b *Buffer) Len() int {
	version := b.preOp()
	n := b.buf.Len()
	b.postOp(version)
	return n
}

// Reset is the same as bytes.Buffer.Reset.
func (b *Buffer) Reset() {
	version := b.preOp()
	b.buf.Reset()
	b.postOp(version)
}

// Release releases the buffer back to the buffer pool.
func (b *Buffer) Release() {
	// Increment the version so overlapping operations fail.
	b.postOp(b.preOp())

	if b.pool.testDetectUseAfterFree {
		b.releaseDetectUseAfterFree()
		return
	}

	// Before releasing a buffer, we should reset it to "clear" the buffer
	// while holding on to the capacity of the buffer.
	b.Reset()

	// We must mark released after the `Reset`, so that `Reset` doesn't
	// trigger use-after-free.
	b.released = true

	b.pool.release(b)
}

func (b *Buffer) reuse() {
	b.released = false
}

func (b *Buffer) releaseDetectUseAfterFree() {
	// Detect any lingering reads of the underlying data by resetting the data.
	// We repeat it in a goroutine to trigger the race detector.
	overwriteData(b.Bytes())
	go overwriteData(b.Bytes())

	// This will cause any future accesses to panic.
	b.released = true
	b.buf = nil
}

func overwriteData(bs []byte) {
	for i := range bs {
		bs[i] = byte(i)
	}
}

// AutoReleaseBuffer wraps a Buffer in a io.ReadCloser implementation
// that returns the underlying Buffer to the pool on Close().
type AutoReleaseBuffer struct {
	*Buffer
}

// NewAutoReleaseBuffer creates a AutoReleaseBuffer
func NewAutoReleaseBuffer() AutoReleaseBuffer {
	buf := Get()
	return AutoReleaseBuffer{
		Buffer: buf,
	}
}

// Close returns the buffer to the pool.
func (arb AutoReleaseBuffer) Close() error {
	Put(arb.Buffer)
	return nil
}
