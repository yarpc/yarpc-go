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
	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/echo"
	"github.com/yarpc/yarpc-go/crossdock/behavior/errors"
	"github.com/yarpc/yarpc-go/crossdock/behavior/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/behavior/headers"
	"github.com/yarpc/yarpc-go/crossdock/behavior/tchclient"
	"github.com/yarpc/yarpc-go/crossdock/behavior/tchserver"
	"github.com/yarpc/yarpc-go/crossdock/server"
)

func main() {
	server.Start()
	crossdock.Start(dispatch)
}

func dispatch(t crossdock.T, behavior string, ps crossdock.Params) {
	switch behavior {
	case "raw":
		echo.Raw(t, ps)
	case "json":
		echo.JSON(t, ps)
	case "thrift":
		echo.Thrift(t, ps)
	case "errors":
		errors.Run(t, ps)
	case "headers":
		headers.Run(t, ps)
	case "tchclient":
		tchclient.Run(t, ps)
	case "tchserver":
		tchserver.Run(t, ps)
	case "thriftgauntlet":
		gauntlet.Run(t, ps)
	default:
		crossdock.Skipf(t, "unknown behavior %q", behavior)
	}
}
