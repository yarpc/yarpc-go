// Copyright (c) 2026 Uber Technologies, Inc.
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

package observability

import (
	"sync"
	"sync/atomic"
)

// edgeMap maps an edge key to its edge. Edges are only ever added, never
// removed, and every call looks one up, so the map is tuned for reads.
//
// A read takes no lock and performs no write to shared memory: it loads an
// immutable snapshot and looks the key up in it. This matters because every
// inbound and outbound call goes through it, and a sync.RWMutex read lock is
// an atomic write to one cache line shared by all cores, which stops
// scaling on hot edges.
//
// New edges are added to a mutex-protected dirty map that always holds every
// edge. Lookups that find an edge only in the dirty map count as misses, and
// once there have been as many misses as there are edges the dirty map is
// copied into a fresh snapshot. This is the same amortisation sync.Map uses.
// Copying is O(n), so insertion stays O(1) amortised.
//
// The map can optionally be capped. Once it holds max edges it refuses to
// create more, and from then on its snapshot is always complete, so a failed
// load is conclusive and callers can stop taking the lock altogether.
//
// The zero value is ready to use, and unlimited. It must not be copied after
// first use.
type edgeMap struct {
	// max is the most edges the map will hold, or 0 for no limit. It must be
	// set before the map is used.
	max int

	// read is an immutable snapshot of a subset of dirty. The map it points to
	// is never modified after being published.
	read atomic.Pointer[map[string]*edge]

	mu sync.Mutex
	// dirty holds every edge, including ones not yet in read.
	dirty map[string]*edge
	// misses counts lookups that missed read but hit dirty since read was last
	// replaced.
	misses int

	// full is set, under mu, once the map has max edges and has been asked for
	// one more. From then on read holds every edge and dirty never changes.
	full atomic.Bool
}

// load returns the edge for key from the snapshot, or nil. It never blocks and
// does not allocate. A nil result does not mean that the edge does not exist,
// only that it is not in the snapshot yet; use loadOrCreate to find out.
func (m *edgeMap) load(key []byte) *edge {
	snap := m.read.Load()
	if snap == nil {
		return nil
	}
	return (*snap)[string(key)] // the compiler elides this string conversion
}

// get returns the edge for key, or nil if there is none. Unlike load, it also
// sees edges that are not in the snapshot yet.
func (m *edgeMap) get(key []byte) *edge {
	if e := m.load(key); e != nil {
		return e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dirty[string(key)]
}

// isFull reports whether the map has reached its limit and refused an edge.
// Once it does, a nil result from load is conclusive: there is no such edge,
// and there never will be.
func (m *edgeMap) isFull() bool {
	return m.full.Load()
}

// loadOrCreate returns the edge for key, calling create to make it if there is
// none. create runs with the map locked, so it must not call back into the
// map, and only runs once per key. It returns nil if the edge does not exist
// and the map is at its limit.
func (m *edgeMap) loadOrCreate(key []byte, create func() *edge) *edge {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.dirty[string(key)]; ok {
		// Found, but not in the snapshot or we would not be here.
		m.misses++
		if m.misses >= len(m.dirty) {
			m.promoteLocked()
		}
		return e
	}

	if m.max > 0 && len(m.dirty) >= m.max {
		// Publish everything, so that readers can tell that edges they
		// cannot load do not exist.
		m.promoteLocked()
		m.full.Store(true)
		return nil
	}

	e := create()
	if m.dirty == nil {
		m.dirty = make(map[string]*edge, _defaultGraphSize)
	}
	m.dirty[string(key)] = e
	return e
}

// len returns the number of edges.
func (m *edgeMap) len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.dirty)
}

// promoteLocked publishes a new snapshot containing every edge.
func (m *edgeMap) promoteLocked() {
	m.misses = 0
	if snap := m.read.Load(); snap != nil && len(*snap) == len(m.dirty) {
		return
	}
	snap := make(map[string]*edge, len(m.dirty))
	for k, e := range m.dirty {
		snap[k] = e
	}
	m.read.Store(&snap)
}
