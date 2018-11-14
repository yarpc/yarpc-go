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

package yarpchttp_test

// func newTransport() yarpc.Dialer {
// 	return yarpchttp.NewTransport(
// 		yarpchttp.Tracer(opentracing.NoopTracer{}),
// 		yarpchttp.DisableKeepAlives(),
// 		yarpchttp.ConnTimeout(testtime.Millisecond),
// 		yarpchttp.ConnBackoff(backoff.None),
// 		yarpchttp.InnocenceWindow(10*time.Second),
// 		yarpchttp.NoJitter(),
// 	)
// }

// var spec = integrationtest.TransportSpec{
// 	Identify: hostport.Identify,
// 	NewServerTransport: func(t *testing.T, addr string) yarpc.Dialer {
// 		return newTransport()
// 	},
// 	NewClientTransport: func(t *testing.T) yarpc.Dialer {
// 		return newTransport()
// 	},
// 	NewUnaryOutbound: func(trans yarpc.Dialer, pc yarpc.Chooser) yarpc.UnaryTransportOutbound {
// 		return trans.(*yarpchttp.Transport).NewOutbound(pc)
// 	},
// 	NewInbound: func(trans yarpc.Dialer, addr string) yarpc.Inbound {
// 		return trans.(*yarpchttp.Transport).NewInbound(addr)
// 	},
// 	Addr: func(trans yarpc.Dialer, ib yarpc.Inbound) string {
// 		return yarpctest.ZeroAddrToHostPort(ib.(*yarpchttp.Inbound).Addr())
// 	},
// }

// func TestHTTPWithRoundRobin(t *testing.T) {
// 	ctx := context.Background()
// 	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
// 	defer cancel()

// 	permanent, permanentAddr := spec.NewServer(t, ":0")
// 	defer permanent.Stop()

// 	temporary, temporaryAddr := spec.NewServer(t, ":0")
// 	defer temporary.Stop()

// 	// Construct a client with a bank of peers. We will keep one running all
// 	// the time. We'll shut one down temporarily.
// 	// The round robin peer list should only choose peers that have
// 	// successfully connected.
// 	client, c := spec.NewClient(t, []string{
// 		permanentAddr,
// 		temporaryAddr,
// 	})
// 	defer client.Stop()

// 	integrationtest.Blast(ctx, t, c)

// 	// Shut down one task in the peer list.
// 	temporary.Stop()
// 	// One of these requests may fail since one of the peers has gone down but
// 	// the HTTP transport will not know until a request is attempted.
// 	integrationtest.Call(ctx, c)
// 	integrationtest.Call(ctx, c)
// 	// All subsequent should succeed since the peer should be removed on
// 	// connection fail.
// 	integrationtest.Blast(ctx, t, c)

// 	// Restore the server on the temporary port.
// 	restored, _ := spec.NewServer(t, temporaryAddr)
// 	defer restored.Stop()
// 	integrationtest.Blast(ctx, t, c)
// }

// func TestHTTPOnSuspect(t *testing.T) {
// 	server, serverAddr := spec.NewServer(t, ":0")

// 	client, c := spec.NewClient(t, []string{serverAddr})
// 	defer client.Stop()

// 	// Exercise OnSuspect
// 	ctx := context.Background()
// 	ctx, cancel := context.WithTimeout(ctx, 50*testtime.Millisecond)
// 	defer cancel()
// 	_ = integrationtest.Timeout(ctx, c)

// 	// Exercise the innocence window
// 	ctx = context.Background()
// 	ctx, cancel = context.WithTimeout(ctx, 50*testtime.Millisecond)
// 	defer cancel()
// 	_ = integrationtest.Timeout(ctx, c)

// 	// Validate that the peer remains available
// 	ctx = context.Background()
// 	ctx, cancel = context.WithTimeout(ctx, 50*testtime.Millisecond)
// 	defer cancel()
// 	integrationtest.Blast(ctx, t, c)

// 	// Induce the peer management loop to exit through its shutdown path.
// 	go server.Stop()
// 	ctx = context.Background()
// 	ctx, cancel = context.WithTimeout(ctx, 50*testtime.Millisecond)
// 	defer cancel()
// 	for {
// 		err := integrationtest.Call(ctx, c)
// 		if err != nil {
// 			// Yielding, it transpires, is necessary to get coverage on leaving
// 			// OnSuspect early due to the innocense window.  Even with this, it
// 			// gets coverage about as often as it wins a coin toss.
// 			runtime.Gosched()
// 			break
// 		}
// 	}
// }

// func TestIntegration(t *testing.T) {
// 	t.Skip("Skipping due to test flakiness")
// 	spec.Test(t)
// }
