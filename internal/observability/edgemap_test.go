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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lookup inspects existing edges in tests without creating or promoting them.
func (m *edgeMap) lookup(key []byte) *edge {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dirty[string(key)]
}

func TestEdgeMapGetOrCreate(t *testing.T) {
	var m edgeMap
	creates := 0
	create := func() *edge {
		creates++
		return &edge{}
	}

	first := m.getOrCreate([]byte("a"), create)
	require.NotNil(t, first)
	assert.Equal(t, 1, creates)

	assert.Same(t, first, m.lookup([]byte("a")))
	assert.Nil(t, m.loadSnapshot([]byte("a")), "not promoted yet")

	// Looking it up again hits the dirty map, which counts as a miss. With one
	// edge, one miss is enough to promote.
	assert.Same(t, first, m.getOrCreate([]byte("a"), create))
	assert.Equal(t, 1, creates, "create must run once per key")
	assert.Same(t, first, m.loadSnapshot([]byte("a")), "promoted")

	// A second edge is not in the snapshot until enough misses accumulate.
	second := m.getOrCreate([]byte("b"), create)
	require.NotSame(t, first, second)
	assert.Equal(t, 2, creates)
	assert.Len(t, m.dirty, 2)
	assert.Nil(t, m.loadSnapshot([]byte("b")))
	assert.Same(t, second, m.getOrCreate([]byte("b"), create)) // miss 1 of 2

	// Snapshot hits must not trigger an early copy of the map.
	for i := 0; i < 10; i++ {
		assert.Same(t, first, m.getOrCreate([]byte("a"), create))
	}
	assert.Equal(t, 1, m.misses)
	assert.Nil(t, m.loadSnapshot([]byte("b")))
	assert.Same(t, second, m.getOrCreate([]byte("b"), create)) // miss 2 of 2
	assert.Same(t, second, m.loadSnapshot([]byte("b")), "promoted")
	assert.Same(t, first, m.loadSnapshot([]byte("a")), "old edges stay in the new snapshot")
	assert.Equal(t, 2, creates)
}

func TestEdgeMapSnapshotIsImmutable(t *testing.T) {
	var m edgeMap
	a := m.getOrCreate([]byte("a"), func() *edge { return &edge{} })
	m.getOrCreate([]byte("a"), nil) // promote
	old := *m.read.Load()
	require.Len(t, old, 1)

	m.getOrCreate([]byte("b"), func() *edge { return &edge{} })
	m.getOrCreate([]byte("b"), nil)
	m.getOrCreate([]byte("b"), nil) // promote again

	assert.Len(t, old, 1, "a published snapshot must never change")
	assert.Same(t, a, old["a"])
	assert.Len(t, *m.read.Load(), 2)
}

func TestEdgeMapDelayedMiss(t *testing.T) {
	var m edgeMap
	a := m.getOrCreate([]byte("a"), func() *edge { return &edge{} })
	require.Nil(t, m.loadSnapshot([]byte("a")))
	m.getOrCreate([]byte("a"), nil) // Another caller promotes before we lock.
	m.getOrCreate([]byte("b"), func() *edge { return &edge{} })
	m.getOrCreate([]byte("b"), nil) // miss 1 of 2

	// Resume the original lookup on the slow path after the snapshot changed.
	assert.Same(t, a, m.getOrCreateSlow([]byte("a"), nil))
	assert.Equal(t, 1, m.misses, "a delayed snapshot hit must not count as a miss")
	assert.Nil(t, m.loadSnapshot([]byte("b")), "must not promote early")
}

func TestEdgeMapCopiesKey(t *testing.T) {
	var m edgeMap
	key := []byte("a")
	a := m.getOrCreate(key, func() *edge { return &edge{} })
	key[0] = 'b'
	b := m.getOrCreate(key, func() *edge { return &edge{} })
	require.NotSame(t, a, b)

	for i := 0; i < 3; i++ {
		assert.Same(t, a, m.getOrCreate([]byte("a"), nil))
		assert.Same(t, b, m.getOrCreate([]byte("b"), nil))
	}
}

func TestEdgeMapConcurrent(t *testing.T) {
	const (
		goroutines = 32
		keys       = 200
		rounds     = 50
	)

	var (
		m       edgeMap
		creates atomic.Int64
		wg      sync.WaitGroup
		seen    [keys]atomic.Pointer[edge]
	)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				for i := 0; i < keys; i++ {
					k := (i + g) % keys
					key := []byte(fmt.Sprintf("key-%d", k))
					e := m.getOrCreate(key, func() *edge {
						creates.Add(1)
						return &edge{}
					})
					if !seen[k].CompareAndSwap(nil, e) {
						assert.Same(t, seen[k].Load(), e, "every caller must see the same edge for %d", k)
					}
				}
			}
		}(g)
	}
	wg.Wait()

	assert.EqualValues(t, keys, creates.Load(), "each key is created exactly once")
	assert.Len(t, m.dirty, keys)
	for k := 0; k < keys; k++ {
		assert.Same(t, seen[k].Load(), m.getOrCreate([]byte(fmt.Sprintf("key-%d", k)), nil))
	}
}

func TestEdgeMapGetOrCreateDoesNotAllocate(t *testing.T) {
	var m edgeMap
	key := []byte("some-edge-key\x00with\x00several\x00parts")
	m.getOrCreate(key, func() *edge { return &edge{} })
	m.getOrCreate(key, nil) // promote
	require.NotNil(t, m.loadSnapshot(key))

	allocs := testing.AllocsPerRun(100, func() {
		if m.getOrCreate(key, nil) == nil {
			t.Fatal("edge went missing")
		}
	})
	assert.Zero(t, allocs, "the hit path must not allocate")
}

// BenchmarkEdgeMapGetOrCreate measures the cost of finding a hot edge from many
// cores, which is what every call does.
func BenchmarkEdgeMapGetOrCreate(b *testing.B) {
	var m edgeMap
	key := []byte("caller\x00service\x00tchannel\x00thrift\x00Svc::method\x00rk\x00rd\x00inbound\x00unary")
	e := &edge{}
	m.getOrCreate(key, func() *edge { return e })
	m.getOrCreate(key, nil) // promote

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if m.getOrCreate(key, nil) != e {
				b.Fatal("edge went missing")
			}
		}
	})
}
