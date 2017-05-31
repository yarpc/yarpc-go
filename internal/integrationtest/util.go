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

package integrationtest

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	peerbind "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/roundrobin"
)

// TransportSpec specifies how to create test clients and servers for a transport.
type TransportSpec struct {
	NewServerTransport func(t *testing.T, addr string) peer.Transport
	NewClientTransport func(t *testing.T) peer.Transport
	NewInbound         func(x peer.Transport, addr string) transport.Inbound
	NewUnaryOutbound   func(x peer.Transport, pc peer.Chooser) transport.UnaryOutbound
	Identify           func(addr string) peer.Identifier
	Addr               func(x peer.Transport, ib transport.Inbound) string
}

// Test runs reusable tests with the transport spec.
func (s TransportSpec) Test(t *testing.T) {
	t.Run("reuse connection with round robin", s.TestReuseConnectionWithRoundRobin)
	t.Run("backoff reconnection with round robin", s.TestBackoffWithRoundRobin)
	t.Run("reconnect using round robin", s.TestReconnectWithRoundRobin)
	t.Run("lose patience connecting with round robin", s.TestLosePatienceWithRoundRobin)
}

// NewClient returns a running dispatcher and a raw client for the echo
// procedure.
func (s TransportSpec) NewClient(t *testing.T, addrs []string) (*yarpc.Dispatcher, raw.Client) {
	// Convert peer addresses into peer identifiers for a peer list.
	ids := make([]peer.Identifier, len(addrs))
	for i, addr := range addrs {
		ids[i] = s.Identify(addr)
	}

	x := s.NewClientTransport(t)

	pl := roundrobin.New(x)
	pc := peerbind.Bind(pl, peerbind.BindPeers(ids))
	ob := s.NewUnaryOutbound(x, pc)
	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"service": transport.Outbounds{
				ServiceName: "service",
				Unary:       ob,
			},
		},
	})
	require.NoError(t, d.Start(), "start client dispatcher")
	c := raw.New(d.ClientConfig("service"))
	return d, c
}

// NewServer creates an echo server using the given inbound from any transport.
func (s TransportSpec) NewServer(t *testing.T, addr string) (*yarpc.Dispatcher, string) {
	x := s.NewServerTransport(t, addr)
	ib := s.NewInbound(x, addr)

	d := yarpc.NewDispatcher(yarpc.Config{
		Name:     "service",
		Inbounds: yarpc.Inbounds{ib},
	})

	handle := func(ctx context.Context, req []byte) ([]byte, error) {
		return req, nil
	}

	d.Register(raw.Procedure("echo", handle))
	require.NoError(t, d.Start(), "start server dispatcher")

	return d, s.Addr(x, ib)
}

// TestLosePatienceWithRoundRobin is a test that any transport can apply to
// exercise a transport dropping connections if the transport is stopped before
// a pending request can complete.
func (s TransportSpec) TestLosePatienceWithRoundRobin(t *testing.T) {
	addr := "127.0.0.1:31172"
	var wg sync.WaitGroup

	client, c := s.NewClient(t, []string{addr})

	wg.Add(1)
	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		assert.Error(t, Call(ctx, c))
		wg.Done()
	}()

	time.Sleep(10 * time.Millisecond)
	client.Stop()

	wg.Wait()
}

// TestReuseConnectionWithRoundRobin is a reusable test that any transport can
// apply to cover connection reuse.
func (s TransportSpec) TestReuseConnectionWithRoundRobin(t *testing.T) {
	var wg sync.WaitGroup

	server, addr := s.NewServer(t, ":0")
	defer server.Stop()

	client, c := s.NewClient(t, []string{addr})
	defer client.Stop()

	call := func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		assert.NoError(t, Call(ctx, c))
		wg.Done()
	}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go call()
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// TestBackoffWithRoundRobin is a reusable test that any transport can apply to
// cover connection management backoff.
func (s TransportSpec) TestBackoffWithRoundRobin(t *testing.T) {
	addr := "127.0.0.1:31782"
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		client, c := s.NewClient(t, []string{addr})
		defer client.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		Call(ctx, c)
		wg.Done()
	}()

	// Give the client time to make multiple connection attempts.
	time.Sleep(50 * time.Millisecond)
	server, _ := s.NewServer(t, addr)
	defer server.Stop()

	wg.Wait()
}

// TestReconnectWithRoundRobin is a reusable test that exercises any
// transport's ability to reconnect to a peer if it is temporarily unavailable
// while being retained.
func (s TransportSpec) TestReconnectWithRoundRobin(t *testing.T) {
	server, addr := s.NewServer(t, ":0")
	// server.Stop() is explicit in this test.

	client, c := s.NewClient(t, []string{addr})
	defer client.Stop()

	// Induce a connection
	func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		assert.NoError(t, Call(ctx, c))
	}()

	// Stop the server so a subsequent request must fail
	server.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		CallUntilSuccess(t, c, 10*time.Millisecond)
		wg.Done()
	}()

	// Restart the server so it can reconnect.
	time.Sleep(10 * time.Millisecond)
	restoredServer, _ := s.NewServer(t, addr)
	defer restoredServer.Stop()

	wg.Wait()
}

// Blast sends a blast of calls to the client and verifies that they do not
// err.
func Blast(ctx context.Context, t *testing.T, c raw.Client) {
	for i := 0; i < 10; i++ {
		assert.NoError(t, Call(ctx, c))
	}
}

// CallUntilSuccess sends a request until it succeeds.
func CallUntilSuccess(t *testing.T, c raw.Client, interval time.Duration) {
	for {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, interval)
		if err := Call(ctx, c); err == nil {
			cancel()
			break
		}
		cancel()
	}
}

// Call sends an echo request to the client.
func Call(ctx context.Context, c raw.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	res, err := c.Call(ctx, "echo", []byte("hello"))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(res, []byte("hello")) {
		return fmt.Errorf("unexpected response %+v", res)
	}
	return nil
}
