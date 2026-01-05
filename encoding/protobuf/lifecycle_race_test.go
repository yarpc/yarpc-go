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

package protobuf_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	grpctransport "go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpcerrors"
)

// reReader wraps a byte slice to provide concurrent-safe re-reading
// without copying data. Each call to newReader() returns a new reader
// positioned at the start.
type reReader struct {
	data []byte
}

func newReReader(r io.Reader) (*reReader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &reReader{data: data}, nil
}

func (rr *reReader) newReader() io.Reader {
	return bytes.NewReader(rr.data)
}

// TestOutboundRequestBodyLifecycle_FanOutAndRetries tests that the request body
// BufferSlice lifecycle is safe in realistic user scenarios:
// 1. Fan-out: Multiple concurrent reads (sending same request to multiple peers)
// 2. Hedging: Concurrent requests racing for the fastest response
// 3. Retries: Sequential re-reading after failures
//
// This verifies that BufferSlice can be safely reread without data races in patterns
// that users commonly employ (retries, fan-out to multiple backends, hedging, etc.).
//
// Run with: go test -race -parallel 20 ./encoding/protobuf -run TestOutboundRequestBodyLifecycle_FanOutAndRetries -v -count 5 -timeout 60s
func TestOutboundRequestBodyLifecycle_FanOutAndRetries(t *testing.T) {
	t.Parallel()

	// Server setup
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverTransport := grpctransport.NewTransport()
	inbound := serverTransport.NewInbound(listener)

	serverDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{inbound},
	})

	serverDispatcher.Register(testpb.BuildTestYARPCProcedures(&echoServer{}))
	require.NoError(t, serverDispatcher.Start())
	defer serverDispatcher.Stop()

	t.Run("fan_out", func(t *testing.T) {
		// Test concurrent fan-out to multiple real server destinations
		// This simulates scenarios where the same request is sent to multiple backends

		// Setup second server
		listener2, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		serverTransport2 := grpctransport.NewTransport()
		inbound2 := serverTransport2.NewInbound(listener2)

		serverDispatcher2 := yarpc.NewDispatcher(yarpc.Config{
			Name:     "server2",
			Inbounds: yarpc.Inbounds{inbound2},
		})

		serverDispatcher2.Register(testpb.BuildTestYARPCProcedures(&echoServer{}))
		require.NoError(t, serverDispatcher2.Start())
		defer serverDispatcher2.Stop()

		// Track which servers received requests
		var server1Calls atomic.Int32
		var server2Calls atomic.Int32

		fanOutMiddleware := middleware.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
			// Create re-reader from original body
			rr, err := newReReader(req.Body)
			if err != nil {
				return nil, err
			}

			// Create two outbounds for fan-out
			clientTransport := grpctransport.NewTransport()
			require.NoError(t, clientTransport.Start())
			defer clientTransport.Stop()

			outbound1 := clientTransport.NewSingleOutbound(listener.Addr().String())
			outbound2 := clientTransport.NewSingleOutbound(listener2.Addr().String())

			require.NoError(t, outbound1.Start())
			defer outbound1.Stop()
			require.NoError(t, outbound2.Start())
			defer outbound2.Stop()

			// Create channels to collect responses
			type result struct {
				resp *transport.Response
				err  error
			}
			const numServers = 2
			resultChan := make(chan result, numServers)

			// Fan out to both servers concurrently
			go func() {
				server1Calls.Add(1)
				// Clone request for server 1
				req1 := &transport.Request{
					Caller:          req.Caller,
					Service:         req.Service,
					Transport:       req.Transport,
					Encoding:        req.Encoding,
					Procedure:       req.Procedure,
					Headers:         req.Headers,
					ShardKey:        req.ShardKey,
					RoutingKey:      req.RoutingKey,
					RoutingDelegate: req.RoutingDelegate,
					Body:            rr.newReader(),
				}
				resp, err := outbound1.Call(ctx, req1)
				resultChan <- result{resp, err}
			}()

			go func() {
				server2Calls.Add(1)
				// Clone request for server 2
				req2 := &transport.Request{
					Caller:          req.Caller,
					Service:         req.Service,
					Transport:       req.Transport,
					Encoding:        req.Encoding,
					Procedure:       req.Procedure,
					Headers:         req.Headers,
					ShardKey:        req.ShardKey,
					RoutingKey:      req.RoutingKey,
					RoutingDelegate: req.RoutingDelegate,
					Body:            rr.newReader(),
				}
				resp, err := outbound2.Call(ctx, req2)
				resultChan <- result{resp, err}
			}()

			// Wait for both responses
			var responses []*transport.Response
			var errors []error
			for i := 0; i < numServers; i++ {
				res := <-resultChan
				if res.err != nil {
					errors = append(errors, res.err)
				} else {
					responses = append(responses, res.resp)
				}
			}

			// Return first successful response, or first error if all failed
			if len(responses) > 0 {
				return responses[0], nil
			}
			if len(errors) > 0 {
				return nil, errors[0]
			}
			return nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "no responses received")
		})

		clientTransport := grpctransport.NewTransport()
		outbound := clientTransport.NewSingleOutbound(listener.Addr().String())

		clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name: "client-fanout",
			Outbounds: yarpc.Outbounds{
				"server": {
					Unary: middleware.ApplyUnaryOutbound(outbound, fanOutMiddleware),
				},
			},
		})
		require.NoError(t, clientDispatcher.Start())
		defer clientDispatcher.Stop()

		client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig("server"))

		// Make 1000 requests, each should fan out to both servers
		const numRequests = 1000
		for i := 0; i < numRequests; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			resp, err := client.Unary(ctx, &testpb.TestMessage{Value: "fanout test"})
			cancel()
			assert.NoError(t, err)
			assert.NotNil(t, resp)
		}

		// Verify both servers received all requests
		assert.Equal(t, int32(numRequests), server1Calls.Load(), "Server 1 should receive all 10 requests")
		assert.Equal(t, int32(numRequests), server2Calls.Load(), "Server 2 should receive all 10 requests")
	})

	t.Run("retries", func(t *testing.T) {
		// Test sequential re-reading of body (retry scenario)
		var retryAttempts atomic.Int32
		var failuresLeft atomic.Int32
		failuresLeft.Store(2) // Fail first 2 attempts

		retryMiddleware := middleware.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
			const maxRetries = 3
			var lastErr error

			// Create re-reader from original body
			rr, err := newReReader(req.Body)
			if err != nil {
				return nil, err
			}

			for attempt := 0; attempt < maxRetries; attempt++ {
				retryAttempts.Add(1)

				// Restore body for this attempt
				req.Body = rr.newReader()

				// Simulate failures for first N attempts
				if failuresLeft.Load() > 0 {
					failuresLeft.Add(-1)
					lastErr = yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "simulated failure")
					continue
				}

				// Try the actual call
				resp, err := next.Call(ctx, req)
				if err == nil {
					return resp, nil
				}
				lastErr = err
			}

			return nil, lastErr
		})

		clientTransport := grpctransport.NewTransport()
		outbound := clientTransport.NewSingleOutbound(listener.Addr().String())

		clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name: "client-retry",
			Outbounds: yarpc.Outbounds{
				"server": {
					Unary: middleware.ApplyUnaryOutbound(outbound, retryMiddleware),
				},
			},
		})
		require.NoError(t, clientDispatcher.Start())
		defer clientDispatcher.Stop()

		client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig("server"))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := client.Unary(ctx, &testpb.TestMessage{Value: "retry test"})
		assert.NoError(t, err, "Should succeed after retries")
		assert.Equal(t, int32(3), retryAttempts.Load(), "Should have made 3 attempts (2 failures + 1 success)")
	})

	t.Run("hedging", func(t *testing.T) {
		// Test concurrent hedged requests racing for fastest response
		var hedgeAttempts atomic.Int32
		const numHedges = 3

		hedgingMiddleware := middleware.UnaryOutboundFunc(func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {

			// Create re-reader from original body
			rr, err := newReReader(req.Body)
			if err != nil {
				return nil, err
			}

			type result struct {
				resp *transport.Response
				err  error
			}
			results := make(chan result, numHedges)

			// Launch hedged requests concurrently
			for i := 0; i < numHedges; i++ {
				go func() {
					hedgeAttempts.Add(1)

					// Each hedge gets its own body reader
					hedgeReq := &transport.Request{
						Caller:          req.Caller,
						Service:         req.Service,
						Transport:       req.Transport,
						Encoding:        req.Encoding,
						Procedure:       req.Procedure,
						Headers:         req.Headers,
						ShardKey:        req.ShardKey,
						RoutingKey:      req.RoutingKey,
						RoutingDelegate: req.RoutingDelegate,
						Body:            rr.newReader(),
					}

					resp, err := next.Call(ctx, hedgeReq)
					results <- result{resp, err}
				}()
			}

			// Take the first successful result
			firstResult := <-results
			return firstResult.resp, firstResult.err
		})

		clientTransport := grpctransport.NewTransport()
		outbound := clientTransport.NewSingleOutbound(listener.Addr().String())

		clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name: "client-hedging",
			Outbounds: yarpc.Outbounds{
				"server": {
					Unary: middleware.ApplyUnaryOutbound(outbound, hedgingMiddleware),
				},
			},
		})
		require.NoError(t, clientDispatcher.Start())
		defer clientDispatcher.Stop()

		client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig("server"))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := client.Unary(ctx, &testpb.TestMessage{Value: "hedging test"})
		assert.NoError(t, err)

		// Give time for all hedges to complete
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(numHedges), hedgeAttempts.Load(), "Mismatched number of hedged requests")
	})
}

// Test helper types
type echoServer struct{}

func (s *echoServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	// Add random delay to make timing non-deterministic and expose potential races
	delay := time.Duration(time.Now().UnixNano()%1000) * time.Microsecond // 0-1ms
	time.Sleep(delay)
	return &testpb.TestMessage{Value: msg.Value}, nil
}

func (s *echoServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}
