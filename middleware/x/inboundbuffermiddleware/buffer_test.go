// Copyright (c) 2020 Uber Technologies, Inc.
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

package inboundbuffermiddleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The buffer constructor is only needed for tests since the buffer is inlined
// into the parent struct otherwise.
func newBuffer(capacity int) *buffer {
	b := new(buffer)
	b.Init(capacity)
	return b
}

func TestBufferDegenerate(t *testing.T) {
	b := newBuffer(0)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.Put(0, 0))
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.Put(0, 0))
}

func TestBufferTrivial(t *testing.T) {
	b := newBuffer(1)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.length)

	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, 0, b.length)
	assert.Equal(t, -1, b.Pop())
}

func TestBufferFavorHigherPriorityLeft(t *testing.T) {
	b := newBuffer(2)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.Put(0, 1))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 1, b.Pop())
	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, -1, b.Pop())
}

func TestBufferFavorHigherPriorityRight(t *testing.T) {
	b := newBuffer(2)
	assert.Equal(t, -1, b.Pop())

	assert.Equal(t, 0, b.Put(0, 1))
	assert.Equal(t, 1, b.Put(0, 0))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, 1, b.Pop())
	assert.Equal(t, -1, b.Pop())
}

func TestBufferDeadlineEviction(t *testing.T) {
	b := newBuffer(3)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.EvictExpired(0))

	assert.Equal(t, 0, b.Put(0, 0))
	assert.Equal(t, 1, b.Put(1, 0))
	assert.Equal(t, 2, b.Put(2, 0))
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, 0, b.EvictExpired(1))
	assert.Equal(t, 1, b.EvictExpired(1))

	assert.Equal(t, -1, b.EvictExpired(1))

	assert.Equal(t, 2, b.EvictExpired(2))
	assert.Equal(t, -1, b.EvictExpired(2))
}

func TestBufferPriorityEviction(t *testing.T) {
	b := newBuffer(3)
	assert.Equal(t, -1, b.Pop())
	assert.Equal(t, -1, b.EvictExpired(0))

	assert.Equal(t, 0, b.Put(0, 3))
	// {0:3}
	assert.Equal(t, 1, b.Put(0, 1))
	// {0:3} {1:1}
	assert.Equal(t, 2, b.Put(0, 2))
	// {0:3} {2:2} {1:1}
	assert.Equal(t, -1, b.Put(0, 0))

	assert.Equal(t, -1, b.EvictLowerPriority(0))
	assert.Equal(t, -1, b.EvictLowerPriority(1))
	assert.Equal(t, 1, b.EvictLowerPriority(2))
	// {0:3} {2:2}

	assert.Equal(t, 2, b.EvictLowerPriority(3))
	// {0:3}

	assert.Equal(t, -1, b.EvictLowerPriority(3))
	assert.Equal(t, 0, b.Pop())
	assert.Equal(t, -1, b.EvictLowerPriority(0))
	assert.Equal(t, -1, b.Pop())
}
