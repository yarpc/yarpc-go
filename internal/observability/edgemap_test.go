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

func TestEdgeMapZeroValue(t *testing.T) {
	var m edgeMap
	assert.Nil(t, m.load([]byte("a")), "load on an empty map")
	assert.Nil(t, m.get([]byte("a")), "get on an empty map")
	assert.Equal(t, 0, m.len())
}

func TestEdgeMapLoadOrCreate(t *testing.T) {
	var m edgeMap
	creates := 0
	create := func() *edge {
		creates++
		return &edge{}
	}

	first := m.loadOrCreate([]byte("a"), create)
	require.NotNil(t, first)
	assert.Equal(t, 1, creates)

	// A new edge is visible to get right away, but not to the lock-free load
	// until the dirty map has been promoted.
	assert.Same(t, first, m.get([]byte("a")))
	assert.Nil(t, m.load([]byte("a")), "not promoted yet")

	// Looking it up again hits the dirty map, which counts as a miss. With one
	// edge, one miss is enough to promote.
	assert.Same(t, first, m.loadOrCreate([]byte("a"), create))
	assert.Equal(t, 1, creates, "create must run once per key")
	assert.Same(t, first, m.load([]byte("a")), "promoted")

	// A second edge is not in the snapshot until enough misses accumulate.
	second := m.loadOrCreate([]byte("b"), create)
	require.NotSame(t, first, second)
	assert.Equal(t, 2, creates)
	assert.Equal(t, 2, m.len())
	assert.Nil(t, m.load([]byte("b")))
	assert.Same(t, second, m.loadOrCreate([]byte("b"), create)) // miss 1 of 2
	assert.Nil(t, m.load([]byte("b")))
	assert.Same(t, second, m.loadOrCreate([]byte("b"), create)) // miss 2 of 2
	assert.Same(t, second, m.load([]byte("b")), "promoted")
	assert.Same(t, first, m.load([]byte("a")), "old edges stay in the new snapshot")
}

func TestEdgeMapSnapshotIsImmutable(t *testing.T) {
	var m edgeMap
	a := m.loadOrCreate([]byte("a"), func() *edge { return &edge{} })
	m.loadOrCreate([]byte("a"), nil) // promote
	old := *m.read.Load()
	require.Len(t, old, 1)

	m.loadOrCreate([]byte("b"), func() *edge { return &edge{} })
	m.loadOrCreate([]byte("b"), nil)
	m.loadOrCreate([]byte("b"), nil) // promote again

	assert.Len(t, old, 1, "a published snapshot must never change")
	assert.Same(t, a, old["a"])
	assert.Len(t, *m.read.Load(), 2)
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
					e := m.load(key)
					if e == nil {
						e = m.loadOrCreate(key, func() *edge {
							creates.Add(1)
							return &edge{}
						})
					}
					if !seen[k].CompareAndSwap(nil, e) {
						assert.Same(t, seen[k].Load(), e, "every caller must see the same edge for %d", k)
					}
				}
			}
		}(g)
	}
	wg.Wait()

	assert.EqualValues(t, keys, creates.Load(), "each key is created exactly once")
	assert.Equal(t, keys, m.len())
	for k := 0; k < keys; k++ {
		assert.Same(t, seen[k].Load(), m.get([]byte(fmt.Sprintf("key-%d", k))))
	}
}

func TestEdgeMapLoadDoesNotAllocate(t *testing.T) {
	var m edgeMap
	key := []byte("some-edge-key\x00with\x00several\x00parts")
	m.loadOrCreate(key, func() *edge { return &edge{} })
	m.loadOrCreate(key, nil) // promote
	require.NotNil(t, m.load(key))

	allocs := testing.AllocsPerRun(100, func() {
		if m.load(key) == nil {
			t.Fatal("edge went missing")
		}
	})
	assert.Zero(t, allocs, "the hit path must not allocate")
}

// BenchmarkEdgeMapLoad measures the cost of finding a hot edge from many
// cores, which is what every call does.
func BenchmarkEdgeMapLoad(b *testing.B) {
	var m edgeMap
	key := []byte("caller\x00service\x00tchannel\x00thrift\x00Svc::method\x00rk\x00rd\x00inbound\x00unary")
	e := &edge{}
	m.loadOrCreate(key, func() *edge { return e })
	m.loadOrCreate(key, nil) // promote

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if m.load(key) != e {
				b.Fatal("edge went missing")
			}
		}
	})
}

func TestEdgeMapMax(t *testing.T) {
	m := edgeMap{max: 2}
	create := func() *edge { return &edge{} }

	a := m.loadOrCreate([]byte("a"), create)
	b := m.loadOrCreate([]byte("b"), create)
	require.NotNil(t, a)
	require.NotNil(t, b)
	assert.False(t, m.isFull(), "at the limit, but nothing has been refused yet")

	assert.Nil(t, m.loadOrCreate([]byte("c"), func() *edge {
		t.Fatal("create must not run at the limit")
		return nil
	}))
	assert.True(t, m.isFull())
	assert.Equal(t, 2, m.len())

	// Once full, the snapshot is complete, so loads are conclusive.
	assert.Same(t, a, m.load([]byte("a")))
	assert.Same(t, b, m.load([]byte("b")))
	assert.Nil(t, m.load([]byte("c")))

	// Existing edges keep being found, and the refusal is repeatable.
	assert.Same(t, a, m.loadOrCreate([]byte("a"), create))
	assert.Nil(t, m.loadOrCreate([]byte("d"), create))
	assert.Equal(t, 2, m.len())
}

func TestEdgeMapNoMaxIsUnlimited(t *testing.T) {
	var m edgeMap
	for i := 0; i < 1000; i++ {
		require.NotNil(t, m.loadOrCreate([]byte(fmt.Sprint(i)), func() *edge { return &edge{} }))
	}
	assert.False(t, m.isFull())
	assert.Equal(t, 1000, m.len())
}

func TestEdgeMapMaxConcurrent(t *testing.T) {
	const (
		max        = 50
		keys       = 200
		goroutines = 16
	)

	m := edgeMap{max: max}
	var (
		creates atomic.Int64
		wg      sync.WaitGroup
	)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < keys; i++ {
				key := []byte(fmt.Sprintf("key-%d", (i+g*13)%keys))
				if m.load(key) != nil {
					continue
				}
				m.loadOrCreate(key, func() *edge {
					creates.Add(1)
					return &edge{}
				})
			}
		}(g)
	}
	wg.Wait()

	assert.EqualValues(t, max, creates.Load(), "exactly the limit is created, however many callers race")
	assert.Equal(t, max, m.len())
	assert.True(t, m.isFull())

	// The snapshot is complete once full: every edge is loadable lock-free.
	found := 0
	for i := 0; i < keys; i++ {
		if m.load([]byte(fmt.Sprintf("key-%d", i))) != nil {
			found++
		}
	}
	assert.Equal(t, max, found)
}
