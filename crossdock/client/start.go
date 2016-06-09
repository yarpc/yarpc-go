// Copyright (c) 2016 Uber Technologies, Inc.
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
	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/client/ctxpropagation"
	"github.com/yarpc/yarpc-go/crossdock/client/echo"
	"github.com/yarpc/yarpc-go/crossdock/client/errors"
	"github.com/yarpc/yarpc-go/crossdock/client/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/client/headers"
	"github.com/yarpc/yarpc-go/crossdock/client/outboundttl"
	"github.com/yarpc/yarpc-go/crossdock/client/tchclient"
	"github.com/yarpc/yarpc-go/crossdock/client/tchserver"
)

var behaviors = crossdock.Behaviors{
	"raw":            echo.Raw,
	"json":           echo.JSON,
	"thrift":         echo.Thrift,
	"headers":        headers.Run,
	"errors":         errors.Run,
	"tchclient":      tchclient.Run,
	"tchserver":      tchserver.Run,
	"thriftgauntlet": gauntlet.Run,
	"outboundttl":    outboundttl.Run,
	"ctxpropagation": ctxpropagation.Run,
}

// Start registers behaviors and begins the Crossdock client
func Start() {
	crossdock.Start(behaviors)
}
