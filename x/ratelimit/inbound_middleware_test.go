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

package ratelimit_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/x/ratelimit"
)

func TestRateLimiterMiddleware(t *testing.T) {
	middleware, err := ratelimit.NewUnaryInboundMiddleware(1, ratelimit.WithoutSlack)
	require.NoError(t, err)
	serverTransport := http.NewTransport()
	serverInbound := serverTransport.NewInbound("127.0.0.1:0")
	serverDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "service",
		Inbounds: yarpc.Inbounds{serverInbound},
		InboundMiddleware: yarpc.InboundMiddleware{
			Unary: middleware,
		},
	})
	integrationtest.Register(serverDispatcher)
	require.NoError(t, serverDispatcher.Start())
	defer serverDispatcher.Stop()
	inboundAddr := serverInbound.Addr()

	outboundAddr := fmt.Sprintf("http://%s/", inboundAddr)
	clientTransport := http.NewTransport()
	clientOutbound := clientTransport.NewSingleOutbound(outboundAddr)
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"service": transport.Outbounds{
				Unary: clientOutbound,
			},
		},
	})
	require.NoError(t, clientDispatcher.Start())
	defer clientDispatcher.Stop()
	rawClient := raw.New(clientDispatcher.ClientConfig("service"))

	assert.NoError(t, integrationtest.Call(context.Background(), rawClient))
	err = integrationtest.Call(context.Background(), rawClient)
	assert.Error(t, err, "rate limit exceeded")
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestInvalidRateLimiterMiddleware(t *testing.T) {
	_, err := ratelimit.NewUnaryInboundMiddleware(-1)
	assert.Error(t, err)
}
