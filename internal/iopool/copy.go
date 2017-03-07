package iopool

import (
	"io"
	"sync"
)

type buffer struct {
	b []byte
}

const _copyBufSize = 1024 * 32

var _pool = sync.Pool{
	New: func() interface{} {
		return &buffer{make([]byte, _copyBufSize)}
	},
}

// Copy copies bytes from the Reader to the Writer until the Reader is exhausted.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	// To avoid unecessary memory allocations we maintain our own pool of
	// buffers.
	buf := _pool.Get().(*buffer)
	written, err := io.CopyBuffer(dst, src, buf.b)
	_pool.Put(buf)
	return written, err
}
