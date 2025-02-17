// Copyright (c) 2025 Uber Technologies, Inc.
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

package grpc

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"google.golang.org/grpc"
)

func TestTransportLifecycle(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	assert.True(t, transport.IsRunning())
	assert.NoError(t, transport.Stop())
	assert.False(t, transport.IsRunning())
}

func TestRetainReleasePeerSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	peerSubscriber := testPeerSubscriber{}

	peer, err := transport.RetainPeer(testIdentifier{address}, peerSubscriber)
	assert.NoError(t, err)
	assert.Equal(t, peer, transport.addressToPeer[address])
	assert.NoError(t, transport.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

func TestRetainReleasePeerErrorPeerIdentifier(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()
}

func TestReleasePeerErrorNoPeer(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := "not_retained"
	peerSubscriber := testPeerSubscriber{}

	assert.Equal(t, peer.ErrTransportHasNoReferenceToPeer{
		TransportName:  "grpc.Transport",
		PeerIdentifier: address,
	}, transport.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

type testPeerSubscriber struct{}

func (testPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
