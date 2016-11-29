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

package secondserviceclient

import (
	"context"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet"
	"go.uber.org/yarpc"
)

// Interface is a client for the SecondService service.
type Interface interface {
	BlahBlah(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
	) (yarpc.CallResMeta, error)

	SecondtestString(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *string,
	) (string, yarpc.CallResMeta, error)
}

// New builds a new client for the SecondService service.
//
// 	client := secondserviceclient.New(dispatcher.ClientConfig("secondservice"))
func New(c transport.ClientConfig, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{
		Service:      "SecondService",
		ClientConfig: c,
	}, opts...)}
}

func init() {
	yarpc.RegisterClientBuilder(func(c transport.ClientConfig) Interface {
		return New(c)
	})
}

type client struct{ c thrift.Client }

func (c client) BlahBlah(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
) (resMeta yarpc.CallResMeta, err error) {

	args := gauntlet.SecondService_BlahBlah_Helper.Args()

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result gauntlet.SecondService_BlahBlah_Result
	if err = result.FromWire(body); err != nil {
		return
	}

	err = gauntlet.SecondService_BlahBlah_Helper.UnwrapResponse(&result)
	return
}

func (c client) SecondtestString(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *string,
) (success string, resMeta yarpc.CallResMeta, err error) {

	args := gauntlet.SecondService_SecondtestString_Helper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result gauntlet.SecondService_SecondtestString_Result
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = gauntlet.SecondService_SecondtestString_Helper.UnwrapResponse(&result)
	return
}
