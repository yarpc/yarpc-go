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

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	yarpcgrpc "go.uber.org/yarpc/v2/yarpcgrpc"
	"go.uber.org/yarpc/v2/yarpchttp"
	commonpb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/common"
	keyvaluepb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/keyvalue"
	streampb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/stream"
	keyvalue "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/src/keyvalue"
	stream "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/src/stream"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

type testInbound interface {
	Start(context.Context) error
	Stop(context.Context) error
}

func newKeyValueClient(t *testing.T, address string) keyvaluepb.StoreYARPCClient {
	dialer := &yarpchttp.Dialer{}
	require.NoError(t, dialer.Start(context.Background()))

	outbound := &yarpchttp.Outbound{
		URL:    &url.URL{Scheme: "http", Host: address},
		Dialer: dialer,
	}

	return keyvaluepb.NewStoreYARPCClient(yarpc.Client{
		Caller:  "test",
		Service: "keyvalue",
		Unary:   outbound,
	})
}

func newHelloClient(t *testing.T, address string) streampb.HelloYARPCClient {
	dialer := &yarpcgrpc.Dialer{}
	require.NoError(t, dialer.Start(context.Background()))

	outbound := &yarpcgrpc.Outbound{
		URL:    &url.URL{Scheme: "http", Host: address},
		Dialer: dialer,
	}
	return streampb.NewHelloYARPCClient(yarpc.Client{
		Caller:  "test",
		Service: "hello",
		Stream:  outbound,
	})
}

func startInbounds(t *testing.T, transport, service string, procedures []yarpc.TransportProcedure) (address string, stop func()) {
	router := yarpcrouter.NewMapRouter(service, procedures)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	var inbound testInbound
	switch transport {
	case "http":
		inbound = &yarpchttp.Inbound{
			Listener: listener,
			Router:   router,
		}
	case "grpc":
		inbound = &yarpcgrpc.Inbound{
			Listener: listener,
			Router:   router,
		}
	default:
		t.Fatalf("unsupported transport: %v", transport)
	}
	require.NoError(t, inbound.Start(context.Background()))

	return listener.Addr().String(), func() { require.NoError(t, inbound.Stop(context.Background())) }
}
func setupStreamingEnv(t *testing.T) (client streampb.HelloYARPCClient, stop func()) {
	procedures := streampb.BuildHelloYARPCProcedures(stream.NewServer())
	assert.Equal(t, 6, len(procedures))

	addr, stop := startInbounds(t, "grpc", "hello", procedures)
	return newHelloClient(t, addr), stop
}

func TestIntegration(t *testing.T) {
	t.Run("simple unary exchange", func(t *testing.T) {
		procedures := keyvaluepb.BuildStoreYARPCProcedures(keyvalue.NewServer())
		assert.Equal(t, 4, len(procedures))

		addr, stop := startInbounds(t, "http", "keyvalue", procedures)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		kv := newKeyValueClient(t, addr)
		_, err := kv.Get(ctx, &commonpb.GetRequest{Key: "notfound"})
		assert.Contains(t, err.Error(), `failed to find value for key: "notfound"`)

		_, err = kv.Set(ctx, &commonpb.SetRequest{Key: "foo", Value: "bar"})
		assert.NoError(t, err)

		res, err := kv.Get(ctx, &commonpb.GetRequest{Key: "foo"})
		assert.NoError(t, err)
		assert.Equal(t, "bar", res.GetValue())
	})
	t.Run("client streaming", func(t *testing.T) {
		hello, stop := setupStreamingEnv(t)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		stream, err := hello.Out(ctx)
		require.NoError(t, err)

		for i := 0; i < 4; i++ {
			require.NoError(t, stream.Send(&streampb.HelloRequest{Greeting: fmt.Sprintf("%d", i)}))
		}

		res, err := stream.CloseAndRecv()
		assert.NoError(t, err)
		assert.Equal(t, "Received 0,1,2,3", res.GetResponse())
	})
	t.Run("server streaming", func(t *testing.T) {
		hello, stop := setupStreamingEnv(t)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		stream, err := hello.In(ctx, &streampb.HelloRequest{Greeting: "four"})
		require.NoError(t, err)

		for i := 0; i < 4; i++ {
			res, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf(`Received %d`, i), res.GetResponse())
		}

		_, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
	})
	t.Run("bidirectional streaming", func(t *testing.T) {
		hello, stop := setupStreamingEnv(t)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		stream, err := hello.Bidirectional(ctx)
		require.NoError(t, err)

		assert.NoError(t, stream.Send(&streampb.HelloRequest{Greeting: "Greetings!"}))

		res, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, `Received "Greetings!"`, res.GetResponse())

		assert.NoError(t, stream.Send(&streampb.HelloRequest{Greeting: "exit"}))

		_, err = stream.Recv()
		assert.Equal(t, err, io.EOF)
	})
}
