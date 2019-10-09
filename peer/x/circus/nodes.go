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

const size = 256

// node tracks the next and previous node in the ring of nodes.
// Since there are exactly 256 nodes and the allocation does not change at run
// time, a single byte is sufficient to track the index of every node in the
// circus.
type node struct {
	next uint8
	prev uint8
}

type nodes [size]node

// shift removes a node from its ring and places it at the end of another ring.
// The index of the node in the allocation does not change, just its own and
// its neighbors next and prev indices.
// the node may or may not be in the same ring as the ring that adopts it.
func (nodes *nodes) shift(i, j uint8) {
	inode := &nodes[i]
	iprev := &nodes[inode.prev]
	inext := &nodes[inode.next]
	jnode := &nodes[j]
	hnode := &nodes[jnode.prev]

	// Abort early if the node is already at the end of the target ring.
	// This check is not an optimization.
	// Failing to stop here affects correctness.
	if inode.next == j {
		return
	}

	// Remove from the current position in the current ring by connecting the
	// neighbors directly.
	iprev.next = inode.next
	inext.prev = inode.prev

	// Fix this node's links as well as the links of its new neighbors.
	// The new order of nodes will be: h <-> i <-> j.
	h := jnode.prev
	hnode.next = i
	inode.next = j
	inode.prev = h
	jnode.prev = i
}

// A ring is empty if the next node from the head node is itself.
func (nodes *nodes) empty(i uint8) bool {
	return nodes[i].next == i
}
