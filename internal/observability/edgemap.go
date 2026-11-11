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

// edgeMap stores edges by key. Edges are never removed or replaced.
//
// Lookups first check an immutable snapshot without locking or writing to
// shared memory. This avoids the shared counter updates of an RWMutex read
// lock on every RPC.
//
// A locked map holds all edges, including new ones not yet in the snapshot.
// After enough lookups find an edge only in that map, it is copied into a
// new snapshot. Published snapshots are never modified.
//
// The zero value is ready to use. It must not be copied after first use.
type edgeMap struct {
	// read contains a subset of dirty. Only the map is immutable, not the edges.
	read atomic.Pointer[map[string]*edge]

	// mu protects dirty and misses.
	mu    sync.Mutex
	dirty map[string]*edge
	// misses counts dirty hits since the last snapshot, excluding snapshot hits.
	misses int
}

// getOrCreate returns the edge for key, creating it if absent. create runs
// once per key, with the map locked. It must return a non-nil edge and must
// not call back into this map. The caller may reuse key after this returns.
func (m *edgeMap) getOrCreate(key []byte, create func() *edge) *edge {
	if e := m.loadSnapshot(key); e != nil {
		return e
	}
	return m.getOrCreateSlow(key, create)
}

// loadSnapshot returns nil if key is not yet in the snapshot.
func (m *edgeMap) loadSnapshot(key []byte) *edge {
	snap := m.read.Load()
	if snap == nil {
		return nil
	}
	return (*snap)[string(key)] // A map lookup does not copy the key bytes.
}

func (m *edgeMap) getOrCreateSlow(key []byte, create func() *edge) *edge {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Another goroutine may have published a snapshot while we waited.
	if e := m.loadSnapshot(key); e != nil {
		return e
	}

	if e, ok := m.dirty[string(key)]; ok {
		m.misses++
		// Copying n entries costs O(n). Waiting for n misses spreads that
		// cost over those lookups: O(1) extra work per miss on average.
		if m.misses >= len(m.dirty) {
			m.promoteLocked()
		}
		return e
	}

	e := create()
	if m.dirty == nil {
		m.dirty = make(map[string]*edge, _defaultGraphSize)
	}
	m.dirty[string(key)] = e // Copy the key; the caller may reuse its buffer.
	return e
}

// promoteLocked copies all edges into a new snapshot. The caller must hold mu.
func (m *edgeMap) promoteLocked() {
	snap := make(map[string]*edge, len(m.dirty))
	for k, e := range m.dirty {
		snap[k] = e
	}
	m.read.Store(&snap)
	m.misses = 0
}
