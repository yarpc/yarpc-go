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
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"golang.org/x/net/context"
)

// Client is a generic Thrift client. It speaks in raw Thrift payloads. The code
// generator is responsible for putting a pretty interface in front of it.
type Client interface {
	Call(ctx context.Context, method string, r *Request) (*Response, yarpc.Meta, error)
}

// Config contains the configuration for the Client.
type Config struct {
	// Name of the Thrift service.
	Service string

	// Outbound through which requests will be sent. Required.
	Outbound transport.Outbound

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
		p:       p,
		t:       c.Outbound,
		service: c.Service,
	}
}

type thriftClient struct {
	t transport.Outbound
	p protocol.Protocol

	service string
}

func (t thriftClient) Call(ctx context.Context, method string, r *Request) (*Response, yarpc.Meta, error) {
	// Code generated for Thrift client calls will probable be something like
	// this:
	//
	// 	func (c *MyServiceClient) someMethod(ctx context.Context, m yarpc.Meta, arg1 Arg1Type, arg2Type) (returnValue, yarpc.Meta, error) {
	// 		args := someMethodArgs{arg1: arg1, arg2: arg2}
	// 		resp, m, err := c.client.Call(ctx, "someMethod", &thrift.Request{
	// 			Meta: m,
	// 			Body: args.ToWire(),
	// 		})
	// 		if err != nil { return nil, m, err }
	// 		if resp.Exception1 != nil {
	// 			return nil, m, resp.Exception1
	// 		}
	// 		if resp.Exception2 != nil {
	// 			return nil, m, resp.Exception2
	// 		}
	// 		if resp.Success != nil {
	// 			return resp.Succ, m, nil
	// 		}
	// 		// TODO: Throw an error here because we expected a non-void return
	// 		// but we got neither an exception, nor a return value.
	// 	}

	// TODO don't store this in memory. Use a ResponseWriter-like interface for
	// underlying transport.
	var buffer bytes.Buffer
	if err := t.p.Encode(r.Body, &buffer); err != nil {
		return nil, nil, encodeError{Reason: err}
	}

	var headers map[string]string
	if r.Meta != nil {
		headers = r.Meta.Headers()
	}

	tres, err := t.t.Call(ctx, &transport.Request{
		Procedure: procedureName(t.service, method),
		Headers:   headers,
		Body:      &buffer,
	})
	if err != nil {
		return nil, nil, err
	}

	defer tres.Body.Close()
	payload, err := ioutil.ReadAll(tres.Body)
	if err != nil {
		return nil, nil, err
	}

	res, err := t.p.Decode(bytes.NewReader(payload), wire.TStruct)
	if err != nil {
		return nil, nil, decodeError{Reason: err}
	}

	meta := yarpc.NewMeta(tres.Headers)
	return &Response{Body: res}, meta, nil
}
