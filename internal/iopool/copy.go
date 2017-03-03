package iopool

import "io"

// Copy wraps the io library's CopyBuffer func with a preallocated buffer from a
// sync.Pool we maintain.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	buf := get()
	written, err := io.CopyBuffer(dst, src, buf.b)
	put(buf)
	return written, err
}
