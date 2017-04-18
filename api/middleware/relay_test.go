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

package middleware_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelay exercises relay middleware by setting up a service and a proxy for
// that service.
// The test sends a requests from the service to the proxy.
// The proxy handles the first request and forwards the second request back to
// the originating service.
func TestRelay(t *testing.T) {

	s := setupService()
	s.Register(json.Procedure("echo", (&handler{"self"}).handle))
	require.NoError(t, s.Start(), "service should start")
	defer s.Stop()

	p := setupProxy()
	require.NoError(t, p.Start(), "proxy should start")
	s.Register(json.Procedure("proxy-echo", (&handler{"proxy"}).handle))
	defer p.Stop()

	client := json.New(s.ClientConfig("service"))
	var res body
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := client.Call(ctx, "proxy-echo", &body{"Hello, World!"}, &res)
	require.NoError(t, err, "call to proxy without error")
	assert.Equal(t, res, body{"proxy: Hello, World!"}, "receive echo response through proxy")

	err = client.Call(ctx, "echo", &body{"Hello, World!"}, &res)
	require.NoError(t, err, "call through proxy without error")
	assert.Equal(t, res, body{"self: Hello, World!"}, "receive echo response through proxy")
}

// Listens on :30000
func setupService() *yarpc.Dispatcher {
	t := http.NewTransport()
	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "service",
		Inbounds: yarpc.Inbounds{
			t.NewInbound(":30000"),
		},
		Outbounds: yarpc.Outbounds{
			"service": transport.Outbounds{
				Unary: t.NewSingleOutbound("http://127.0.0.1:30001"),
			},
		},
	})
	return d
}

// Listens on :30001, forwards all requests to :30000
func setupProxy() *yarpc.Dispatcher {
	t := http.NewTransport()
	m := middleware.NewRelayMiddleware()
	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "relay",
		Inbounds: yarpc.Inbounds{
			t.NewInbound(":30001"),
		},
		Outbounds: yarpc.Outbounds{
			"forward": transport.Outbounds{
				Unary: t.NewSingleOutbound("http://127.0.0.1:30000"),
			},
		},
		RouterMiddleware: m,
	})
	m.SetClientConfig(d.ClientConfig("forward"))
	return d
}

type body struct {
	Message string `json:"message"`
}

type handler struct {
	responder string
}

func (h *handler) handle(ctx context.Context, req *body) (*body, error) {
	return &body{h.responder + ": " + req.Message}, nil
}
