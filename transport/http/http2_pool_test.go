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

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// newTestH2Pool starts a cleartext HTTP/2 test server and returns its
// host:port address plus a pool factory that builds *http2.Transport
// instances wired the same way buildH2Transport wires the real one.
func newTestH2Pool(t *testing.T, handler http.HandlerFunc) (addr string, newTransport func() *http2.Transport) {
	t.Helper()

	h2s := &http2.Server{
		IdleTimeout: defaultIdleConnTimeout,
	}
	h1s := httptest.NewUnstartedServer(handler)
	h1s.Config.Protocols = new(http.Protocols)
	h1s.Config.Protocols.SetHTTP1(true)
	h1s.Config.Protocols.SetUnencryptedHTTP2(true)
	http2.ConfigureServer(h1s.Config, h2s)
	h1s.Start()
	t.Cleanup(h1s.Close)

	u, err := url.Parse(h1s.URL)
	require.NoError(t, err)

	transportOpts := newTransportOptions()
	return u.Host, func() *http2.Transport { return buildH2Transport(&transportOpts) }
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s", timeout)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestHTTP2PoolPickConnGrowsWhenEmpty(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newHTTP2Pool(addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	defer pool.Close()

	conn, err := pool.pickConn()
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.Len(t, *pool.connsPtr.Load(), 1)
}

func TestHTTP2PoolPickConnPicksLeastLoaded(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newHTTP2Pool(addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	defer pool.Close()

	busy := pool.newConn()
	idle := pool.newConn()
	pool.addConn(busy)
	pool.addConn(idle)

	// Manually simulate an in-flight request on busy: incInflight/decInflight
	// is the pool's own accounting, not something a real request drives in
	// this test, since http2Conn no longer reads live state off the wire.
	busy.incInflight()

	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.Same(t, idle, picked, "pool should prefer the less-loaded connection")
	defer picked.decInflight() // pickConn reserves the slot via incInflight

	busy.decInflight()
}

func TestHTTP2PoolMaybeScaleUpSingleFlight(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConns = 5
	cfg.scaleUpThreshold = 0.5
	cfg.maxConcurrentStreams = 2
	pool := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c := pool.newConn()
	pool.addConn(c)

	// Occupy c with in-flight requests so streamsActive crosses the
	// scale-up threshold (2 active out of maxConcurrentStreams=2, with a
	// 0.5 threshold).
	c.incInflight()
	c.incInflight()

	// Gate all 20 calls behind a shared start channel so they race against
	// each other as tightly as possible: this is what would otherwise
	// surface the bug maybeScaleUp guards against, now that growing the
	// pool is instant (no dial) -- without the fresh re-check, each of the
	// 20 goroutines observes the same stale, still-over-threshold c and
	// piles on its own scale-up as soon as it gets a turn, growing the pool
	// to maxConns instead of by one.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			pool.maybeScaleUp(c)
		}()
	}
	close(start)
	wg.Wait()

	waitForCondition(t, time.Second, func() bool { return len(*pool.connsPtr.Load()) == 2 })
	assert.Len(t, *pool.connsPtr.Load(), 2, "concurrent scale-up attempts should add exactly one connection")
}

func TestHTTP2PoolScaleDownParksLastConn(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	pool := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c1 := pool.newConn()
	c2 := pool.newConn()
	pool.addConn(c1)
	pool.addConn(c2)

	pool.maybeScaleDown()
	assert.False(t, c1.parked())
	assert.True(t, c2.parked(), "the last connection should be parked when load is zero")
	assert.Len(t, *pool.connsPtr.Load(), 2, "parking must not remove the connection from the pool")

	// A parked connection is never picked.
	for i := 0; i < 5; i++ {
		picked, err := pool.pickConn()
		require.NoError(t, err)
		assert.Same(t, c1, picked)
	}

	// Already at minConns active: nothing more to park.
	pool.maybeScaleDown()
	assert.False(t, c1.parked())
}

func TestHTTP2PoolScaleDownAvoidsFlapping(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	cfg.maxConcurrentStreams = 100
	cfg.scaleUpThreshold = 0.8
	cfg.scaleDownGap = 0.1
	pool := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c1 := pool.newConn()
	c2 := pool.newConn()
	pool.addConn(c1)
	pool.addConn(c2)

	// 75 total requests would leave the survivor at 75/100: below the 0.8
	// scale-up trigger, but above the 0.7 scale-down threshold. Parking here
	// would put the pool right at the edge of scaling back up.
	for i := 0; i < 40; i++ {
		c1.incInflight()
	}
	for i := 0; i < 35; i++ {
		c2.incInflight()
	}
	pool.maybeScaleDown()
	assert.False(t, c1.parked() || c2.parked(), "must not scale down when the survivor would sit near the scale-up threshold")

	// 60 total leaves the survivor comfortably under 0.7.
	for i := 0; i < 15; i++ {
		c1.decInflight()
	}
	pool.maybeScaleDown()
	assert.True(t, c2.parked())
	assert.False(t, c1.parked())

	// And the survivor, at 60/100, does not trigger an immediate scale-up.
	pool.maybeScaleUp(c1)
	assert.True(t, c2.parked(), "scale-up must not fire right after a scale-down")
	assert.Len(t, *pool.connsPtr.Load(), 2)
}

func TestHTTP2PoolScaleUpUnparksBeforeDialing(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConns = 2
	cfg.maxConcurrentStreams = 2
	cfg.scaleUpThreshold = 0.5
	pool := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c1 := pool.newConn()
	c2 := pool.newConn()
	pool.addConn(c1)
	pool.addConn(c2)
	require.True(t, c2.park())

	// Pool is at maxConns, but a parked connection is available.
	c1.incInflight()
	pool.maybeScaleUp(c1)
	assert.False(t, c2.parked(), "scale-up should re-activate the parked connection")
	assert.Len(t, *pool.connsPtr.Load(), 2, "no new connection should be built")

	// pickConn's grow path re-activates too, when every active conn is full.
	c1.incInflight() // c1 at capacity
	require.True(t, c2.park())
	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.Same(t, c2, picked)
	assert.False(t, c2.parked())
}

func TestHTTP2ConnParkUnparkOnce(t *testing.T) {
	c := newHTTP2Conn(nil)
	assert.False(t, c.unpark(), "an active connection cannot be unparked")
	assert.True(t, c.park())
	assert.False(t, c.park(), "park only transitions once")
	assert.False(t, c.usable())
	assert.True(t, c.unpark())
	assert.False(t, c.unpark(), "unpark only transitions once")
	assert.True(t, c.usable())
}

func TestHTTP2PoolCleanupConnsClosesLongParkedIdleConn(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.idleTimeout = time.Millisecond
	pool := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c := pool.newConn()
	pool.addConn(c)
	// Simulate a connection that served a request and went idle.
	c.incInflight()
	c.decInflight()
	require.True(t, c.park())

	waitForCondition(t, time.Second, func() bool {
		return time.Since(c.idleSince()) > cfg.idleTimeout
	})

	pool.cleanupConns()
	assert.Len(t, *pool.connsPtr.Load(), 1, "a parked connection stays in the pool so it can be re-activated")
	assert.True(t, c.parked())
}

func TestHTTP2PoolAddRemoveConnRace(t *testing.T) {
	addr, newTransport := newTestH2Pool(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newHTTP2Pool(addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	defer pool.Close()

	const n = 16
	conns := make([]*http2Conn, n)
	for i := range conns {
		conns[i] = newHTTP2Conn(nil) // not usable, but fine for CAS bookkeeping
	}

	var wg sync.WaitGroup
	for _, c := range conns {
		wg.Add(1)
		go func(c *http2Conn) {
			defer wg.Done()
			pool.addConn(c)
		}(c)
	}
	wg.Wait()
	assert.Len(t, *pool.connsPtr.Load(), n)

	for _, c := range conns {
		wg.Add(1)
		go func(c *http2Conn) {
			defer wg.Done()
			pool.removeConn(c)
		}(c)
	}
	wg.Wait()
	assert.Empty(t, *pool.connsPtr.Load())
}
