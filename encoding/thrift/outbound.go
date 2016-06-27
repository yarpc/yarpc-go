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

package thrift

import (
	"bytes"
	"io/ioutil"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/internal/meta"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/envelope"
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
)

// Client is a generic Thrift client. It speaks in raw Thrift payloads. The code
// generator is responsible for putting a pretty interface in front of it.
type Client interface {
	// Call the given Thrift method.
	Call(
		reqMeta yarpc.CallReqMeta,
		reqBody envelope.Enveloper,
	) (wire.Value, yarpc.CallResMeta, error)
}

// Config contains the configuration for the Client.
type Config struct {
	// Name of the Thrift service. This is the name used in the Thrift file
	// with the 'service' keyword.
	Service string

	// Channel through which requests will be sent. Required.
	Channel transport.Channel

	// Thrift encoding protocol. Defaults to Binary if nil.
	Protocol protocol.Protocol
}

// New creates a new Thrift client.
func New(c Config) Client {
	// Code generated for Thrift client instantiation will probably be something
	// like this:
	//
	// 	func New(t transport.Outbound) *MyServiceClient {
	// 		c := thrift.New(thrift.Config{
	// 			Service: "MyService",
	// 			Outbound: t,
	// 			Protocol: protocol.Binary,
	// 		})
	// 		return &MyServiceClient{client: c}
	// 	}
	//
	// So Config is really the internal config as far as consumers of the
	// generated client are concerned.

	p := c.Protocol
	if p == nil {
		p = protocol.Binary
	}

	return thriftClient{
		p:             p,
		t:             c.Channel.Outbound,
		caller:        c.Channel.Caller,
		service:       c.Channel.Service,
		thriftService: c.Service,
	}
}

type thriftClient struct {
	t transport.Outbound
	p protocol.Protocol

	// name of the Thrift service
	thriftService string

	// names of the services making the requests and receiving it.
	caller, service string
}

func (c thriftClient) Call(
	reqMeta yarpc.CallReqMeta,
	reqBody envelope.Enveloper,
) (wire.Value, yarpc.CallResMeta, error) {
	// Code generated for Thrift client calls will probably be something like
	// this:
	//
	// 	func (c *MyServiceClient) someMethod(reqMeta yarpc.CallReqMeta, arg1 Arg1Type, arg2 arg2Type) (returnValue, yarpc.CallResMeta, error) {
	// 		args := myservice.SomeMethodHelper.Args(arg1, arg2)
	// 		resBody, resMeta, err := c.client.Call(reqMeta, args)
	// 		var result myservice.SomeMethodResult
	// 		if err = result.FromWire(resBody); err != nil {
	// 			return nil, resMeta, err
	// 		}
	// 		success, err := myservice.SomeMethodHelper.UnwrapResponse(&result)
	// 		return success, resMeta, err
	// 	}
	treq := transport.Request{
		Caller:   c.caller,
		Service:  c.service,
		Encoding: Encoding,
	}
	ctx := meta.ToTransportRequest(reqMeta, &treq)
	// Always override the procedure name to the Thrift procedure name.
	treq.Procedure = procedureName(c.thriftService, reqBody.MethodName())

	var buffer bytes.Buffer
	if value, err := reqBody.ToWire(); err != nil {
		// ToWire validates the request. If it failed, we should return the error
		// as-is because it's not an encoding error.
		return wire.Value{}, nil, err
	} else if err := c.p.Encode(value, &buffer); err != nil {
		return wire.Value{}, nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buffer
	tres, err := c.t.Call(ctx, &treq)
	if err != nil {
		return wire.Value{}, nil, err
	}

	defer tres.Body.Close()
	payload, err := ioutil.ReadAll(tres.Body)
	if err != nil {
		return wire.Value{}, nil, err
	}

	resBody, err := c.p.Decode(bytes.NewReader(payload), wire.TStruct)
	if err != nil {
		return wire.Value{}, nil, encoding.ResponseBodyDecodeError(&treq, err)
	}

	// TODO: when transport returns response context, use that here.
	return resBody, meta.FromTransportResponse(ctx, tres), nil
}
