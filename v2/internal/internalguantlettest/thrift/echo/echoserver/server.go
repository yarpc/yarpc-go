// Code generated by thriftrw-plugin-yarpc
// @generated

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

package echoserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalguantlettest/thrift/echo"
	"go.uber.org/yarpc/v2/yarpcthrift"
)

// Interface is the server-side interface for the Echo service.
type Interface interface {
	Echo(
		ctx context.Context,
		Request *echo.EchoRequest,
	) (*echo.EchoResponse, error)
}

// New prepares an implementation of the Echo service for
// registration.
//
// 	handler := EchoHandler{}
// 	dispatcher.Register(echoserver.New(handler))
func New(impl Interface, opts ...yarpcthrift.RegisterOption) []yarpc.TransportProcedure {
	h := handler{impl}
	service := yarpcthrift.Service{
		Name: "Echo",
		Methods: []yarpcthrift.Method{

			yarpcthrift.Method{
				Name:         "Echo",
				Handler:      yarpcthrift.Handler(h.Echo),
				Signature:    "Echo(Request *echo.EchoRequest) (*echo.EchoResponse)",
				ThriftModule: echo.ThriftModule,
			},
		},
	}

	procedures := make([]yarpc.TransportProcedure, 0, 1)
	procedures = append(procedures, yarpcthrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }

func (h handler) Echo(ctx context.Context, body wire.Value) (yarpcthrift.Response, error) {
	var args echo.Echo_Echo_Args
	if err := args.FromWire(body); err != nil {
		return yarpcthrift.Response{}, err
	}

	success, err := h.impl.Echo(ctx, args.Request)

	hadError := err != nil
	result, err := echo.Echo_Echo_Helper.WrapResponse(success, err)

	var response yarpcthrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
	}
	return response, err
}
