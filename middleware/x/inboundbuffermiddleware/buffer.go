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

// buffer is the internal data structure of the QOS buffer.
// The buffer assigns entity indexes for any accepted request, which the outer
// buffer uses to track RPC specific details like context, request, response
// writer, and the error return channel.
// The inner buffer indexes all received requests by deadline and priority and
// implements eviction policies.
//
// Instead of using reflection, the internal representation of a heap
// is a pair of arrays that track all of the entitiy indexes, the graph and
// co-graph.
// For example, the minDeadlines and coMinDeadlines arrays track every entity
// by its corresponding deadline (the deadlines array).
//
// The minDeadlines array is a heap of entity indexes with the
// index of the nearest approaching deadline on top.
// The coMinDeadliens array is the inverse: it tracks the index in the
// minDeadlines array for every entity.
//
// This approach allows us to use the same integer heap implementation three
// times without specialized types and without reflection.
//
// At any moment, only the values left of the `length` in any of these arrays
// corresponds to a scheduled entity (a request).
// The free list contains the index of every possible entity in the bounded
// queue.
// Those to the left of `length` are scheduled and those to the right are free.
type buffer struct {
	capacity int
	length   int

	// All of the following are parallel.
	// Each index corresponds to the same entity.
	deadlines       []uint64 // maxint indicates an unallocated entity
	priorities      []uint64 // maxint indicates an unallocated entity
	coFree          []int    // free[coFree[i]] == i
	coMinDeadlines  []int    // minDeadlines[coMinDeadlines[i]] == i
	coMinPriorities []int    // minPriorities[coMinPriorities[i]] == i
	coMaxPriorities []int    // maxPriorities[coMaxPriorities[i]] == i

	// Indexes
	free          []int // list of unused entities
	minDeadlines  []int // min heap by deadline
	minPriorities []int // min heap by priority
	maxPriorities []int // max heap by priority
}

// Init initializes the buffer.
// The outer buffer allocates the inner buffer inline,
// so this serves as a initailizer and no constructor is useful.
func (b *buffer) Init(capacity int) {
	b.capacity = capacity
	b.deadlines = makeMaxSliceUint64(capacity)
	b.priorities = makeMaxSliceUint64(capacity)
	b.free = jot(capacity)
	b.coFree = jot(capacity)
	b.minDeadlines = jot(capacity)
	b.coMinDeadlines = jot(capacity)
	b.minPriorities = jot(capacity)
	b.coMinPriorities = jot(capacity)
	b.maxPriorities = jot(capacity)
	b.coMaxPriorities = jot(capacity)
}

// Full indicates that no further entities can be added to the buffer until one
// is removed.
func (b *buffer) Full() bool {
	return b.length >= b.capacity
}

// Adds an entity with the given deadline and priority (lower priorities have
// precedence), returns the allocated entity index, or -1 if the buffer is
// full.
func (b *buffer) Put(deadline uint64, priority uint64) int {
	if b.length >= b.capacity {
		return -1
	}

	// Choose an entity index directly on the partition of the free entity
	// index.
	i := b.free[b.length]
	b.deadlines[i] = deadline
	b.priorities[i] = priority

	// Move the entity to the partition on all corresponding indexes.
	swap(b.maxPriorities, b.coMaxPriorities, b.coMaxPriorities[i], b.length)
	swap(b.minPriorities, b.coMinPriorities, b.coMinPriorities[i], b.length)
	swap(b.minDeadlines, b.coMinDeadlines, b.coMinDeadlines[i], b.length)
	// Shift the partition, introducing the entity to every index.
	b.length++

	// Adjust indexes:

	// Assuming all deadlines are positive in the Unix epoch.
	// Time travellers, please do not use this algorithm before 1970.
	fixHeapTowardRoot(minHeap, b.deadlines, b.minDeadlines, b.coMinDeadlines, i)
	fixHeapTowardRoot(minHeap, b.priorities, b.minPriorities, b.coMinPriorities, i)
	fixHeapTowardRoot(maxHeap, b.priorities, b.maxPriorities, b.coMaxPriorities, i)

	return i
}

// Pop removes and returns the index for the highest priority entity in the
// buffer.
func (b *buffer) Pop() int {
	if b.length == 0 {
		return -1
	}

	// Index of highest priority entity.
	i := b.maxPriorities[0]

	b.evict(i)

	return i
}

// Evict removes and returns the index of the entity with the earliest expired
// deadline, or -1 if no entity has expired.
func (b *buffer) EvictExpired(now uint64) int {
	if b.length == 0 {
		return -1
	}

	// Index of next entity to expire.
	i := b.minDeadlines[0]

	if b.deadlines[i] > now {
		return -1
	}

	b.evict(i)

	return i
}

// EvictLowerPriority removes and returns the index of the entity in the buffer
// that has a lower priority (lower priorities are numerically higher).
func (b *buffer) EvictLowerPriority(priority uint64) int {
	if b.length == 0 {
		return -1
	}

	// Index of of lowest priority entity.
	i := b.minPriorities[0]

	// Favor keeping an entity over replacing with an equivalent, for the sake
	// of churn.
	if b.priorities[i] >= priority {
		return -1
	}

	b.evict(i)

	return i
}

// evict removes an entity from the buffer, frees its index for a future
// entity, and adjusts the internal heaps.
func (b *buffer) evict(i int) {
	// Reset values
	b.deadlines[i] = maxUint64
	b.priorities[i] = maxUint64

	// One less entity.
	// Move partition first, so we can use the new length as the destination
	// index for swaps.
	b.length--

	// Move the selected entity out beyond the horizon.
	swap(b.free, b.coFree, b.coFree[i], b.length)

	// Similarly, swap entities in each index, then fix the heaps.
	heapEvict(minHeap, b.length, b.deadlines, b.minDeadlines, b.coMinDeadlines, i)
	heapEvict(minHeap, b.length, b.priorities, b.minPriorities, b.coMinPriorities, i)
	heapEvict(maxHeap, b.length, b.priorities, b.maxPriorities, b.coMaxPriorities, i)
}

func heapEvict(dir heapDir, length int, values []uint64, heap, coHeap []int, i int) {
	j := coHeap[i]
	if j != length {
		k := heap[length]
		swap(heap, coHeap, j, length)
		fixHeap(dir, length, values, heap, coHeap, k)
	}
}
