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

package protobuf_test

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/transport/grpc"
)

// BenchmarkGRPCCodec_RoundTrip measures the performance of the codec registry
// system with different message sizes under realistic gRPC round-trip scenarios.
//
// Run with:
//
//	go test -bench=BenchmarkGRPCCodec_RoundTrip -benchmem ./encoding/protobuf
func BenchmarkGRPCCodec_RoundTrip(b *testing.B) {
	scenarios := []struct {
		name    string
		message *testpb.TestMessage
	}{
		{
			name: "Small_350B",
			message: &testpb.TestMessage{
				Value: strings.Repeat("x", 300), // ~350 bytes serialized
			},
		},
		{
			name: "Medium_10KB",
			message: &testpb.TestMessage{
				Value: strings.Repeat("x", 10*1024), // ~10KB
			},
		},
		{
			name: "Large_1MB",
			message: &testpb.TestMessage{
				Value: strings.Repeat("x", 1024*1024), // ~1MB
			},
		},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			runCodecBenchmark(b, sc.message)
		})
	}
}

func runCodecBenchmark(b *testing.B, message *testpb.TestMessage) {
	// Create server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	serverTransport := grpc.NewTransport()
	inbound := serverTransport.NewInbound(listener)

	serverDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "benchmark-server",
		Inbounds: yarpc.Inbounds{inbound},
	})
	serverDispatcher.Register(testpb.BuildTestYARPCProcedures(&benchEchoServer{}))

	if err := serverDispatcher.Start(); err != nil {
		b.Fatalf("failed to start server: %v", err)
	}
	defer serverDispatcher.Stop()

	serverAddr := listener.Addr().String()

	// Create client
	clientTransport := grpc.NewTransport()
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "benchmark-client",
		Outbounds: yarpc.Outbounds{
			"benchmark-server": {
				Unary: clientTransport.NewSingleOutbound(serverAddr),
			},
		},
	})

	if err := clientDispatcher.Start(); err != nil {
		b.Fatalf("failed to start client: %v", err)
	}
	defer clientDispatcher.Stop()

	client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig("benchmark-server"))

	// Reset timer after setup
	b.ResetTimer()
	b.ReportAllocs()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		response, err := client.Unary(ctx, message)
		cancel()

		if err != nil {
			b.Fatalf("request %d failed: %v", i+1, err)
		}

		// Verify response matches request
		if response.Value != message.Value {
			b.Fatalf("response mismatch at request %d", i+1)
		}
	}
}

// benchEchoServer is a simple echo server for benchmarking
type benchEchoServer struct{}

func (s *benchEchoServer) Unary(ctx context.Context, req *testpb.TestMessage) (*testpb.TestMessage, error) {
	return req, nil
}

func (s *benchEchoServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
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
