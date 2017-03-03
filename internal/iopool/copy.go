package iopool

import (
	"io"
	"sync"
)

type buffer struct {
	b []byte
}

var _copyBufSize = 1024 * 32
var _pool = sync.Pool{
	New: func() interface{} {
		return &buffer{make([]byte, _copyBufSize)}
	},
}

// Copy wraps the io library's CopyBuffer func with a preallocated buffer from a
// sync.Pool we maintain.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	buf := _pool.Get().(*buffer)
	written, err := io.CopyBuffer(dst, src, buf.b)
	_pool.Put(buf)
	return written, err
}
