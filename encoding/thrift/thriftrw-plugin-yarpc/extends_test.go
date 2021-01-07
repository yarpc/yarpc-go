// Copyright (c) 2021 Uber Technologies, Inc.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/barclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/barserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/fooclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/fooserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/nameclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/nameserver"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/transport/http"
)

func setupTest(t *testing.T, p []transport.Procedure) (*yarpc.Dispatcher, func()) {
	httpInbound := http.NewTransport().NewInbound("127.0.0.1:0")

	server := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{httpInbound},
	})
	server.Register(p)
	require.NoError(t, server.Start())

	outbound := http.NewTransport().NewSingleOutbound(
		fmt.Sprintf("http://%v", yarpctest.ZeroAddrToHostPort(httpInbound.Addr())))

	client := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary:  outbound,
				Oneway: outbound,
			},
		},
	})
	require.NoError(t, client.Start())

	return client, func() {
		assert.NoError(t, client.Stop())
		assert.NoError(t, server.Stop())
	}
}

func TestExtendsProcedure(t *testing.T) {
	t.Run("base service: Name::name", func(t *testing.T) {
		d, cleanup := setupTest(t, nameserver.New(&nameHandler{}))
		defer cleanup()

		cli := nameclient.New(d.ClientConfig("server"))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		res, err := cli.Name(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Name::name", res)
	})
	t.Run("foo service: Foo::name", func(t *testing.T) {
		d, cleanup := setupTest(t, fooserver.New(&fooHandler{}))
		defer cleanup()

		cli := fooclient.New(d.ClientConfig("server"))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		res, err := cli.Name(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Foo::name", res)
	})
	t.Run("bar service: Bar::name", func(t *testing.T) {
		d, cleanup := setupTest(t, barserver.New(&barHandler{}))
		defer cleanup()

		cli := barclient.New(d.ClientConfig("server"))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		res, err := cli.Name(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Bar::name", res)
	})
}

type nameHandler struct{}

func (*nameHandler) Name(ctx context.Context) (string, error) {
	return yarpc.CallFromContext(ctx).Procedure(), nil
}

type fooHandler struct{}

func (*fooHandler) Name(ctx context.Context) (string, error) {
	return yarpc.CallFromContext(ctx).Procedure(), nil
}

type barHandler struct{}

func (*barHandler) Name(ctx context.Context) (string, error) {
	return yarpc.CallFromContext(ctx).Procedure(), nil
}
