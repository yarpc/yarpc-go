// Copyright (c) 2024 Uber Technologies, Inc.
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
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/zap/zaptest"
)

var spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		return NewTransport(BackoffStrategy(backoff.None))
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		return NewTransport(BackoffStrategy(backoff.None))
	},
	NewUnaryOutbound: func(x peer.Transport, peerChooser peer.Chooser) transport.UnaryOutbound {
		return x.(*Transport).NewOutbound(peerChooser)
	},
	NewInbound: func(t peer.Transport, address string) transport.Inbound {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			panic(err.Error())
		}
		return t.(*Transport).NewInbound(listener)
	},
	Addr: func(_ peer.Transport, inbound transport.Inbound) string {
		return yarpctest.ZeroAddrToHostPort(inbound.(*Inbound).listener.Addr())
	},
}

func TestPeerWithRoundRobin(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	permanent, permanentAddr := spec.NewServer(t, "127.0.0.1:0")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, "127.0.0.1:0")
	defer temporary.Stop()

	// Construct a client with a bank of peers. We will keep one running all
	// the time. We'll shut one down temporarily.
	// The round robin peer list should only choose peers that have
	// successfully connected.
	client, c := spec.NewClient(t, []string{
		permanentAddr,
		temporaryAddr,
	})
	defer client.Stop()

	integrationtest.Blast(ctx, t, c)

	// Shut down one task in the peer list.
	require.NoError(t, temporary.Stop())

	// One of these requests may fail since one of the peers has gone down but
	// the gRPC transport will not know until a request is attempted.
	integrationtest.Call(ctx, c)
	integrationtest.Call(ctx, c)
	// All subsequent should succeed since the peer should be removed on
	// connection fail.
	integrationtest.Blast(ctx, t, c)

	// Restore the server on the temporary port.
	restored, _ := spec.NewServer(t, temporaryAddr)
	defer restored.Stop()
	integrationtest.Blast(ctx, t, c)
}

func TestPeerIntegration(t *testing.T) {
	t.Skip("Skipping due to test flakiness")
	spec.Test(t)
}

func TestReconnectionCalledForIDLE(t *testing.T) {
	logger := zaptest.NewLogger(t)

	grpcTransport := NewTransport(Logger(logger))
	require.NoError(t, grpcTransport.Start())

	chooser := roundrobin.New(grpcTransport, roundrobin.Logger(logger))
	outbound := grpcTransport.NewOutbound(chooser)
	require.NoError(t, outbound.Start())

	permanent, permanentAddr := spec.NewServer(t, "127.0.0.1:0")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, "127.0.0.1:0")
	defer temporary.Stop()

	require.NoError(t, chooser.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify(permanentAddr),
			hostport.Identify(temporaryAddr),
		},
	}))

	dispatcher := integrationtest.CreateAndStartClientDispatcher(t, outbound)
	defer dispatcher.Stop()

	rawClient := raw.New(dispatcher.ClientConfig(integrationtest.ServiceName))

	makeBlastCall(t, rawClient, 1*time.Second)

	// Shut down one service.
	require.NoError(t, temporary.Stop())

	waitForPeerStatus(t, chooser, temporaryAddr, peer.Unavailable, 2*time.Second)
	makeBlastCall(t, rawClient, 1*time.Second)

	// Restore the server on the temporary port.
	restored, _ := spec.NewServer(t, temporaryAddr)
	defer restored.Stop()

	waitForPeerStatus(t, chooser, temporaryAddr, peer.Available, 2*time.Second)
	makeBlastCall(t, rawClient, 1*time.Second)
}

func waitForPeerStatus(t *testing.T, peerList *roundrobin.List, peerAddr string, status peer.ConnectionStatus, wait time.Duration) {
	peerAvailable := make(chan struct{})
	go func() {
		for {
			for _, p := range peerList.Peers() {
				if p.Identifier() == peerAddr {
					if p.Status().ConnectionStatus == status {
						close(peerAvailable)
						return
					}
				}
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-time.After(wait):
		t.Fatal("failed waiting to connect to peer")
	case <-peerAvailable:
		return
	}
}

func makeBlastCall(t *testing.T, rawClient raw.Client, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	integrationtest.Blast(ctx, t, rawClient)
}
