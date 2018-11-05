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

package yarpctchannel

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

func TestDirectAddress(t *testing.T) {
	type body struct {
		Message string
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := yarpc.Address(listener.Addr().String())

	procs, err := yarpc.EncodingToTransportProcedures(yarpcjson.Procedure("echo", func(ctx context.Context, body *body) (*body, error) {
		return body, nil
	}))
	require.NoError(t, err)
	server := &Inbound{
		Service:  "server",
		Listener: listener,
		Router:   yarpcrouter.NewMapRouter("server", procs),
	}
	require.NoError(t, server.Start(ctx))
	defer server.Stop(ctx)

	dialer := &Dialer{
		Caller: "client",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)

	client := yarpcjson.New(yarpc.Client{
		Caller:  "client",
		Service: "server",
		Unary: &Outbound{
			Dialer: dialer,
		},
	})
	var res body
	var retAddr yarpc.Identifier
	require.NoError(t, client.Call(ctx, "echo", &body{Message: "hello"}, &res, yarpc.To(addr), yarpc.ResponseFrom(&retAddr)))
	assert.NotNil(t, retAddr)
	assert.Equal(t, addr, retAddr)
}
