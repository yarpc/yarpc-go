// Code generated by thriftrw

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
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/service/secondservice"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
)

type Interface interface {
	BlahBlah(reqMeta yarpc.CallReqMeta) (yarpc.CallResMeta, error)
	SecondtestString(reqMeta yarpc.CallReqMeta, thing *string) (string, yarpc.CallResMeta, error)
}

func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{Service: "SecondService", Channel: c, Protocol: protocol.Binary}, opts...)}
}

type client struct{ c thrift.Client }

func (c client) BlahBlah(reqMeta yarpc.CallReqMeta) (resMeta yarpc.CallResMeta, err error) {
	args := secondservice.BlahBlahHelper.Args()
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
	if err != nil {
		return
	}
	var result secondservice.BlahBlahResult
	if err = result.FromWire(body); err != nil {
		return
	}
	err = secondservice.BlahBlahHelper.UnwrapResponse(&result)
	return
}

func (c client) SecondtestString(reqMeta yarpc.CallReqMeta, thing *string) (success string, resMeta yarpc.CallResMeta, err error) {
	args := secondservice.SecondtestStringHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
	if err != nil {
		return
	}
	var result secondservice.SecondtestStringResult
	if err = result.FromWire(body); err != nil {
		return
	}
	success, err = secondservice.SecondtestStringHelper.UnwrapResponse(&result)
	return
}
