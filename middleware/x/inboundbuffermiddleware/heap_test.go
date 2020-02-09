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

func jotUint64(num int) []uint64 {
	slice := make([]uint64, num)
	for i := 0; i < num; i++ {
		slice[i] = uint64(i)
	}
	return slice
}

func TestHeap(t *testing.T) {
	size := 10

	values := jotUint64(size)
	heap := jot(size)
	coHeap := jot(size)

	t.Run("down", func(t *testing.T) {
		for i := 0; i < size-1; i++ {
			values[i] = maxUint64
			fixHeap(minHeap, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i+1), values[heap[0]])
		}

		values[size-1] = maxUint64
		fixHeap(minHeap, size, values, heap, coHeap, size-1)
		assert.Equal(t, maxUint64, values[heap[0]])
	})

	t.Run("up", func(t *testing.T) {
		for i := size - 1; i > 0; i-- {
			values[i] = uint64(i)
			fixHeap(minHeap, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i), values[heap[0]])
		}
	})

	t.Run("up_with_duplicates", func(t *testing.T) {
		for i := 0; i < size-1; i++ {
			values[i] = maxUint64
			fixHeap(minHeap, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(i+1), values[heap[0]])
		}

		for i := size - 1; i > 0; i-- {
			// Dividing by half results in 0, 0, 1, 1, &c.
			// So, the second sift runs into its doppleganger and exercises the
			// case where the parent is not less than itself.
			values[i] = uint64(size / 2)
			fixHeap(minHeap, size, values, heap, coHeap, i)
			assert.Equal(t, uint64(size/2), values[heap[0]])
		}
	})
}
