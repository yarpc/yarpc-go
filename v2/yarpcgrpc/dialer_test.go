// Copyright (c) 2018 Uber Technologies, Inc.
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
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"google.golang.org/grpc"
)

func TestRetainReleasePeerSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	dialer := Dialer{}
	assert.NoError(t, dialer.Start(context.Background()))
	defer func() { assert.NoError(t, dialer.Stop(context.Background())) }()

	address := listener.Addr().String()
	peerSubscriber := testPeerSubscriber{}

	peer, err := dialer.RetainPeer(testIdentifier{address}, peerSubscriber)
	assert.NoError(t, err)
	assert.Equal(t, peer, dialer.addressToPeer[address])
	assert.NoError(t, dialer.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

func TestRetainReleasePeerErrorPeerIdentifier(t *testing.T) {
	dialer := Dialer{}
	assert.NoError(t, dialer.Start(context.Background()))
	defer func() { assert.NoError(t, dialer.Stop(context.Background())) }()
}

func TestReleasePeerErrorNoPeer(t *testing.T) {
	dialer := Dialer{}
	assert.NoError(t, dialer.Start(context.Background()))
	defer func() { assert.NoError(t, dialer.Stop(context.Background())) }()

	address := "not_retained"
	peerSubscriber := testPeerSubscriber{}

	assert.Equal(t, peer.ErrDialerHasNoReferenceToPeer{
		DialerName:     "grpc.Dialer",
		PeerIdentifier: address,
	}, dialer.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

type testPeerSubscriber struct{}

func (testPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
