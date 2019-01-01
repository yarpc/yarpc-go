// Copyright (c) 2019 Uber Technologies, Inc.
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

package tchannel_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/transport/tchannel"
)

var spec = integrationtest.TransportSpec{
	Identify: identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		x, err := tchannel.NewTransport(
			tchannel.ServiceName("service"),
			tchannel.ListenAddr(addr),
		)
		require.NoError(t, err, "must construct transport")
		return x
	},
	NewInbound: func(x peer.Transport, addr string) transport.Inbound {
		return x.(*tchannel.Transport).NewInbound()
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		x, err := tchannel.NewTransport(
			tchannel.ServiceName("client"),
			tchannel.ConnTimeout(10*testtime.Millisecond),
			tchannel.ConnBackoff(backoff.None),
		)
		require.NoError(t, err, "must construct transport")
		return x
	},
	NewUnaryOutbound: func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound {
		return x.(*tchannel.Transport).NewOutbound(pc)
	},
	Addr: func(x peer.Transport, ib transport.Inbound) string {
		return yarpctest.ZeroAddrStringToHostPort(x.(*tchannel.Transport).ListenAddr())
	},
}

// TestWithRoundRobin verifies that TChannel appropriately notifies all
// subscribed peer lists when peers become available and unavailable.
// It does so by constructing a round robin peer list backed by the TChannel transport,
// communicating to three servers. One will always work. One will go down
// temporarily. One will be a bogus TCP port that never completes a TChannel
// handshake.
func TestWithRoundRobin(t *testing.T) {
	t.Skip()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()

	permanent, permanentAddr := spec.NewServer(t, "")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, "")
	defer temporary.Stop()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "listen for bogus server")
	invalidAddr := l.Addr().String()
	defer l.Close()

	// Construct a client with a bank of peers. We will keep one running all
	// the time. We'll shut one down temporarily. One will be invalid.
	// The round robin peer list should only choose peers that have
	// successfully connected.
	client, c := spec.NewClient(t, []string{
		permanentAddr,
		temporaryAddr,
		invalidAddr,
	})
	defer client.Stop()

	// All requests should succeed. The invalid peer never enters the rotation.
	integrationtest.Blast(ctx, t, c)

	// Shut down one task in the peer list.
	temporary.Stop()
	// One of these requests may fail since one of the peers has gone down but
	// the TChannel transport will not know until a request is attempted.
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

func TestIntegration(t *testing.T) {
	spec.Test(t)
}

type noSub struct{}

func (noSub) NotifyStatusChanged(pid peer.Identifier) {}

func TestCancelMaintainConn(t *testing.T) {
	transport, err := tchannel.NewTransport()
	require.NoError(t, err)
	transport.Start()
	transport.Stop()
	_, err = transport.RetainPeer(identify("127.0.0.1:66408"), noSub{})
	require.NoError(t, err)
}

func identify(id string) peer.Identifier {
	return &testIdentifier{id}
}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
