// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2019 Uber Technologies, Inc.
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

package secondserviceserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet"
)

// Interface is the server-side interface for the SecondService service.
type Interface interface {
	BlahBlah(
		ctx context.Context,
	) error

	SecondtestString(
		ctx context.Context,
		Thing *string,
	) (string, error)
}

// New prepares an implementation of the SecondService service for
// registration.
//
// 	handler := SecondServiceHandler{}
// 	dispatcher.Register(secondserviceserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Procedure {
	h := handler{impl}
	service := thrift.Service{
		Name: "SecondService",
		Methods: []thrift.Method{

			thrift.Method{
				Name: "blahBlah",
				HandlerSpec: thrift.HandlerSpec{

					Type:  transport.Unary,
					Unary: thrift.UnaryHandler(h.BlahBlah),
				},
				Signature:    "BlahBlah()",
				ThriftModule: gauntlet.ThriftModule,
			},

			thrift.Method{
				Name: "secondtestString",
				HandlerSpec: thrift.HandlerSpec{

					Type:  transport.Unary,
					Unary: thrift.UnaryHandler(h.SecondtestString),
				},
				Signature:    "SecondtestString(Thing *string) (string)",
				ThriftModule: gauntlet.ThriftModule,
			},
		},
	}

	procedures := make([]transport.Procedure, 0, 2)
	procedures = append(procedures, thrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }

func (h handler) BlahBlah(ctx context.Context, body wire.Value) (thrift.Response, error) {
	var args gauntlet.SecondService_BlahBlah_Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	err := h.impl.BlahBlah(ctx)

	hadError := err != nil
	result, err := gauntlet.SecondService_BlahBlah_Helper.WrapResponse(err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
	}
	return response, err
}

func (h handler) SecondtestString(ctx context.Context, body wire.Value) (thrift.Response, error) {
	var args gauntlet.SecondService_SecondtestString_Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, err := h.impl.SecondtestString(ctx, args.Thing)

	hadError := err != nil
	result, err := gauntlet.SecondService_SecondtestString_Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
	}
	return response, err
}
