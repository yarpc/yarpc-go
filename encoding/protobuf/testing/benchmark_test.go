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
	"testing"

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
