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

package tchannel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
)

func TestTransportCancellationOptions(t *testing.T) {
	tests := []struct {
		name                        string
		options                     []TransportOption
		wantPropagateCancel         bool
		wantSendCancelOnCtxCanceled bool
	}{
		{
			name: "defaults",
		},
		{
			name:                "propagate cancel",
			options:             []TransportOption{PropagateCancel(true)},
			wantPropagateCancel: true,
		},
		{
			name:                        "send cancel on context canceled",
			options:                     []TransportOption{SendCancelOnContextCanceled(true)},
			wantSendCancelOnCtxCanceled: true,
		},
		{
			name: "both",
			options: []TransportOption{
				PropagateCancel(true),
				SendCancelOnContextCanceled(true),
			},
			wantPropagateCancel:         true,
			wantSendCancelOnCtxCanceled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := append([]TransportOption{ServiceName("test")}, tt.options...)
			transport, err := NewTransport(options...)
			require.NoError(t, err)
			require.NoError(t, transport.Start())
			t.Cleanup(func() {
				assert.NoError(t, transport.Stop())
			})

			connectionOptions := transport.ch.ConnectionOptions()
			assert.Equal(t, tt.wantPropagateCancel, connectionOptions.PropagateCancel)
			assert.Equal(t, tt.wantSendCancelOnCtxCanceled, connectionOptions.SendCancelOnContextCanceled)
		})
	}
}

func TestRetainDuplicatePeers(t *testing.T) {
	trans, err := NewTransport(ServiceName("test-service"))
	require.NoError(t, err)
	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()

	const address = "127.0.0.1:4040"
	sub := noSubscriber{}

	// The same address listed twice must yield two peers, each maintaining its
	// own connection, but both dialing the real address.
	first, err := trans.RetainPeer(testPeerIdentifier(address+"#1"), sub)
	require.NoError(t, err)
	second, err := trans.RetainPeer(testPeerIdentifier(address+"#2"), sub)
	require.NoError(t, err)

	assert.NotSame(t, first, second, "duplicate peers must not be shared")
	assert.Len(t, trans.peers, 2)
	assert.Equal(t, address, first.(*tchannelPeer).addr, "peer must dial the bare address")
	assert.Equal(t, address, second.(*tchannelPeer).addr, "peer must dial the bare address")

	// Retaining the same identifier again reuses the existing peer.
	firstAgain, err := trans.RetainPeer(testPeerIdentifier(address+"#1"), sub)
	require.NoError(t, err)
	assert.Same(t, first, firstAgain)
	assert.Len(t, trans.peers, 2)

	require.NoError(t, trans.ReleasePeer(testPeerIdentifier(address+"#1"), sub))
	assert.Len(t, trans.peers, 1)
	require.NoError(t, trans.ReleasePeer(testPeerIdentifier(address+"#2"), sub))
	assert.Empty(t, trans.peers)
}

func TestOnPeerStatusChangedNotifiesAllDuplicates(t *testing.T) {
	trans, err := NewTransport(ServiceName("test-service"))
	require.NoError(t, err)
	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()

	const address = "127.0.0.1:4041"

	// Build the peers directly: getOrCreatePeer would also start connection
	// loops, which would consume the notifications this test asserts on.
	first := newPeer(address, trans, trans.ch)
	second := newPeer(address, trans, trans.ch)
	other := newPeer("127.0.0.1:4042", trans, trans.ch)
	trans.peers = map[string]*tchannelPeer{
		address + "#1":     first,
		address + "#2":     second,
		"127.0.0.1:4042#1": other,
	}

	// TChannel reports status per network address; every YARPC peer sharing
	// that address has to hear about it, not just the first one found.
	trans.onPeerStatusChanged(trans.ch.Peers().GetOrAdd(address))

	assert.Len(t, first.changed, 1, "first duplicate was not notified")
	assert.Len(t, second.changed, 1, "second duplicate was not notified")
	assert.Empty(t, other.changed, "unrelated peer must not be notified")
}

type noSubscriber struct{}

func (noSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testPeerIdentifier string

func (p testPeerIdentifier) Identifier() string { return string(p) }
