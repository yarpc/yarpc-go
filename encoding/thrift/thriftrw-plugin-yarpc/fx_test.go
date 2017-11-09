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

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystorefx"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoreserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storeclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storefx"
	"go.uber.org/yarpc/transport/http"
)

func TestFxClient(t *testing.T) {
	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "myservice",
		Outbounds: yarpc.Outbounds{
			"store": {Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1/yarpc")},
		},
	})

	assert.NotPanics(t, func() {
		p := storefx.Params{
			Provider: d,
		}
		f := storefx.Client("store").(func(storefx.Params) storefx.Result)
		f(p)
	}, "failed to build client")

	assert.Panics(t, func() {
		f := storefx.Client("not-store").(func(*yarpc.Dispatcher) storeclient.Interface)
		f(d)
	}, "expected panic")
}

func TestFxServer(t *testing.T) {
	handler := readOnlyStoreHandler{
		"foo":    1,
		"bar":    2,
		"answer": 42,
	}

	var out struct {
		Procedures []transport.Procedure `group:"yarpcfx"`
	}
	serverApp := fxtest.New(t,
		fx.Provide(
			func() readonlystoreserver.Interface { return handler },
			readonlystorefx.Server(),
		),
		fx.Extract(&out),
	)
	defer serverApp.RequireStart().RequireStop()

	inbound := http.NewTransport().NewInbound(":0")
	serverD := yarpc.NewDispatcher(yarpc.Config{
		Name:     "myserver",
		Inbounds: yarpc.Inbounds{inbound},
	})
	serverD.Register(out.Procedures)
	require.NoError(t, serverD.Start(), "failed to start server")
	defer func() {
		assert.NoError(t, serverD.Stop(), "failed to stop server")
	}()

	clientD := yarpc.NewDispatcher(yarpc.Config{
		Name: "myclient",
		Outbounds: yarpc.Outbounds{
			"myserver": {
				Unary: http.NewTransport().NewSingleOutbound(
					fmt.Sprintf("http://%s/", inbound.Addr()),
				),
			},
		},
	})
	require.NoError(t, clientD.Start(), "failed to start client")
	defer func() {
		assert.NoError(t, clientD.Stop(), "failed to stop client")
	}()

	// Can use read-write client to call read-only server
	client := storeclient.New(clientD.ClientConfig("myserver"))

	ctx := context.Background()

	t.Run("Integer", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		res, err := client.Integer(ctx, ptr.String("answer"))
		assert.NoError(t, err, "request failed")
		assert.Equal(t, int64(42), res, "result did not match")
	})

	t.Run("Integer error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err := client.Integer(ctx, ptr.String("baz")) // baz does not exist
		assert.Error(t, err, "request failed")

		exc, ok := err.(*atomic.KeyDoesNotExist)
		require.True(t, ok, "error must be a *KeyDoesNotExist, not %T", err)
		assert.Equal(t, "baz", *exc.Key, "exception key did not match")
	})
}

type readOnlyStoreHandler map[string]int64

func (readOnlyStoreHandler) Healthy(context.Context) (bool, error) {
	return true, nil
}

func (h readOnlyStoreHandler) Integer(ctx context.Context, k *string) (int64, error) {
	v, ok := h[*k]
	if !ok {
		return 0, &atomic.KeyDoesNotExist{Key: k}
	}
	return v, nil
}
