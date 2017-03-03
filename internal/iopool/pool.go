package iopool

import "sync"

var _maxCapacity = 1024 * 50 // The max capacity for a buffer is 32 KiB
var _defaultCapacity = 1024 * 32
var _pool = sync.Pool{
	New: func() interface{} {
		return newBuffer(_defaultCapacity)
	},
}

// Get returns a new Byte Buffer from the buffer pool
// that has been reset
func get() *buffer {
	buf := _pool.Get().(*buffer)
	return buf
}

// Put returns byte buffer to the buffer pool
func put(buf *buffer) {
	if cap(buf.b) < _maxCapacity {
		_pool.Put(buf)
	}
}
