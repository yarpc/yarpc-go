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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentReleasePeerAndStop releases isolated peers while the transport
// stops. A failed first dial leaves a started, empty pool, as happens when a
// destination is down. Registering a released pool's teardown must not race
// with Stop waiting for it.
func TestConcurrentReleasePeerAndStop(t *testing.T) {
	const peers = 16
	id := testIdentifier{"127.0.0.1:1"}
	sub := idSubscriber{1}
	for iteration := 0; iteration < 500; iteration++ {
		tr := NewTransport(DialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("destination unavailable")
		}))
		if err := tr.Start(); err != nil {
			t.Fatal(err)
		}
		dialers := make([]*Dialer, peers)
		for i := range dialers {
			d := tr.NewDialer().WithConnectionIsolation()
			pi, err := d.RetainPeer(id, sub)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = pi.(*httpPeer).h2Sender()
			dialers[i] = d
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(peers + 1)
		for _, d := range dialers {
			go func(d *Dialer) {
				defer wg.Done()
				<-start
				if err := d.ReleasePeer(id, sub); err != nil {
					t.Error(err)
				}
			}(d)
		}
		go func() {
			defer wg.Done()
			<-start
			if err := tr.Stop(); err != nil {
				t.Error(err)
			}
		}()
		close(start)
		wg.Wait()
	}
}

// Stop's pool snapshot must not miss a cold peer's first pool. Block an
// existing connection's close so a request can reach the cold peer after the
// snapshot, while Stop is still waiting for the existing pool.
func TestH2SenderDoesNotCreatePoolDuringTransportStop(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()
	closeStarted, allowClose := make(chan struct{}), make(chan struct{})
	var dials atomic.Int32
	tr := NewTransport(DialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		if dials.Add(1) == 1 {
			return &shutdownBlockingConn{Conn: conn, closeStarted: closeStarted, allowClose: allowClose}, nil
		}
		return conn, nil
	}))
	require.NoError(t, tr.Start())
	id, sub := testIdentifier{strings.TrimPrefix(server.URL, "http://")}, idSubscriber{1}
	aDialer, bDialer := tr.NewDialer().WithConnectionIsolation(), tr.NewDialer().WithConnectionIsolation()
	a, err := aDialer.RetainPeer(id, sub)
	require.NoError(t, err)
	b, err := bDialer.RetainPeer(id, sub)
	require.NoError(t, err)
	aPeer, bPeer := a.(*httpPeer), b.(*httpPeer)
	stopResult := make(chan error, 1)
	stopStarted := false
	t.Cleanup(func() {
		close(allowClose)
		assert.NoError(t, aDialer.ReleasePeer(id, sub))
		assert.NoError(t, bDialer.ReleasePeer(id, sub))
		if stopStarted {
			select {
			case err := <-stopResult:
				assert.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Error("transport shutdown did not finish")
			}
		} else {
			assert.NoError(t, tr.Stop())
		}
		for _, p := range []*httpPeer{aPeer, bPeer} {
			if pool := p.loadH2Pool(); pool != nil {
				pool.Wait()
			}
		}
	})
	_, err = aPeer.h2Sender()
	require.NoError(t, err)
	require.Nil(t, bPeer.loadH2Pool())
	stopStarted = true
	go func() { stopResult <- tr.Stop() }()
	select {
	case <-closeStarted:
		// Stop has collected the pool pointers and is closing the first pool.
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not start closing the first connection")
	}
	_, err = bPeer.h2Sender()
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, bPeer.loadH2Pool(), "shutdown must prevent a pool from escaping its snapshot")
	assert.EqualValues(t, 1, dials.Load(), "a cold peer must not dial after shutdown starts")
}

type shutdownBlockingConn struct {
	net.Conn
	closeStarted chan<- struct{}
	allowClose   <-chan struct{}
	once         sync.Once
}

func (c *shutdownBlockingConn) Close() error {
	c.once.Do(func() { close(c.closeStarted) })
	<-c.allowClose
	return c.Conn.Close()
}
