// Copyright (c) 2017 Uber Technologies, Inc.
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

package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/transport/http"
)

func newTransport() peer.Transport {
	return http.NewTransport(
		http.Tracer(opentracing.NoopTracer{}),
		http.KeepAlive(20*time.Millisecond),
		http.ConnTimeout(time.Millisecond),
		http.ConnBackoff(backoff.None),
	)
}

var spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		return newTransport()
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		return newTransport()
	},
	NewUnaryOutbound: func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound {
		return x.(*http.Transport).NewOutbound(pc)
	},
	NewInbound: func(x peer.Transport, addr string) transport.Inbound {
		return x.(*http.Transport).NewInbound(addr)
	},
	Addr: func(x peer.Transport, ib transport.Inbound) string {
		return yarpctest.ZeroAddrToHostPort(ib.(*http.Inbound).Addr())
	},
}

func TestHTTPWithRoundRobin(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	permanent, permanentAddr := spec.NewServer(t, ":0")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, ":0")
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
	temporary.Stop()
	// One of these requests may fail since one of the peers has gone down but
	// the HTTP transport will not know until a request is attempted.
	err1 := integrationtest.Call(ctx, c)
	err2 := integrationtest.Call(ctx, c)
	if err1 == nil && err2 == nil {
		t.Fatal("both calls succeeded after bringing one peer down")
	}
	if err1 != nil && err2 != nil {
		t.Fatal("neither call succeeded after bringing one peer down")
	}
	// All subsequent should succeed since the peer should be removed on
	// connection fail.
	integrationtest.Blast(ctx, t, c)

	// Restore the server on the temporary port.
	// TODO the http client unfortunately reuses idle connections from the
	// prior generation of the http server on this port, rendering this test
	// useless.
	// restored, _ := spec.NewServer(t, temporaryAddr)
	// defer restored.Stop()
	// integrationtest.Blast(ctx, t, c)
}

func TestIntegration(t *testing.T) {
	t.Skip("Skipping due to test flakiness")
	spec.Test(t)
}
