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
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcclient"
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/readonlystorefx"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/readonlystoreserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/storeclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/storefx"
)

func TestFxClient(t *testing.T) {
	d := yarpcclient.NewProvider()
	d.Register("store", yarpc.Client{})

	assert.NotPanics(t, func() {
		p := storefx.Params{
			Provider: d,
		}
		f := storefx.Client("store").(func(storefx.Params) (storefx.Result, error))
		f(p)
	}, "failed to build client")

	assert.Panics(t, func() {
		f := storefx.Client("not-store").(func(yarpc.ClientProvider) storeclient.Interface)
		f(d)
	}, "expected panic")
}

func extractProcedures(procs *[]yarpc.Procedure) fx.Option {
	type params struct {
		fx.In

		// We need to handle both cases: A single transport.Procedure provided
		// to the "yarpcfx" group and a []transport.Procedure provided to the
		// "yarpcfx" group.
		SingleProcedures []yarpc.Procedure   `group:"yarpcfx"`
		ProcedureLists   [][]yarpc.Procedure `group:"yarpcfx"`
	}

	return fx.Invoke(func(p params) {
		for _, proc := range p.SingleProcedures {
			*procs = append(*procs, proc)
		}
		for _, procList := range p.ProcedureLists {
			*procs = append(*procs, procList...)
		}
	})
}

func echoJSON(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	return req, nil
}

func TestFxServer(t *testing.T) {
	type jsonProcedures struct {
		fx.Out

		Procedures []yarpc.Procedure `group:"yarpcfx"`
	}

	handler := readOnlyStoreHandler{
		"foo":    1,
		"bar":    2,
		"answer": 42,
	}

	var procedures []yarpc.Procedure
	serverApp := fxtest.New(t,
		fx.Provide(
			func() readonlystoreserver.Interface { return handler },
			readonlystorefx.Server(),
			func() jsonProcedures {
				return jsonProcedures{Procedures: yarpcjson.Procedure("echoJSON", echoJSON)}
			},
		),
		extractProcedures(&procedures),
	)
	defer serverApp.RequireStart().RequireStop()

	router := yarpcrouter.NewMapRouter("myserver")
	router.Register(procedures)
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	serverD := &yarpchttp.Inbound{
		Listener: listener,
		Router:   router,
	}
	require.NoError(t, serverD.Start(context.Background()), "failed to start server")
	defer func() {
		assert.NoError(t, serverD.Stop(context.Background()), "failed to stop server")
	}()

	dialer := &yarpchttp.Dialer{}
	require.NoError(t, dialer.Start(context.Background()))

	outbound := &yarpchttp.Outbound{
		URL:    &url.URL{Scheme: "http", Host: listener.Addr().String()},
		Dialer: dialer,
	}

	clientD := yarpc.Client{
		Caller:  "myclient",
		Service: "myserver",
		Unary:   outbound,
	}

	// Can use read-write client to call read-only server
	client := storeclient.New(clientD)

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
		require.True(t, ok, "error '%+v' must be a *KeyDoesNotExist, not %T", err, err)
		assert.Equal(t, "baz", *exc.Key, "exception key did not match")
	})

	// rawClient := raw.New(clientD.ClientConfig("myserver"))
	jsonClient := yarpcjson.New(clientD)
	t.Run("json", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		req := map[string]string{"hello": "world"}
		var res map[string]string
		err := jsonClient.Call(ctx, "echoJSON", req, &res)
		require.NoError(t, err, "request failed")
		assert.Equal(t, map[string]string{"hello": "world"}, res, "response body did not match")
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
