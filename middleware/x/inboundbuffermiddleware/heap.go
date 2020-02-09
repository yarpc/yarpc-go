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
	"fmt"
	"os"
)

type heapDir bool

const (
	minHeap heapDir = true
	maxHeap         = false
)

func fixHeap(dir heapDir, length int, values []uint64, heap, coHeap []int, index int) {
	fmt.Fprintf(os.Stderr, "fix heap index %d i %d\n", index, coHeap[index])
	fixHeapTowardRoot(dir, values, heap, coHeap, index)
	fixHeapAwayFromRoot(dir, length, values, heap, coHeap, index)
}

func fixHeapTowardRoot(dir heapDir, values []uint64, heap, coHeap []int, index int) {
	value := values[index]
	i := coHeap[index]
	for i > 0 {
		pi := (i+1)/2 - 1
		parent := values[heap[pi]]
		if parent > value == dir {
			fmt.Fprintf(os.Stderr, "fix heap toward root swap %d %d index %d\n", i, pi, index)
			swap(heap, coHeap, i, pi)
			i = pi
		} else {
			break
		}
	}
}

func fixHeapAwayFromRoot(dir heapDir, length int, values []uint64, heap, coHeap []int, index int) {
	value := values[index]
	i := coHeap[index]
	for {
		// i = index, ri = right index, li = left index, si = swap index.
		// Variant: i increases, following either the left or right child
		// index, until there are no more children.
		ri := (i + 1) * 2
		li := ri - 1
		si := -1 // sentinel indicates no lesser child found.

		var left uint64
		// Consider the left child, if one exists.
		if li < length {
			left = values[heap[li]]
			if left < value == dir {
				si = li
			}
		}

		// Independently, consider the right child, if one exists.
		if ri < length {
			right := values[heap[ri]]
			// The right child will override a planned swap with the left child
			// if the right child is less than the left.
			if si >= 0 {
				if right < left == dir {
					si = ri
				}
			} else {
				if right < value == dir {
					si = ri
				}
			}
		}

		// Swap with either the left or right child if either were found to be
		// less than their parent.
		if si >= 0 {
			fmt.Fprintf(os.Stderr, "swap child %d %d\n", i, si)
			swap(heap, coHeap, i, si)
			i = si
		} else {
			// Otherwise, we are guaranteed that either there are no children
			// or no descendants less than the current node.
			break
		}
	}
}
