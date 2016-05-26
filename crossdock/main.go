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

package main

import (
	"github.com/yarpc/yarpc-go/crossdock-go/crossdock"
	"github.com/yarpc/yarpc-go/crossdock/client/echo"
	"github.com/yarpc/yarpc-go/crossdock/client/errors"
	"github.com/yarpc/yarpc-go/crossdock/client/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/client/headers"
	"github.com/yarpc/yarpc-go/crossdock/client/tchclient"
	"github.com/yarpc/yarpc-go/crossdock/client/tchserver"
	"github.com/yarpc/yarpc-go/crossdock/server"
)

func main() {
	server.Start()
	crossdock.Start(dispatch)
}

func dispatch(s crossdock.Sink, behavior string, ps crossdock.Params) {
	switch behavior {
	case "raw":
		echo.Raw(s, ps)
	case "json":
		echo.JSON(s, ps)
	case "thrift":
		echo.Thrift(s, ps)
	case "errors":
		errors.Run(s, ps)
	case "headers":
		headers.Run(s, ps)
	case "tchclient":
		tchclient.Run(s, ps)
	case "tchserver":
		tchserver.Run(s, ps)
	case "thriftgauntlet":
		gauntlet.Run(s, ps)
	default:
		crossdock.Skipf(s, "unknown behavior %q", behavior)
	}
}
