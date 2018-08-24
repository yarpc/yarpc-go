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

package integrationtest

// const (
// 	maxAttempts        = 1000
// 	concurrentAttempts = 100
// )

// // TransportSpec specifies how to create test clients and servers for a transport.
// type TransportSpec struct {
// 	NewServerTransport func(t *testing.T, addr string) yarpc.Dialer
// 	NewClientTransport func(t *testing.T) yarpc.Dialer
// 	NewInbound         func(trans yarpc.Dialer, addr string, router yarpc.Router) func()
// 	NewUnaryOutbound   func(trans yarpc.Dialer, pc yarpc.Chooser) yarpc.UnaryOutbound
// 	Identify           func(addr string) yarpc.Identifier
// 	Addr               func(trans yarpc.Dialer, inbound yarpc.Inbound) string
// }

// // Test runs reusable tests with the transport spec.
// func (s TransportSpec) Test(t *testing.T) {
// 	t.Run("reuseConnRoundRobin", s.TestConcurrentClientsRoundRobin)
// 	t.Run("backoffConnRoundRobin", s.TestBackoffConnRoundRobin)
// 	t.Run("connectAndStopRoundRobin", s.TestConnectAndStopRoundRobin)
// }

// // NewClient returns a running dispatcher and a raw client for the echo
// // procedure.
// func (s TransportSpec) NewClient(t *testing.T, addrs []string) (yarpcraw.Client, func()) {
// 	ids := make([]yarpc.Identifier, len(addrs))
// 	for i, addr := range addrs {
// 		ids[i] = s.Identify(addr)
// 	}

// 	trans := s.NewClientTransport(t)

// 	pl := yarpcroundrobin.New(trans)
// 	pl.Update(yarpc.ListUpdates{
// 		Additions: yarpc.Addresses(addrs),
// 	})
// 	pl.Start()

// 	ob := s.NewUnaryOutbound(trans, pl)
// 	rawClient := yarpcraw.New(&yarpc.OutboundConfig{
// 		CallerName: "service",
// 		Outbounds: &yarpc.Outbounds{
// 			ServiceName: "service",
// 			Unary:       ob,
// 		},
// 	})

// 	return rawClient, func() {
// 		trans.Stop()
// 		pl.Stop()
// 	}
// }

// // NewServer creates an echo server using the given inbound from any transport.
// func (s TransportSpec) NewServer(t *testing.T, addr string) (string, func()) {
// 	router := NewRouter()
// 	Register(router)
// 	trans := s.NewServerTransport(t, addr)
// 	inbound := s.NewInbound(trans, addr)
// 	require.NoError(t, inbound.Start(), "start inbound")

// 	return s.Addr(trans, inbound), func() {
// 		require.NoError(t, inbound.Stop())
// 	}
// }

// // TestConnectAndStopRoundRobin is a test that any transport can apply to
// // exercise a transport dropping connections if the transport is stopped before
// // a pending request can complete.
// func (s TransportSpec) TestConnectAndStopRoundRobin(t *testing.T) {
// 	addr := "127.0.0.1:31172"

// 	client, rawClient := s.NewClient(t, []string{addr})

// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)
// 		ctx := context.Background()
// 		ctx, cancel := context.WithTimeout(ctx, 50*testtime.Millisecond)
// 		defer cancel()
// 		assert.Error(t, Call(ctx, rawClient))
// 	}()

// 	time.Sleep(10 * testtime.Millisecond)
// 	assert.NoError(t, client.Stop())

// 	<-done
// }

// // TestConcurrentClientsRoundRobin is a reusable test that any transport can
// // apply to cover connection reuse.
// func (s TransportSpec) TestConcurrentClientsRoundRobin(t *testing.T) {
// 	var wg sync.WaitGroup
// 	count := concurrentAttempts

// 	server, addr := s.NewServer(t, ":0")
// 	defer server.Stop()

// 	client, rawClient := s.NewClient(t, []string{addr})
// 	defer client.Stop()

// 	wg.Add(count)
// 	call := func() {
// 		defer wg.Done()
// 		ctx := context.Background()
// 		ctx, cancel := context.WithTimeout(ctx, 150*testtime.Millisecond)
// 		defer cancel()
// 		assert.NoError(t, Call(ctx, rawClient))
// 	}
// 	for i := 0; i < count; i++ {
// 		go call()
// 		time.Sleep(10 * testtime.Millisecond)
// 	}

// 	wg.Wait()
// }

// // TestBackoffConnRoundRobin is a reusable test that any transport can apply to
// // cover connection management backoff.
// func (s TransportSpec) TestBackoffConnRoundRobin(t *testing.T) {
// 	addr := "127.0.0.1:31782"

// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)

// 		client, rawClient := s.NewClient(t, []string{addr})
// 		defer client.Stop()

// 		ctx := context.Background()
// 		ctx, cancel := context.WithTimeout(ctx, testtime.Second)
// 		defer cancel()

// 		// Eventually succeeds, when the server comes online.
// 		assert.NoError(t, Call(ctx, rawClient))
// 	}()

// 	// Give the client time to make multiple connection attempts.
// 	time.Sleep(10 * testtime.Millisecond)
// 	server, _ := s.NewServer(t, addr)
// 	defer server.Stop()

// 	<-done
// }

// // Blast sends a blast of calls to the client and verifies that they do not
// // err.
// func Blast(ctx context.Context, t *testing.T, rawClient yarpcraw.Client) {
// 	for i := 0; i < 10; i++ {
// 		assert.NoError(t, Call(ctx, rawClient))
// 	}
// }

// // CallUntilSuccess sends a request until it succeeds.
// func CallUntilSuccess(t *testing.T, rawClient yarpcraw.Client, interval time.Duration) {
// 	for i := 0; i < maxAttempts; i++ {
// 		ctx := context.Background()
// 		ctx, cancel := context.WithTimeout(ctx, interval)
// 		err := Call(ctx, rawClient)
// 		cancel()
// 		if err == nil {
// 			return
// 		}
// 	}
// 	assert.Fail(t, "call until success failed multiple times")
// }

// // Call sends an echo request to the client.
// func Call(ctx context.Context, rawClient yarpcraw.Client) error {
// 	ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
// 	defer cancel()
// 	res, err := rawClient.Call(ctx, "echo", []byte("hello"))
// 	if err != nil {
// 		return err
// 	}
// 	if !bytes.Equal(res, []byte("hello")) {
// 		return fmt.Errorf("unexpected response %+v", res)
// 	}
// 	return nil
// }

// // Timeout sends a request to the client, which will timeout on the server.
// func Timeout(ctx context.Context, rawClient yarpcraw.Client) error {
// 	_, err := rawClient.Call(ctx, "timeout", []byte{})
// 	return err
// }

// // Register registers an echo procedure handler on a router.
// func Register(router yarpc.Router) {
// 	router.Register(yarpcraw.Procedure("echo", func(ctx context.Context, req []byte) ([]byte, error) {
// 		return req, nil
// 	}))
// 	router.Register(yarpcraw.Procedure("timeout", func(ctx context.Context, req []byte) ([]byte, error) {
// 		<-ctx.Done()
// 		return nil, context.DeadlineExceeded
// 	}))
// }
