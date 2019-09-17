// Copyright (c) 2019 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoreclient"
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

func extractProcedures(procs *[]transport.Procedure) fx.Option {
	type params struct {
		fx.In

		// We need to handle both cases: A single transport.Procedure provided
		// to the "yarpcfx" group and a []transport.Procedure provided to the
		// "yarpcfx" group.
		SingleProcedures []transport.Procedure   `group:"yarpcfx"`
		ProcedureLists   [][]transport.Procedure `group:"yarpcfx"`
	}

	return fx.Invoke(func(p params) {
		*procs = append(*procs, p.SingleProcedures...)
		for _, procList := range p.ProcedureLists {
			*procs = append(*procs, procList...)
		}
	})
}

func extractThriftProcedures(procs *[]transport.Procedure) fx.Option {
	type params struct {
		fx.In

		Procedures [][]transport.Procedure `group:"thrift"`
	}

	return fx.Invoke(func(p params) {
		for _, procList := range p.Procedures {
			*procs = append(*procs, procList...)
		}
	})
}

func echoRaw(ctx context.Context, req []byte) ([]byte, error) { return req, nil }

func TestFxServer(t *testing.T) {
	type rawProcedures struct {
		fx.Out

		Procedures []transport.Procedure `group:"yarpcfx"`
	}

	handler := readOnlyStoreHandler{
		"foo":    1,
		"bar":    2,
		"answer": 42,
	}

	var (
		procedures []transport.Procedure
		// the 'thrift' value group is unused in YARPC, so we only test that the
		// procedures are provided correctly
		thriftProcedures []transport.Procedure
	)

	serverApp := fxtest.New(t,
		fx.Provide(
			func() readonlystoreserver.Interface { return handler },
			readonlystorefx.Server(),
			func() rawProcedures {
				return rawProcedures{Procedures: raw.Procedure("echoRaw", echoRaw)}
			},
		),
		extractProcedures(&procedures),
		extractThriftProcedures(&thriftProcedures),
	)
	defer serverApp.RequireStart().RequireStop()

	inbound := http.NewTransport().NewInbound("127.0.0.1:0")
	serverD := yarpc.NewDispatcher(yarpc.Config{
		Name:     "myserver",
		Inbounds: yarpc.Inbounds{inbound},
	})

	// ensure we provide the same procedures into the container under different
	// value groups: `yarpcfx` and `thrift`.
	require.NotNil(t, thriftProcedures, "thrift procedures were not added to 'thrift' value group")
	assert.Len(t, thriftProcedures, len(procedures)-1) // subtract one for the "raw" procedure we manually added

	serverD.Register(procedures)
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

	client := readonlystoreclient.New(clientD.ClientConfig("myserver"))

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

	rawClient := raw.New(clientD.ClientConfig("myserver"))

	t.Run("raw", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		res, err := rawClient.Call(ctx, "echoRaw", []byte("hello"))
		require.NoError(t, err, "request failed")
		assert.Equal(t, "hello", string(res), "response body did not match")
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
