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

package echoserver

import (
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	echot "github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/service/echo"
	"github.com/yarpc/yarpc-go/encoding/thrift"
)

type Interface interface {
	Echo(req *thrift.Request, ping *echot.Ping) (*echot.Pong, *thrift.Response, error)
}

type Handler struct{ impl Interface }

func New(impl Interface) Handler {
	return Handler{impl}
}

func (Handler) Name() string {
	return "Echo"
}

func (Handler) Protocol() protocol.Protocol {
	return protocol.Binary
}

func (h Handler) Handlers() map[string]thrift.Handler {
	return map[string]thrift.Handler{
		"echo": thrift.HandlerFunc(h.Echo),
	}
}

func (h Handler) Echo(req *thrift.Request, body wire.Value) (wire.Value, *thrift.Response, error) {
	var args echo.EchoArgs
	if err := args.FromWire(body); err != nil {
		return wire.Value{}, nil, err
	}

	success, res, err := h.impl.Echo(req, args.Ping)
	result, err := echo.EchoHelper.WrapResponse(success, err)
	return result.ToWire(), res, err
}
