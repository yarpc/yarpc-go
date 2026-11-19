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
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
)

// blockingDialer is a DialContext that holds every dial until release is
// closed (or the dial's ctx ends), then dials the real address, counting how
// many dials were started.
type blockingDialer struct {
	started atomic.Int32
	release chan struct{}
	once    sync.Once
	// err, if set, is returned once the dial is released instead of dialing.
	err error
}

func newBlockingDialer() *blockingDialer { return &blockingDialer{release: make(chan struct{})} }

func (d *blockingDialer) unblock() { d.once.Do(func() { close(d.release) }) }

func (d *blockingDialer) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	d.started.Add(1)
	select {
	case <-d.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if d.err != nil {
		return nil, d.err
	}
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func newColdPeer(t *testing.T, addr string, opts ...TransportOption) *httpPeer {
	t.Helper()
	tr := NewTransport(opts...)
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })
	p := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
			pool.Wait()
		}
	})
	return p
}

// A request waiting for a cold peer's first dial gives up when its own
// deadline passes, rather than queueing for the whole dial, and the dial it
// was waiting on is shared, not repeated.
func TestH2SenderWaiterReturnsOnOwnDeadline(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	t.Cleanup(dialer.unblock)

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))

	const waiters = 10
	start := time.Now()
	var wg sync.WaitGroup
	errs := make([]error, waiters)
	for i := range waiters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, errs[i] = p.h2Sender(ctx)
		}()
	}
	wg.Wait()

	assert.Less(t, time.Since(start), 10*testtime.Second, "waiters must not block on the dial")
	for _, err := range errs {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}
	// The dial runs in its own goroutine, so it may start after the waiters
	// have already given up.
	require.Eventually(t, func() bool { return dialer.started.Load() == 1 },
		testtime.Second, time.Millisecond, "waiters must share one dial")

	// The dial outlives the waiters that gave up: once it completes, the next
	// request finds the connection without dialing again.
	dialer.unblock()
	s, err := p.h2Sender(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.EqualValues(t, 1, dialer.started.Load(), "the abandoned dial must be reused, not repeated")
}

// Every request that arrives during the first dial is served by it.
func TestH2SenderWaitersShareFirstDial(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	t.Cleanup(dialer.unblock)

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))

	const waiters = 20
	senders := make([]sender, waiters)
	errs := make([]error, waiters)
	var wg sync.WaitGroup
	for i := range waiters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			senders[i], errs[i] = p.h2Sender(context.Background())
		}()
	}
	require.Eventually(t, func() bool { return dialer.started.Load() == 1 },
		testtime.Second, time.Millisecond, "the first dial must start")
	dialer.unblock()
	wg.Wait()

	for i := range waiters {
		require.NoError(t, errs[i])
		assert.Same(t, senders[0].(*h2ConnSender).wrapper, senders[i].(*h2ConnSender).wrapper)
	}
	assert.EqualValues(t, 1, dialer.started.Load())
}

// Through h2PeerSender, a request whose deadline ends its wait reports its
// context error, not an unavailable peer.
func TestH2PeerSenderReturnsContextErrorWhileDialing(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	t.Cleanup(dialer.unblock)

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	_, err = (&h2PeerSender{peer: p}).Do(req)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// Requests that arrive while the first dial is in flight share its outcome
// instead of each dialing, and a failure is not remembered: the next request
// dials again, so a destination that comes back is reached immediately.
func TestH2SenderSharesFailedDialButDoesNotRememberIt(t *testing.T) {
	dialErr := errors.New("connection refused")
	dialer := newBlockingDialer()
	dialer.err = dialErr
	t.Cleanup(dialer.unblock)
	p := newColdPeer(t, "127.0.0.1:1", DialContext(dialer.dial))

	first, err := p.joinH2Dial()
	require.NoError(t, err)
	for range 9 {
		d, err := p.joinH2Dial()
		require.NoError(t, err)
		assert.Same(t, first, d, "a request during the dial must join it")
	}

	dialer.unblock()
	<-first.done
	require.ErrorIs(t, first.err, dialErr)
	assert.EqualValues(t, 1, dialer.started.Load(), "all of them shared one dial")

	second, err := p.joinH2Dial()
	require.NoError(t, err)
	assert.NotSame(t, first, second, "a failed dial must not be reused")
	<-second.done
	require.ErrorIs(t, second.err, dialErr)
	assert.EqualValues(t, 2, dialer.started.Load(), "the next request dials again")
}

// A failed first dial must not stop the peer from connecting once the
// destination is reachable again.
func TestH2SenderRecoversAfterFailedDial(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	var failing atomic.Bool
	failing.Store(true)
	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(
		func(ctx context.Context, network, addr string) (net.Conn, error) {
			if failing.Load() {
				return nil, errors.New("connection refused")
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}))

	_, err := p.h2Sender(context.Background())
	require.Error(t, err)

	failing.Store(false)
	s, err := p.h2Sender(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, s)
}

// A peer that no HTTP/2 request has used gets no pool and no dial from a
// warm-up, so HTTP/1 peers cost nothing extra.
func TestH2WarmUpSkipsPeerNotUsedForHTTP2(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	dialer.unblock()

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))
	p.warmH2()

	// A warm-up that ran would do so in a goroutine, so assert the absence over
	// a short window; Never fails as soon as a pool or dial shows up.
	require.Never(t, func() bool {
		return p.loadH2Pool() != nil || dialer.started.Load() > 0
	}, 100*time.Millisecond, time.Millisecond, "warm-up must not touch a peer HTTP/2 has not used")
}

// Warm-ups share the single in-flight dial, so calling it repeatedly (every
// time the peer becomes Available, say) never adds a second connection.
func TestH2WarmUpDialsOnce(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	dialer.unblock()

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))
	p.h2Used.Store(true)
	for range 5 {
		p.warmH2()
	}

	require.Eventually(t, func() bool {
		pool := p.loadH2Pool()
		return pool != nil && pool.PickConn() != nil
	}, testtime.Second, time.Millisecond, "warm-up must leave a connection ready")
	p.warmH2()
	assert.EqualValues(t, 1, dialer.started.Load())
}

// When the pool's only connection is evicted, a peer that serves HTTP/2
// replaces it without waiting for the next request.
func TestH2WarmUpReplacesEvictedConnection(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	dialer := newBlockingDialer()
	dialer.unblock()

	p := newColdPeer(t, strings.TrimPrefix(server.URL, "http://"), DialContext(dialer.dial))
	s, err := p.h2Sender(context.Background())
	require.NoError(t, err)
	evicted := s.(*h2ConnSender).wrapper

	evicted.Cancel()

	require.Eventually(t, func() bool {
		w := p.loadH2Pool().PickConn()
		return w != nil && w != evicted
	}, testtime.Second, time.Millisecond, "evicted connection must be replaced with no request")
	// At least the warm-up's dial: the scaling monitor's first min-fill pass can
	// also replace the connection if the eviction lands right after the first
	// dial, so the exact count is not deterministic.
	assert.GreaterOrEqual(t, dialer.started.Load(), int32(2))
}
