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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
)

// idSubscriber is a comparable subscriber with a distinct identity, used to
// model distinct outbounds in tests.
type idSubscriber struct{ id int }

func (idSubscriber) NotifyStatusChanged(peer.Identifier) {}

// TestDialersSharePeerByDefault verifies that ordinary dialers retain the
// existing address-based peer sharing behavior.
func TestDialersSharePeerByDefault(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}
	dialer1 := transport.NewDialer()
	dialer2 := transport.NewDialer()
	sub1, sub2 := idSubscriber{1}, idSubscriber{2}

	p1, err := dialer1.RetainPeer(id, sub1)
	require.NoError(t, err)
	p2, err := dialer2.RetainPeer(id, sub2)
	require.NoError(t, err)

	assert.Same(t, p1, p2)
	assert.Len(t, transport.peers, 1)
}

// TestIsolatedDialersDoNotSharePeer verifies that isolated dialers get
// distinct peers, and therefore distinct dedicated *http2.Transports, while
// subscribers using the same dialer continue to share a peer.
func TestIsolatedDialersDoNotSharePeer(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}
	baseDialer := transport.NewDialer()
	dialer1 := baseDialer.WithConnectionIsolation()
	dialer2 := baseDialer.WithConnectionIsolation()
	sub1, sub2, sub3 := idSubscriber{1}, idSubscriber{2}, idSubscriber{3}

	p1, err := dialer1.RetainPeer(id, sub1)
	require.NoError(t, err)
	p1Again, err := dialer1.RetainPeer(id, sub2)
	require.NoError(t, err)

	// Request-scoped subscribers within one outbound share its peer.
	assert.Same(t, p1, p1Again)
	assert.Len(t, transport.peers, 1)

	p2, err := dialer2.RetainPeer(id, sub3)
	require.NoError(t, err)

	// Two isolated dialers to the same address get separate peers, and
	// therefore separate dedicated *http2.Transports (see peer.go).
	assert.NotSame(t, p1, p2)
	assert.Len(t, transport.peers, 2)
	httpPeer1, ok := p1.(*httpPeer)
	require.True(t, ok)
	httpPeer2, ok := p2.(*httpPeer)
	require.True(t, ok)
	assert.NotSame(t, httpPeer1.h2Transport, httpPeer2.h2Transport,
		"isolated peers must own independent *http2.Transports")

	// Releasing one subscriber leaves the other subscriber and dialer's peer
	// intact.
	require.NoError(t, dialer1.ReleasePeer(id, sub1))
	assert.Len(t, transport.peers, 2)

	require.NoError(t, dialer1.ReleasePeer(id, sub2))
	assert.Len(t, transport.peers, 1)
}

// TestIsolatedDialersConcurrent hammers RetainPeer/ReleasePeer from many
// independently isolated dialers. Run with -race to detect data races on the
// shared peers map.
func TestIsolatedDialersConcurrent(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}

	const (
		goroutines = 16
		iterations = 50
	)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			dialer := transport.NewDialer().WithConnectionIsolation()
			sub := idSubscriber{g}
			for range iterations {
				if _, err := dialer.RetainPeer(id, sub); !assert.NoError(t, err) {
					return
				}
				assert.NoError(t, dialer.ReleasePeer(id, sub))
			}
		}(g)
	}
	wg.Wait()
}
