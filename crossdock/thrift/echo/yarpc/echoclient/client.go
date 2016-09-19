// Code generated by thriftrw-plugin-yarpc
// @generated

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
	"golang.org/x/net/context"
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	echo2 "github.com/yarpc/yarpc-go/crossdock/thrift/echo/service/echo"
	"github.com/thriftrw/thriftrw-go/wire"
)

// Interface is a client for the Echo service.
type Interface interface {
	Echo(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Ping *echo.Ping,
	) (*echo.Pong, yarpc.CallResMeta, error)
}

// New builds a new client for the Echo service.
//
// 	client := echoclient.New(dispatcher.Channel("echo"))
func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{
		Service:  "Echo",
		Channel:  c,
		Protocol: protocol.Binary,
	}, opts...)}
}

type client struct{ c thrift.Client }

func (c client) Echo(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Ping *echo.Ping,
) (success *echo.Pong, resMeta yarpc.CallResMeta, err error) {
	args := echo2.EchoHelper.Args(_Ping)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result echo2.EchoResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = echo2.EchoHelper.UnwrapResponse(&result)
	return
}
