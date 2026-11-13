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
	assert.EqualValues(t, 1, dialer.started.Load(), "waiters must share one dial")

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

// A destination that is down is dialed once, not once per request, until the
// backoff passes.
func TestH2SenderFailedDialNotRepeatedWithinBackoff(t *testing.T) {
	var dials atomic.Int32
	dialErr := errors.New("connection refused")
	p := newColdPeer(t, "127.0.0.1:1", DialContext(
		func(context.Context, string, string) (net.Conn, error) {
			dials.Add(1)
			return nil, dialErr
		}))

	for range 20 {
		_, err := p.h2Sender(context.Background())
		require.ErrorIs(t, err, dialErr)
	}
	assert.EqualValues(t, 1, dials.Load(), "requests within the backoff must not dial again")

	p.h2DialMu.Lock()
	p.h2DialFailedAt = time.Now().Add(-2 * h2DialFailureBackoff)
	p.h2DialMu.Unlock()

	_, err := p.h2Sender(context.Background())
	require.ErrorIs(t, err, dialErr)
	assert.EqualValues(t, 2, dials.Load(), "a dial is attempted again once the backoff has passed")
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
	p.h2DialMu.Lock()
	p.h2DialFailedAt = time.Now().Add(-2 * h2DialFailureBackoff)
	p.h2DialMu.Unlock()

	s, err := p.h2Sender(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, s)
}
