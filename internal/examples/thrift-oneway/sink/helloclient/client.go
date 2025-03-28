// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2025 Uber Technologies, Inc.
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

package helloclient

import (
	context "context"
	yarpc "go.uber.org/yarpc"
	transport "go.uber.org/yarpc/api/transport"
	thrift "go.uber.org/yarpc/encoding/thrift"
	sink "go.uber.org/yarpc/internal/examples/thrift-oneway/sink"
	reflect "reflect"
)

// Interface is a client for the Hello service.
type Interface interface {
	Sink(
		ctx context.Context,
		Snk *sink.SinkRequest,
		opts ...yarpc.CallOption,
	) (yarpc.Ack, error)
}

// New builds a new client for the Hello service.
//
//	client := helloclient.New(dispatcher.ClientConfig("hello"))
func New(c transport.ClientConfig, opts ...thrift.ClientOption) Interface {
	return client{
		c: thrift.New(thrift.Config{
			Service:      "Hello",
			ClientConfig: c,
		}, opts...),
		nwc: thrift.NewNoWire(thrift.Config{
			Service:      "Hello",
			ClientConfig: c,
		}, opts...),
	}
}

func init() {
	yarpc.RegisterClientBuilder(
		func(c transport.ClientConfig, f reflect.StructField) Interface {
			return New(c, thrift.ClientBuilderOptions(c, f)...)
		},
	)
}

type client struct {
	c   thrift.Client
	nwc thrift.NoWireClient
}

func (c client) Sink(
	ctx context.Context,
	_Snk *sink.SinkRequest,
	opts ...yarpc.CallOption,
) (yarpc.Ack, error) {
	args := sink.Hello_Sink_Helper.Args(_Snk)
	return c.c.CallOneway(ctx, args, opts...)
}
