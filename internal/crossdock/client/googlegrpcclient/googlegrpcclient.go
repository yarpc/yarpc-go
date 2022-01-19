// Copyright (c) 2022 Uber Technologies, Inc.
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

package googlegrpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/crossdockpb"
	"go.uber.org/yarpc/internal/grpcctx"
	ggrpc "google.golang.org/grpc"
)

var wrap = grpcctx.NewContextWrapper().
	WithCaller("client").
	WithService("yarpc-test").
	WithEncoding("proto").Wrap

// Run tests a grpc-go call to the yarpc server.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.Server)
	fatals.NotEmpty(server, "server is required")

	clientConn, err := ggrpc.Dial(fmt.Sprintf("%s:8089", server), ggrpc.WithInsecure())
	fatals.NoError(err, "grpc.Dial failed")

	client := crossdockpb.NewEchoClient(clientConn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	token := random.String(5)

	pong, err := client.Echo(wrap(ctx), &crossdockpb.Ping{Beep: token})

	crossdock.Fatals(t).NoError(err, "call to Echo::echo failed: %v", err)
	crossdock.Assert(t).Equal(token, pong.Boop, "server said: %v", pong.Boop)
}
