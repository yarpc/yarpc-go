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
	"fmt"
	"io/ioutil"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift/internal"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/meta"
	"go.uber.org/yarpc/transport"

	"context"

	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

// Client is a generic Thrift client. It speaks in raw Thrift payloads.
//
// Users should use the client generated by the code generator rather than
// using this directly.
type Client interface {
	// Call the given Thrift method.
	Call(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody envelope.Enveloper) (wire.Value, yarpc.CallResMeta, error)
}

// Config contains the configuration for the Client.
type Config struct {
	// Name of the Thrift service. This is the name used in the Thrift file
	// with the 'service' keyword.
	Service string

	// Channel through which requests will be sent. Required.
	Channel transport.Channel
}

// New creates a new Thrift client.
func New(c Config, opts ...ClientOption) Client {
	// Code generated for Thrift client instantiation will probably be something
	// like this:
	//
	// 	func New(ch transport.Channel, opts ...ClientOption) *MyServiceClient {
	// 		c := thrift.New(thrift.Config{
	// 			Service: "MyService",
	// 			Channel: ch,
	// 			Protocol: protocol.Binary,
	// 		}, opts...)
	// 		return &MyServiceClient{client: c}
	// 	}
	//
	// So Config is really the internal config as far as consumers of the
	// generated client are concerned.

	var cc clientConfig
	for _, opt := range opts {
		opt.applyClientOption(&cc)
	}

	p := protocol.Binary
	if cc.Protocol != nil {
		p = cc.Protocol
	}

	if cc.Multiplexed {
		p = multiplexedOutboundProtocol{
			Protocol: p,
			Service:  c.Service,
		}
	}

	return thriftClient{
		p:             p,
		ch:            c.Channel,
		thriftService: c.Service,
		Enveloping:    cc.Enveloping,
	}
}

type thriftClient struct {
	ch transport.Channel
	p  protocol.Protocol

	// name of the Thrift service
	thriftService string
	Enveloping    bool
}

func (c thriftClient) Call(ctx context.Context, reqMeta yarpc.CallReqMeta, reqBody envelope.Enveloper) (wire.Value, yarpc.CallResMeta, error) {
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

	proto := c.p
	if !c.Enveloping {
		proto = disableEnvelopingProtocol{
			Protocol: proto,
			Type:     wire.Reply, // we only decode replies with this instance
		}
	}

	treq := transport.Request{
		Caller:   c.ch.Caller(),
		Service:  c.ch.Service(),
		Encoding: Encoding,
	}
	meta.ToTransportRequest(reqMeta, &treq)
	// Always override the procedure name to the Thrift procedure name.
	treq.Procedure = procedureName(c.thriftService, reqBody.MethodName())

	value, err := reqBody.ToWire()
	if err != nil {
		// ToWire validates the request. If it failed, we should return the error
		// as-is because it's not an encoding error.
		return wire.Value{}, nil, err
	}

	reqEnvelopeType := reqBody.EnvelopeType()
	if reqEnvelopeType != wire.Call {
		return wire.Value{}, nil, encoding.RequestBodyEncodeError(
			&treq, errUnexpectedEnvelopeType(reqEnvelopeType),
		)
	}

	var buffer bytes.Buffer
	err = proto.EncodeEnveloped(wire.Envelope{
		Name:  reqBody.MethodName(),
		Type:  reqEnvelopeType,
		SeqID: 1, // don't care
		Value: value,
	}, &buffer)
	if err != nil {
		return wire.Value{}, nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buffer
	tres, err := c.ch.GetOutbound().Call(ctx, &treq)
	if err != nil {
		return wire.Value{}, nil, err
	}

	defer tres.Body.Close()
	payload, err := ioutil.ReadAll(tres.Body)
	if err != nil {
		return wire.Value{}, nil, err
	}

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(payload))
	if err != nil {
		return wire.Value{}, nil, encoding.ResponseBodyDecodeError(&treq, err)
	}

	switch envelope.Type {
	case wire.Reply:
		return envelope.Value, meta.FromTransportResponse(tres), nil
	case wire.Exception:
		var exc internal.TApplicationException
		if err := exc.FromWire(envelope.Value); err != nil {
			return wire.Value{}, nil, encoding.ResponseBodyDecodeError(&treq, err)
		}
		return wire.Value{}, nil, thriftException{
			Service:   treq.Service,
			Procedure: treq.Procedure,
			Reason:    &exc,
		}
	default:
		return wire.Value{}, nil, encoding.ResponseBodyDecodeError(
			&treq, errUnexpectedEnvelopeType(envelope.Type))
	}
}

type thriftException struct {
	Service   string
	Procedure string
	Reason    *internal.TApplicationException
}

func (e thriftException) Error() string {
	return fmt.Sprintf(
		"thrift request to procedure %q of service %q encountered an internal failure: %v",
		e.Procedure, e.Service, e.Reason)
}
