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

package echoserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/crossdock/thrift/echo"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc"
)

// Interface is the server-side interface for the Echo service.
type Interface interface {
	Echo(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Ping *echo.Ping,
	) (*echo.Pong, yarpc.ResMeta, error)
}

// New prepares an implementation of the Echo service for
// registration.
//
// 	handler := EchoHandler{}
// 	dispatcher.Register(echoserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Registrant {
	h := handler{impl}
	service := thrift.Service{
		Name: "Echo",
		Methods: map[string]thrift.Handler{
			"echo": thrift.HandlerFunc(h.Echo),
		},
	}
	return thrift.BuildRegistrants(service, opts...)
}

type handler struct{ impl Interface }

func (h handler) Echo(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args echo.Echo_Echo_Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.Echo(ctx, reqMeta, args.Ping)

	hadError := err != nil
	result, err := echo.Echo_Echo_Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
