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
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpchttp"
	commonpb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/common"
	keyvaluepb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/keyvalue"
	streampb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/stream"
	keyvalue "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/src/keyvalue"
	stream "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/src/stream"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

func newKeyValueClient(t *testing.T) keyvaluepb.StoreYARPCClient {
	dialer := &yarpchttp.Dialer{}
	require.NoError(t, dialer.Start(context.Background()))

	outbound := &yarpchttp.Outbound{
		URL:    &url.URL{Scheme: "http", Host: "127.0.0.1:8888"},
		Dialer: dialer,
	}

	return keyvaluepb.NewStoreYARPCClient(yarpc.Client{
		Caller:  "test",
		Service: "keyvalue",
		Unary:   outbound,
	})
}

func newHelloClient(t *testing.T) streampb.HelloYARPCClient {
	return streampb.NewHelloYARPCClient(yarpc.Client{
		Caller:  "test",
		Service: "hello",
	})
}

func startProcedures(t *testing.T, service string, procedures []yarpc.Procedure) (stop func() error) {
	router := yarpcrouter.NewMapRouter(service)
	router.Register(procedures)

	inbound := &yarpchttp.Inbound{
		Addr:   ":8888",
		Router: router,
	}
	require.NoError(t, inbound.Start(context.Background()))

	return func() error { return inbound.Stop(context.Background()) }
}

func TestIntegration(t *testing.T) {
	t.Run("simple unary exchange", func(t *testing.T) {
		procedures := keyvaluepb.BuildStoreYARPCProcedures(keyvalue.NewServer())
		assert.Equal(t, 4, len(procedures))

		stop := startProcedures(t, "keyvalue", procedures)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		kv := newKeyValueClient(t)
		_, err := kv.Get(ctx, &commonpb.GetRequest{Key: "notfound"})
		assert.Contains(t, err.Error(), `failed to find value for key: "notfound"`)

		_, err = kv.Set(ctx, &commonpb.SetRequest{Key: "foo", Value: "bar"})
		assert.NoError(t, err)

		res, err := kv.Get(ctx, &commonpb.GetRequest{Key: "foo"})
		assert.NoError(t, err)
		assert.Equal(t, "bar", res.GetValue())
	})
	t.Run("bidirectional streaming", func(t *testing.T) {
		t.Skip("TODO(mensch): Use the gRPC transport when available")
		procedures := streampb.BuildHelloYARPCProcedures(stream.NewServer())
		assert.Equal(t, 8, len(procedures))

		stop := startProcedures(t, "hello", procedures)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		hello := newHelloClient(t)
		stream, err := hello.Bidirectional(ctx)
		require.NoError(t, err)

		assert.NoError(t, stream.Send(&streampb.HelloRequest{Greeting: "Greetings!"}))

		res, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, `Received "Greetings!"`, res.Response)

		assert.NoError(t, stream.Send(&streampb.HelloRequest{Greeting: "exit"}))

		res, err = stream.Recv()
		require.NoError(t, err)
		assert.Nil(t, res)
	})
}
