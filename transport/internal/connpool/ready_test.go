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

package connpool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connectingSet tracks which fake connections are still "connecting", the way
// a gRPC connection is right after grpc.Dial returns.
type connectingSet struct{ m sync.Map }

func (s *connectingSet) add(c *fakeConn) { s.m.Store(c, struct{}{}) }
func (s *connectingSet) ready(c *fakeConn) bool {
	_, connecting := s.m.Load(c)
	return !connecting
}

// newReadyTestPool returns a pool whose connections are ready unless added to
// the returned set, configured to scale up as soon as a connection has any
// stream.
func newReadyTestPool(t *testing.T, minConns int) (*Pool[*fakeConn], *connectingSet) {
	t.Helper()
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 10
	cfg.ScaleUpThreshold = 0.1 // threshold = 1 stream
	cfg.MinConnections = minConns
	p, _ := newTestPool(t, cfg)
	set := &connectingSet{}
	p.Ready = set.ready
	return p, set
}

func TestPickConn_PrefersReadyOverIdleConnecting(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	loaded := p.LoadConns()[0]
	for range 5 {
		loaded.IncStreamCount()
	}
	fresh, err := p.AddConn()
	require.NoError(t, err)
	connecting.add(fresh.Conn)

	// The fresh connection has fewer streams but cannot serve yet.
	assert.Same(t, loaded, p.PickConn())

	connecting.m.Delete(fresh.Conn)
	assert.Same(t, fresh, p.PickConn(), "a ready connection competes on load again")
}

func TestPickConn_FallsBackWhenNoneReady(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	connecting.add(conns[0].Conn)
	connecting.add(conns[1].Conn)
	conns[0].IncStreamCount()

	assert.Same(t, conns[1], p.PickConn(), "least loaded of the connecting connections")
}

func TestPickConn_IgnoresReadyWhenUnset(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	conns[0].IncStreamCount()
	assert.Same(t, conns[1], p.PickConn())
}

func TestScaler_TryScaleUpWaitsForConnectingConn(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	loaded := p.LoadConns()[0]
	loaded.IncStreamCount()
	loaded.IncStreamCount()

	// First scale-up dials a connection that stays "connecting".
	p.Dial = wrapDial(p.Dial, connecting)
	p.TryScaleUp(loaded)
	require.Eventually(t, func() bool { return len(p.LoadConns()) == 2 }, time.Second, time.Millisecond)

	// The loaded connection is still over the threshold on every request, but
	// the pool must not keep dialing while the new connection connects.
	for range 50 {
		p.TryScaleUp(p.PickConn())
	}
	time.Sleep(20 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 2, "no further dials while a connection is connecting")
}

func TestScaler_TryScaleUpResumesOnceConnectionReady(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	loaded := p.LoadConns()[0]
	loaded.IncStreamCount()
	loaded.IncStreamCount()
	p.Dial = wrapDial(p.Dial, connecting)
	p.TryScaleUp(loaded)
	require.Eventually(t, func() bool { return len(p.LoadConns()) == 2 }, time.Second, time.Millisecond)

	fresh := p.LoadConns()[1]
	connecting.m.Delete(fresh.Conn)
	fresh.IncStreamCount()
	fresh.IncStreamCount() // both ready connections are now over the threshold
	p.TryScaleUp(p.PickConn())
	require.Eventually(t, func() bool { return len(p.LoadConns()) == 3 }, time.Second, time.Millisecond)
}

func TestScaler_TryScaleUpIgnoresStuckConnectingConn(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	stuck := conns[1]
	connecting.add(stuck.Conn)
	stuck.createdAt = time.Now().Add(-2 * connectGracePeriod)
	conns[0].IncStreamCount()
	conns[0].IncStreamCount()

	p.TryScaleUp(conns[0])
	require.Eventually(t, func() bool { return len(p.LoadConns()) == 3 }, time.Second, time.Millisecond,
		"a connection stuck connecting past the grace period must not block scale-up")
}

// wrapDial makes every connection dialed through dial start out connecting.
func wrapDial(dial func(ctx context.Context) (*fakeConn, error), set *connectingSet) func(ctx context.Context) (*fakeConn, error) {
	return func(ctx context.Context) (*fakeConn, error) {
		c, err := dial(ctx)
		if err == nil {
			set.add(c)
		}
		return c, err
	}
}

func TestScaler_TryScaleUpDialsWhenNothingIsReadyYet(t *testing.T) {
	p, connecting := newReadyTestPool(t, 1)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	only := p.LoadConns()[0]
	connecting.add(only.Conn)
	only.IncStreamCount()
	only.IncStreamCount()

	// No connection is ready, so no load is being carried that a connecting
	// connection could take over: scale-up behaves as it did before Ready.
	p.TryScaleUp(only)
	require.Eventually(t, func() bool { return len(p.LoadConns()) == 2 }, time.Second, time.Millisecond)
}
