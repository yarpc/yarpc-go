// Copyright (c) 2025 Uber Technologies, Inc.
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

package client

import (
	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc/internal/crossdock/client/apachethrift"
	"go.uber.org/yarpc/internal/crossdock/client/ctxpropagation"
	"go.uber.org/yarpc/internal/crossdock/client/echo"
	"go.uber.org/yarpc/internal/crossdock/client/errorshttpclient"
	"go.uber.org/yarpc/internal/crossdock/client/errorstchclient"
	"go.uber.org/yarpc/internal/crossdock/client/gauntlet"
	"go.uber.org/yarpc/internal/crossdock/client/googlegrpcclient"
	"go.uber.org/yarpc/internal/crossdock/client/googlegrpcserver"
	"go.uber.org/yarpc/internal/crossdock/client/grpc"
	"go.uber.org/yarpc/internal/crossdock/client/headers"
	"go.uber.org/yarpc/internal/crossdock/client/httpserver"
	"go.uber.org/yarpc/internal/crossdock/client/oneway"
	"go.uber.org/yarpc/internal/crossdock/client/onewayctxpropagation"
	"go.uber.org/yarpc/internal/crossdock/client/tchclient"
	"go.uber.org/yarpc/internal/crossdock/client/tchserver"
	"go.uber.org/yarpc/internal/crossdock/client/timeout"
)

var behaviors = crossdock.Behaviors{
	"raw":                   echo.Raw,
	"json":                  echo.JSON,
	"thrift":                echo.Thrift,
	"protobuf":              echo.Protobuf,
	"google_grpc_client":    googlegrpcclient.Run,
	"google_grpc_server":    googlegrpcserver.Run,
	"grpc":                  grpc.Run,
	"headers":               headers.Run,
	"errors_httpclient":     errorshttpclient.Run,
	"errors_tchclient":      errorstchclient.Run,
	"tchclient":             tchclient.Run,
	"tchserver":             tchserver.Run,
	"thriftgauntlet":        gauntlet.Run,
	"timeout":               timeout.Run,
	"ctxpropagation":        ctxpropagation.Run,
	"httpserver":            httpserver.Run,
	"apachethrift":          apachethrift.Run,
	"oneway":                oneway.Run,
	"oneway_ctxpropagation": onewayctxpropagation.Run,
}

// Start registers behaviors and begins the Crossdock client
func Start() {
	crossdock.Start(behaviors)
}
