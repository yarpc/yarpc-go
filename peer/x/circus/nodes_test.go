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

package circus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodes(t *testing.T) {
	nodes := _zero

	// Verify the initial states of the circus rings.
	assert.True(t, nodes.empty(_no))
	assert.True(t, nodes.empty(_hi))
	assert.True(t, nodes.empty(_lo))
	assert.False(t, nodes.empty(_free))

	// Count and move the nodes in the freelist to the low load ring.
	{
		count := 0
		for !nodes.empty(_free) {
			nodes.shift(nodes[_free].next, _lo)
			count++
		}
		assert.Equal(t, size-4, count)
		assert.True(t, nodes.empty(_free))
	}

	// And then the high load ring.
	{
		count := 0
		for !nodes.empty(_lo) {
			nodes.shift(nodes[_lo].next, _hi)
			count++
		}
		assert.Equal(t, size-4, count)
	}

	// Then perform a complete rotation in place and verify that the states
	// match before and after.
	{
		order := nodes.gather(_hi)
		at := nodes[_hi].next
		for i := 0; i < size-4; i++ {
			nodes.shift(nodes[_hi].next, _hi)
		}
		assert.Equal(t, at, nodes[_hi].next)
		assert.Equal(t, order, nodes.gather(_hi))
	}
}

// TestShiftVersusSlice performs a sequence of shifts for nodes at each index
// of a ring.
// The results are verified by comparing the analogous mutation to the same
// sequence of indexes on a slice.
func TestShiftVersusSlice(t *testing.T) {
	nodes := _zero

	seq := jot(4, 256)
	assert.Equal(t, nodes.gather(_free), seq)

	for index := 0; index < 256-4; index++ {
		t.Run(fmt.Sprintf("%d", index), func(*testing.T) {
			at := nodes.index(_free, index+1)
			nodes.shift(at, _free)
			seq = cat(seq[:index], seq[index+1:], []uint8{at})
			assert.Equal(t, seq, nodes.gather(_free))
		})
	}
}

// gather constructs a slice consisting of all the node indexes about a given
// head node.
func (nodes *nodes) gather(head uint8) (indices []uint8) {
	index := head
	for {
		index = nodes[index].next
		if index == head {
			break
		}
		indices = append(indices, index)
	}
	return
}

// index searches forward to the index some number of links ahead of the given
// node.
func (nodes *nodes) index(head uint8, count int) uint8 {
	for index := 0; index < count; index++ {
		head = nodes[head].next
	}
	return head
}

// cat concatenates slices of bytes.
func cat(slices ...[]uint8) (cat []uint8) {
	for _, slice := range slices {
		cat = append(cat, slice...)
	}
	return
}

// jot produces a sequence of bytes.
func jot(start, stop int) (indices []uint8) {
	for ; start < stop; start++ {
		indices = append(indices, uint8(start))
	}
	return
}
