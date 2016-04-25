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

	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
)

// Client is a generic Thrift client. It speaks in raw Thrift payloads. The code
// generator is responsible for putting a pretty interface in front of it.
type Client interface {
	// Call the given Thrift method.
	Call(method string, req *Request, body wire.Value) (wire.Value, *Response, error)
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

func (c thriftClient) Call(method string, req *Request, reqBody wire.Value) (wire.Value, *Response, error) {
	// Code generated for Thrift client calls will probably be something like
	// this:
	//
	// 	func (c *MyServiceClient) someMethod(req *thrift.Request, arg1 Arg1Type, arg2Type) (returnValue, *thrift.Response, error) {
	// 		args := someMethodArgs{arg1: arg1, arg2: arg2}
	// 		resBody, res, err := c.client.Call("someMethod", req, args.ToWire())
	// 		if err != nil { return nil, res, err }
	// 		if resBody.Exception1 != nil {
	// 			return nil, res, resBody.Exception1
	// 		}
	// 		if resBody.Exception2 != nil {
	// 			return nil, res, resBody.Exception2
	// 		}
	// 		if resBody.Success != nil {
	// 			return resp.Succ, res, nil
	// 		}
	// 		// TODO: Throw an error here because we expected a non-void return
	// 		// but we got neither an exception, nor a return value.
	// 	}

	treq := transport.Request{
		Caller:    c.caller,
		Service:   c.service,
		Encoding:  Encoding,
		Procedure: procedureName(c.thriftService, method),
		Headers:   req.Headers,
		TTL:       req.TTL,
	}

	var buffer bytes.Buffer
	if err := c.p.Encode(reqBody, &buffer); err != nil {
		return wire.Value{}, nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buffer
	tres, err := c.t.Call(req.Context, &treq)
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

	return resBody, &Response{Headers: tres.Headers}, nil
}
