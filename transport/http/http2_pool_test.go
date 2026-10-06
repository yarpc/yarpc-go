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
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// newTestH2TransportServer starts a cleartext HTTP/2 test server and returns its
// host:port address plus a pool factory that builds *http2.Transport
// instances wired the same way buildH2Transport wires the real one.
func newTestH2TransportServer(t *testing.T, handler http.HandlerFunc) (addr string, newTransport func() *http2.Transport) {
	t.Helper()

	h2s := &http2.Server{
		IdleTimeout: defaultIdleConnTimeout,
	}
	h1s := httptest.NewUnstartedServer(h2c.NewHandler(handler, h2s))
	h1s.Start()
	t.Cleanup(h1s.Close)

	u, err := url.Parse(h1s.URL)
	require.NoError(t, err)

	transportOpts := newTransportOptions()
	return u.Host, func() *http2.Transport { return buildH2Transport(&transportOpts) }
}

// newEmptyH2Pool builds a pool and discards the connections it pre-creates, so
// tests can add exactly the connections they want. The discarded slots never
// dialed, so there is nothing to close.
func newEmptyH2Pool(t *testing.T, addr string, newTransport func() *http2.Transport, cfg http2PoolConfig, logger *zap.Logger) *http2Pool {
	t.Helper()
	p, err := newHTTP2Pool(addr, newTransport, cfg, logger)
	require.NoError(t, err)
	p.connsPtr.Store(&[]*http2Conn{})
	return p
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newEmptyH2Pool(t, addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	defer pool.Close()

	conn, err := pool.pickConn()
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.Len(t, *pool.connsPtr.Load(), 1)
}

func TestHTTP2PoolPickConnPicksLeastLoaded(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newEmptyH2Pool(t, addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConns = 5
	cfg.scaleUpThreshold = 0.5
	cfg.maxConcurrentStreams = 2
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	cfg.maxConcurrentStreams = 100
	cfg.scaleUpThreshold = 0.8
	cfg.scaleDownGap = 0.1
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConns = 2
	cfg.maxConcurrentStreams = 2
	cfg.scaleUpThreshold = 0.5
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
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
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.idleTimeout = time.Millisecond
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
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

func TestHTTP2PoolConfigValidation(t *testing.T) {
	tests := []struct {
		desc    string
		mutate  func(*http2PoolConfig)
		wantErr string
	}{
		{"default is valid", func(*http2PoolConfig) {}, ""},
		{"negative minConns", func(c *http2PoolConfig) { c.minConns = -1 }, "minConns must be non-negative"},
		{"maxConns below minConns", func(c *http2PoolConfig) { c.minConns = 3; c.maxConns = 2 }, "must be >= minConns"},
		{"zero maxConns", func(c *http2PoolConfig) { c.minConns = 0; c.maxConns = 0 }, "maxConns must be at least 1"},
		{"zero maxConcurrentStreams", func(c *http2PoolConfig) { c.maxConcurrentStreams = 0 }, "maxConcurrentStreams must be at least 1"},
		{"zero scaleUpThreshold", func(c *http2PoolConfig) { c.scaleUpThreshold = 0 }, "scaleUpThreshold must be in (0, 1]"},
		{"scaleUpThreshold above 1", func(c *http2PoolConfig) { c.scaleUpThreshold = 1.1 }, "scaleUpThreshold must be in (0, 1]"},
		{"gap eliminates scale-down threshold", func(c *http2PoolConfig) { c.scaleUpThreshold = 0.5; c.scaleDownGap = 0.5 }, "minus scaleDownGap"},
		{"negative idleTimeout", func(c *http2PoolConfig) { c.idleTimeout = -time.Second }, "idleTimeout must be non-negative"},
		{"zero scalingMonitorInterval", func(c *http2PoolConfig) { c.scalingMonitorInterval = 0 }, "scalingMonitorInterval must be positive"},
		{"negative scalingMonitorInterval", func(c *http2PoolConfig) { c.scalingMonitorInterval = -time.Second }, "scalingMonitorInterval must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cfg := defaultHTTP2PoolConfig()
			tt.mutate(&cfg)
			pool, err := newHTTP2Pool("127.0.0.1:0", func() *http2.Transport { return &http2.Transport{} }, cfg, zap.NewNop())
			if tt.wantErr == "" {
				require.NoError(t, err)
				pool.Close()
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Nil(t, pool, "a rejected config must not build a pool (or start its monitor goroutine)")
		})
	}
}

func TestHTTP2PoolPrecreatesMinConns(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 3
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()
	assert.Len(t, *pool.connsPtr.Load(), 3)
}

func TestHTTP2PoolDynamicScalingDisabled(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.dynamicScalingEnabled = false
	cfg.minConns = 3
	cfg.maxConcurrentStreams = 2
	cfg.scaleUpThreshold = 0.5
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	conns := *pool.connsPtr.Load()
	require.Len(t, conns, 1, "a fixed pool holds a single connection regardless of minConns")
	c := conns[0]

	// Saturate the only connection: neither maybeScaleUp nor pickConn's grow
	// path may add another.
	c.incInflight()
	c.incInflight()
	pool.maybeScaleUp(c)
	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.Same(t, c, picked, "with scaling disabled an over-capacity request falls back to the only connection")
	assert.Len(t, *pool.connsPtr.Load(), 1)

	// And scale-down never parks.
	other := pool.newConn()
	pool.addConn(other)
	pool.maybeScaleDown()
	assert.False(t, other.parked())
}

func TestHTTP2PoolPickConnExcludesSaturatedConn(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 2
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	full := pool.newConn()
	spare := pool.newConn()
	pool.addConn(full)
	pool.addConn(spare)
	full.incInflight()
	full.incInflight() // at the ceiling
	for i := 0; i < 5; i++ {
		spare.incInflight()
		spare.decInflight()
	}

	// full has fewer total requests ever, but is saturated: it must never be
	// picked while spare has room.
	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.Same(t, spare, picked)
	assert.Equal(t, 2, full.streamsActive())
}

func TestHTTP2PoolPickConnGrowsWhenAllSaturated(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 1
	cfg.maxConns = 2
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c := pool.newConn()
	pool.addConn(c)
	c.incInflight()

	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.NotSame(t, c, picked, "a saturated pool below maxConns should grow")
	assert.Len(t, *pool.connsPtr.Load(), 2)
}

func TestHTTP2PoolPickConnFallsBackAtMaxConns(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 1
	cfg.maxConns = 2
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	c1 := pool.newConn()
	c2 := pool.newConn()
	pool.addConn(c1)
	pool.addConn(c2)
	c1.incInflight()
	c1.incInflight()
	c2.incInflight()

	// Every connection is saturated and the pool is at maxConns: the request
	// goes to the least-loaded connection rather than failing.
	picked, err := pool.pickConn()
	require.NoError(t, err)
	assert.Same(t, c2, picked)
	assert.Len(t, *pool.connsPtr.Load(), 2)
}

func TestLeastLoadedPrefersActiveOverParked(t *testing.T) {
	busyActive := newHTTP2Conn(nil)
	idleParked := newHTTP2Conn(nil)
	for i := 0; i < 10; i++ {
		busyActive.incInflight()
	}
	require.True(t, idleParked.park())

	// The parked conn has fewer streams, but an active conn must win.
	got, err := leastLoaded([]*http2Conn{idleParked, busyActive})
	require.NoError(t, err)
	assert.Same(t, busyActive, got)

	// Among active conns, the least loaded wins.
	lighter := newHTTP2Conn(nil)
	lighter.incInflight()
	got, err = leastLoaded([]*http2Conn{busyActive, idleParked, lighter})
	require.NoError(t, err)
	assert.Same(t, lighter, got)
}

func TestLeastLoadedFallsBackToParkedOnlyWhenNoneActive(t *testing.T) {
	a := newHTTP2Conn(nil)
	b := newHTTP2Conn(nil)
	a.incInflight()
	a.incInflight()
	b.incInflight()
	require.True(t, a.park())
	require.True(t, b.park())

	got, err := leastLoaded([]*http2Conn{a, b})
	require.NoError(t, err)
	assert.Same(t, b, got, "with nothing active, the least-loaded parked conn is the last resort")
}

func TestHTTP2PoolPickConnEmptyPoolIsUnavailable(t *testing.T) {
	_, err := leastLoaded(nil)
	require.Error(t, err)
	assert.True(t, yarpcerrors.IsUnavailable(err), "got %v", err)
}

func TestHTTP2PoolIdleSingleConnIsKept(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	cfg.idleTimeout = time.Millisecond
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	c := (*pool.connsPtr.Load())[0]
	c.incInflight()
	c.decInflight()
	waitForCondition(t, time.Second, func() bool {
		return time.Since(c.idleSince()) > cfg.idleTimeout
	})

	pool.maybeScaleDown()
	pool.cleanupConns()
	assert.False(t, c.parked(), "the last active connection must never be parked")
	assert.Len(t, *pool.connsPtr.Load(), 1)
}

func TestHTTP2PoolCloseIsIdempotentAndWaitsForMonitor(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)

	pool.Close()
	pool.Close() // must not panic on a second close

	done := make(chan struct{})
	go func() {
		pool.monitorWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitor goroutine still running after Close")
	}
}

func TestHTTP2PoolConcurrentPickScaleRace(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.minConns = 1
	cfg.maxConns = 4
	cfg.maxConcurrentStreams = 4
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)
	defer pool.Close()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				c, err := pool.pickConn()
				if err != nil {
					t.Error(err)
					return
				}
				c.decInflight()
			}
		}()
	}
	// Race scale-down and cleanup against the pickers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 200; j++ {
			pool.maybeScaleDown()
			pool.cleanupConns()
		}
	}()
	wg.Wait()

	conns := *pool.connsPtr.Load()
	assert.LessOrEqual(t, len(conns), cfg.maxConns)
	assert.NotEmpty(t, activeConns(conns), "at least one connection must stay active")
	for _, c := range conns {
		assert.Zero(t, c.streamsActive(), "every reserved slot must be released")
	}
}

func TestHTTP2PoolAddConnRace(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	pool := newEmptyH2Pool(t, addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	defer pool.Close()

	const n = 16
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.addConn(pool.newConn())
		}()
	}
	wg.Wait()
	assert.Len(t, *pool.connsPtr.Load(), n, "copy-on-write adds must not lose a connection")
}

// closeSignalConn reports on closed once, the first time it is closed.
type closeSignalConn struct {
	net.Conn
	once   sync.Once
	closed chan<- struct{}
}

func (c *closeSignalConn) Close() error {
	c.once.Do(func() { c.closed <- struct{}{} })
	return c.Conn.Close()
}

// withCloseSignal wraps a Transport factory so every socket it dials signals
// on the returned channel when it is closed.
func withCloseSignal(newTransport func() *http2.Transport) (func() *http2.Transport, <-chan struct{}) {
	closed := make(chan struct{}, 16)
	return func() *http2.Transport {
		tr := newTransport()
		dial := tr.DialTLSContext
		tr.DialTLSContext = func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			conn, err := dial(ctx, network, addr, cfg)
			if err != nil {
				return nil, err
			}
			return &closeSignalConn{Conn: conn, closed: closed}, nil
		}
		return tr
	}, closed
}

// doRequest sends one GET through c's Transport and drains the response.
func doRequest(t *testing.T, c *http2Conn, addr string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/", nil)
	require.NoError(t, err)
	resp, err := c.transport.RoundTrip(req)
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

func TestHTTP2PoolClosedPoolRefusesWork(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 1
	cfg.scaleUpThreshold = 0.5
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.NewNop())
	require.NoError(t, err)

	before := len(*pool.connsPtr.Load())
	pool.Close()

	_, err = pool.pickConn()
	require.Error(t, err)
	assert.True(t, yarpcerrors.IsUnavailable(err), "got %v", err)

	_, err = pool.growPool()
	require.Error(t, err)
	assert.True(t, yarpcerrors.IsUnavailable(err), "got %v", err)

	// Even an overloaded connection must not grow a closed pool.
	c := (*pool.connsPtr.Load())[0]
	c.incInflight()
	pool.maybeScaleUp(c)
	assert.Len(t, *pool.connsPtr.Load(), before, "a closed pool must not open connections")
}

func TestHTTP2PoolCloseReleasesIdleSockets(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	newTransport, closed := withCloseSignal(newTransport)

	pool, err := newHTTP2Pool(addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	require.NoError(t, err)

	c, err := pool.pickConn()
	require.NoError(t, err)
	doRequest(t, c, addr)
	c.decInflight()

	select {
	case <-closed:
		t.Fatal("socket closed before the pool was")
	default:
	}

	pool.Close()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("idle socket was not closed by Close")
	}
}

func TestHTTP2PoolCloseReleasesBusySocketWhenRequestFinishes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.Write([]byte("ok"))
	})
	newTransport, closed := withCloseSignal(newTransport)

	pool, err := newHTTP2Pool(addr, newTransport, defaultHTTP2PoolConfig(), zap.NewNop())
	require.NoError(t, err)

	c, err := pool.pickConn()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		defer close(done)
		doRequest(t, c, addr)
	}()
	<-started

	// Close while the request is in flight: the socket is busy, so it must stay
	// open for now.
	pool.Close()
	select {
	case <-closed:
		t.Fatal("a socket with an in-flight request must not be closed")
	case <-time.After(100 * time.Millisecond):
	}

	// Once the last in-flight request completes, the socket is released
	// instead of lingering until IdleConnTimeout.
	close(release)
	<-done
	c.decInflight()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("busy socket was not closed after its last request finished")
	}
}

func TestHTTP2PoolGrowPoolBurstGrowsByOne(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 100
	cfg.maxConns = 50
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.NewNop())
	defer pool.Close()

	full := pool.newConn()
	pool.addConn(full)
	for i := 0; i < 100; i++ {
		full.incInflight() // saturated
	}

	// A burst against a saturated pool needs one more connection, not one per
	// caller: the new connection has room for all of them.
	const callers = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := pool.pickConn(); err != nil {
				t.Error(err)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Len(t, *pool.connsPtr.Load(), 2, "burst must grow the pool by one")
}

func TestHTTP2PoolClampsScalingMonitorInterval(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {})

	core, logs := observer.New(zap.DebugLevel)
	cfg := defaultHTTP2PoolConfig()
	cfg.scalingMonitorInterval = time.Millisecond
	pool, err := newHTTP2Pool(addr, newTransport, cfg, zap.New(core))
	require.NoError(t, err)
	defer pool.Close()

	assert.Equal(t, minHTTP2PoolScalingMonitorInterval, pool.cfg.scalingMonitorInterval)
	assert.Equal(t, 1, logs.FilterMessageSnippet("clamping").Len(), "clamp must be logged")
	assert.Equal(t, 1, logs.FilterMessage("http2 pool: resolved config").Len(), "resolved config must be logged at debug")
}

func TestHTTP2PoolLogsAtMaxConnectionsOnce(t *testing.T) {
	addr, newTransport := newTestH2TransportServer(t, func(w http.ResponseWriter, r *http.Request) {})

	core, logs := observer.New(zap.DebugLevel)
	cfg := defaultHTTP2PoolConfig()
	cfg.maxConcurrentStreams = 1
	cfg.maxConns = 1
	pool := newEmptyH2Pool(t, addr, newTransport, cfg, zap.New(core))
	defer pool.Close()

	c := pool.newConn()
	pool.addConn(c)
	c.incInflight()

	for i := 0; i < 5; i++ {
		_, err := pool.pickConn() // saturated and at max: falls back each time
		require.NoError(t, err)
	}
	assert.Equal(t, 1, logs.FilterMessageSnippet("at max connections").Len())
}
