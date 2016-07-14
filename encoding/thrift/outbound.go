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

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/thrift/internal"
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
func New(c Config, opts ...ClientOption) Client {
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

	var cc clientConfig
	for _, opt := range opts {
		opt.applyClientOption(&cc)
	}

	return thriftClient{
		p: disableEnveloper{
			Protocol: p,
			Type:     wire.Reply, // we only decode replies
		},
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
	err = c.p.EncodeEnveloped(wire.Envelope{
		Name:  reqBody.MethodName(),
		Type:  reqEnvelopeType,
		SeqID: 1, // don't care
		Value: value,
	}, &buffer)
	if err != nil {
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

	envelope, err := c.p.DecodeEnveloped(bytes.NewReader(payload))
	if err != nil {
		return wire.Value{}, nil, encoding.ResponseBodyDecodeError(&treq, err)
	}

	switch envelope.Type {
	case wire.Reply:
		// TODO(abg): when transport returns response context, use that here.
		return envelope.Value, meta.FromTransportResponse(ctx, tres), nil
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
