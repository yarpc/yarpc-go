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

package testing

import (
	"io"
	"net"
	"runtime"
	"strings"
	"testing"

	"runtime/debug"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/prototest/example"
	"go.uber.org/yarpc/internal/prototest/examplepb"
	"go.uber.org/yarpc/internal/prototest/exampleutil"
	"go.uber.org/yarpc/internal/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))
	// Disable GC for benchmarks
	debug.SetGCPercent(-1)
}

func BenchmarkIntegrationYARPC(b *testing.B) {
	for _, transportType := range testutils.AllTransportTypes {
		b.Run(transportType.String(), func(b *testing.B) {
			benchmarkForTransportType(b, transportType, func(clients *exampleutil.Clients) error {
				benchmarkIntegrationYARPC(b, clients.KeyValueYARPCClient)
				return nil
			})
		})
	}
}

func BenchmarkIntegrationGRPCClient(b *testing.B) {
	benchmarkForTransportType(b, testutils.TransportTypeGRPC, func(clients *exampleutil.Clients) error {
		benchmarkIntegrationGRPC(b, clients.KeyValueGRPCClient, clients.ContextWrapper)
		return nil
	})
}

func BenchmarkIntegrationGRPCAll(b *testing.B) {
	server := grpc.NewServer()
	examplepb.RegisterKeyValueServer(server, example.NewKeyValueYARPCServer())
	listener, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		b.Fatal(err.Error())
	}
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	//lint:ignore SA1019 grpc.Dial is deprecated
	grpcClientConn, err := grpc.Dial("0.0.0.0:1234", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err.Error())
	}
	benchmarkIntegrationGRPC(b, examplepb.NewKeyValueClient(grpcClientConn), grpcctx.NewContextWrapper())
}

func BenchmarkIntegrationGRPCMemorypool(b *testing.B) {
	server := grpc.NewServer()
	examplepb.RegisterKeyValueServer(server, example.NewKeyValueYARPCServer())
	listener, err := net.Listen("tcp", "0.0.0.0:1236")
	if err != nil {
		b.Fatal(err.Error())
	}
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	//lint:ignore SA1019 grpc.Dial is deprecated
	grpcClientConn, err := grpc.Dial("0.0.0.0:1236", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err.Error())
	}
	client := examplepb.NewKeyValueClient(grpcClientConn)
	contextWrapper := grpcctx.NewContextWrapper()

	// Test payloads
	payloads := []struct {
		name  string
		value string
	}{
		{"Small", strings.Repeat("a", 350)},
		{"Large", strings.Repeat("a", 1024*1024)}, // 1MB
	}

	// Force GC before starting
	runtime.GC()
	// Scenario 1: Small payloads only
	b.Run("SmallOnly", func(b *testing.B) {
		runMixedBurst(b, client, contextWrapper, 100, "small_only", "small_only", payloads)
	})

	runtime.GC()
	// Scenario 2: Large payloads only
	b.Run("LargeOnly", func(b *testing.B) {
		runMixedBurst(b, client, contextWrapper, 0, "large_only", "large_only", payloads)
	})

	runtime.GC()
	// Scenario 3: 90% small, 10% large
	b.Run("Mixed90_10", func(b *testing.B) {
		runMixedBurst(b, client, contextWrapper, 90, "90_10_small", "90_10_large", payloads)
	})

	runtime.GC()
	// Scenario 4: 99% small, 1% large
	b.Run("Mixed99_1", func(b *testing.B) {
		runMixedBurst(b, client, contextWrapper, 99, "99_1_small", "99_1_large", payloads)
	})

}

func runMixedBurst(b *testing.B, client examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper, smallPayloadPercent int, smallPayloadKey, largePayloadKey string, payloads []struct {
	name  string
	value string
}) {
	// Set up both values
	if err := setValueGRPC(client, contextWrapper, smallPayloadKey, payloads[0].value); err != nil {
		b.Fatal(err)
	}
	if err := setValueGRPC(client, contextWrapper, largePayloadKey, payloads[1].value); err != nil {
		b.Fatal(err)
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	beforeAlloc := stats.TotalAlloc
	beforeSys := stats.Sys
	beforeMallocs := stats.Mallocs
	beforeFrees := stats.Frees

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%100 < smallPayloadPercent {
			if _, err := getValueGRPC(client, contextWrapper, smallPayloadKey); err != nil {
				b.Fatal(err)
			}
		} else {
			if _, err := getValueGRPC(client, contextWrapper, largePayloadKey); err != nil {
				b.Fatal(err)
			}
		}
	}

	runtime.ReadMemStats(&stats)
	b.ReportMetric(float64(stats.TotalAlloc-beforeAlloc)/float64(b.N), "B/op_total")
	b.ReportMetric(float64(stats.Sys-beforeSys)/float64(b.N), "B/op_sys")
	b.ReportMetric(float64(stats.Mallocs-beforeMallocs)/float64(b.N), "mallocs/op")
	b.ReportMetric(float64(stats.Frees-beforeFrees)/float64(b.N), "frees/op")

}

func benchmarkForTransportType(b *testing.B, transportType testutils.TransportType, f func(*exampleutil.Clients) error) {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	fooYARPCServer := example.NewFooYARPCServer(transport.NewHeaders())
	exampleutil.WithClients(transportType, keyValueYARPCServer, fooYARPCServer, nil, f)
}

func benchmarkIntegrationYARPC(b *testing.B, keyValueYARPCClient examplepb.KeyValueYARPCClient) {
	b.Run("Get", func(b *testing.B) {
		setValue(keyValueYARPCClient, "foo", "bar")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getValue(keyValueYARPCClient, "foo")
		}
	})
}

func benchmarkIntegrationGRPC(b *testing.B, keyValueClient examplepb.KeyValueClient, contextWrapper *grpcctx.ContextWrapper) {
	b.Run("Get", func(b *testing.B) {
		setValueGRPC(keyValueClient, contextWrapper, "foo", "bar")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getValueGRPC(keyValueClient, contextWrapper, "foo")
		}
	})
}
