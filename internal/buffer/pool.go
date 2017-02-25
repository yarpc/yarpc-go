package buffer

import (
	"bytes"
	"sync"
)

var _pool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// Get returns a new Byte Buffer from the buffer pool
// that has been reset
func Get() *bytes.Buffer {
	buf := _pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns byte buffer to the buffer pool
func Put(buf *bytes.Buffer) {
	_pool.Put(buf)
}
