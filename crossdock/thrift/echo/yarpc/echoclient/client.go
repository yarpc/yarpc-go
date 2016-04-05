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

package echoclient

import (
	"github.com/thriftrw/thriftrw-go/protocol"
	echot "github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/service/echo"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
)

type Interface interface {
	Echo(req *thrift.Request, ping *echot.Ping) (*echot.Pong, *thrift.Response, error)
}

type client struct {
	client thrift.Client
}

func New(c transport.Channel) Interface {
	return client{client: thrift.New(thrift.Config{
		Service:  "Echo",
		Channel:  c,
		Protocol: protocol.Binary,
	})}
}

func (c client) Echo(req *thrift.Request, ping *echot.Ping) (*echot.Pong, *thrift.Response, error) {
	args := echo.EchoHelper.Args(ping)
	resBody, res, err := c.client.Call("echo", req, args.ToWire())
	if err != nil {
		return nil, res, err
	}

	var result echo.EchoResult
	if err := result.FromWire(resBody); err != nil {
		return nil, res, err
	}

	success, err := echo.EchoHelper.UnwrapResponse(&result)
	return success, res, err
}
