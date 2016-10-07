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

package keyvalueclient

import (
	"go.uber.org/thriftrw/wire"
	"golang.org/x/net/context"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/examples/thrift/keyvalue/kv/service/keyvalue"
	"go.uber.org/yarpc"
)

// Interface is a client for the KeyValue service.
type Interface interface {
	GetValue(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Key *string,
	) (string, yarpc.CallResMeta, error)

	SetValue(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Key *string,
		Value *string,
	) (yarpc.CallResMeta, error)
}

// New builds a new client for the KeyValue service.
//
// 	client := keyvalueclient.New(dispatcher.Channel("keyvalue"))
func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{
		Service: "KeyValue",
		Channel: c,
	}, opts...)}
}

func init() {
	yarpc.RegisterClientBuilder(func(c transport.Channel) Interface {
		return New(c)
	})
}

type client struct{ c thrift.Client }

func (c client) GetValue(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Key *string,
) (success string, resMeta yarpc.CallResMeta, err error) {
	args := keyvalue.GetValueHelper.Args(_Key)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result keyvalue.GetValueResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = keyvalue.GetValueHelper.UnwrapResponse(&result)
	return
}

func (c client) SetValue(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Key *string,
	_Value *string,
) (resMeta yarpc.CallResMeta, err error) {
	args := keyvalue.SetValueHelper.Args(_Key, _Value)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result keyvalue.SetValueResult
	if err = result.FromWire(body); err != nil {
		return
	}

	err = keyvalue.SetValueHelper.UnwrapResponse(&result)
	return
}
