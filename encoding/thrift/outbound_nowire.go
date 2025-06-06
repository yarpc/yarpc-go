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

package thrift

import (
	"bytes"
	"context"
	"fmt"

	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal"
	"go.uber.org/yarpc/pkg/encoding"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/procedure"
)

// NoWireClient is a generic Thrift client for encoding/decoding using
// ThriftRW's "streaming" mechanisms. The body of the provided request
// ('reqBody') will be written out through its method 'Encode',
// while the body of the response ('resBody') is read out through its method
// 'Decode'.
// It speaks in raw Thrift payloads.
//
// Users should use the client generated by the code generator rather than
// using this directly.
type NoWireClient interface {
	// Call the given Thrift method.
	Call(ctx context.Context, reqBody stream.Enveloper, resBody stream.BodyReader, opts ...yarpc.CallOption) error
	CallOneway(ctx context.Context, reqBody stream.Enveloper, opts ...yarpc.CallOption) (transport.Ack, error)

	// Enabled returns whether or not this client is enabled through a
	// ClientOption. This ClientOption is toggled through the 'NoWire(bool)'
	// option.
	Enabled() bool
}

// NewNoWire creates a new Thrift client that leverages ThriftRW's "streaming"
// implementation.
func NewNoWire(c Config, opts ...ClientOption) NoWireClient {
	// Code generated for Thrift client instantiation will probably be something
	// like this:
	//
	// 	func New(cc transport.ClientConfig, opts ...ClientOption) *MyServiceClient {
	// 		c := thrift.NewNoWire(thrift.Config{
	// 			Service: "MyService",
	// 			ClientConfig: cc,
	// 			Protocol: binary.Default,
	// 		}, opts...)
	// 		return &MyServiceClient{client: c}
	// 	}
	//
	// So Config is really the internal config as far as consumers of the
	// generated client are concerned.

	// default NoWire to true because this is the our final state to achieve
	// but we still allow users to opt out by overriding NoWire to false.
	cc := clientConfig{NoWire: true}
	for _, opt := range opts {
		opt.applyClientOption(&cc)
	}

	var p stream.Protocol = binary.Default
	if cc.Protocol != nil {
		if val, ok := cc.Protocol.(stream.Protocol); ok {
			p = val
		} else {
			panic(fmt.Sprintf(
				"Protocol config option provided, NewNoWire expects provided protocol %T to implement stream.Protocol", cc.Protocol))
		}
	}

	svc := c.Service
	if cc.ServiceName != "" {
		svc = cc.ServiceName
	}

	if cc.Multiplexed {
		p = multiplexedOutboundNoWireProtocol{
			Protocol: p,
			Service:  svc,
		}
	}

	return noWireThriftClient{
		p:             p,
		cc:            c.ClientConfig,
		thriftService: svc,
		Enveloping:    cc.Enveloping,
		NoWire:        cc.NoWire,
	}
}

type noWireThriftClient struct {
	cc transport.ClientConfig
	p  stream.Protocol

	// name of the Thrift service
	thriftService string
	Enveloping    bool
	NoWire        bool
}

func (c noWireThriftClient) Call(ctx context.Context, reqBody stream.Enveloper, resBody stream.BodyReader, opts ...yarpc.CallOption) error {
	// Code generated for Thrift client calls will probably be something like
	// this:
	//
	// 	func (c *MyServiceClient) someMethod(ctx context.Context, arg1 Arg1Type, arg2 arg2Type, opts ...yarpc.CallOption) (returnValue, error) {
	// 		var result myservice.SomeMethodResult
	// 		args := myservice.SomeMethodHelper.Args(arg1, arg2)
	// 		err := c.client.Call(ctx, args, result, opts...)
	//
	// 		success, err := myservice.SomeMethodHelper.UnwrapResponse(&result)
	// 		return success, err
	// 	}

	out := c.cc.GetUnaryOutbound()

	treq, proto, err := c.buildTransportRequest(reqBody)
	if err != nil {
		return err
	}

	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	ctx, err = call.WriteToRequest(ctx, treq)
	if err != nil {
		return err
	}

	tres, err := out.Call(ctx, treq)
	if err != nil {
		return err
	}
	defer tres.Body.Close()

	if _, err := call.ReadFromResponse(ctx, tres); err != nil {
		return err
	}

	sr := proto.Reader(tres.Body)
	defer sr.Close()

	envelope, err := sr.ReadEnvelopeBegin()
	if err != nil {
		return errors.ResponseBodyDecodeError(treq, err)
	}

	switch envelope.Type {
	case wire.Reply:
		if err := resBody.Decode(sr); err != nil {
			return err
		}
		return sr.ReadEnvelopeEnd()
	case wire.Exception:
		var exc internal.TApplicationException
		if err := exc.Decode(sr); err != nil {
			return errors.ResponseBodyDecodeError(treq, err)
		}
		defer func() {
			_ = sr.ReadEnvelopeEnd()
		}()

		return thriftException{
			Service:   treq.Service,
			Procedure: treq.Procedure,
			Reason:    &exc,
		}
	default:
		return errors.ResponseBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}
}

func (c noWireThriftClient) CallOneway(ctx context.Context, reqBody stream.Enveloper, opts ...yarpc.CallOption) (transport.Ack, error) {
	out := c.cc.GetOnewayOutbound()

	treq, _, err := c.buildTransportRequest(reqBody)
	if err != nil {
		return nil, err
	}

	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	ctx, err = call.WriteToRequest(ctx, treq)
	if err != nil {
		return nil, err
	}

	return out.CallOneway(ctx, treq)
}

func (c noWireThriftClient) Enabled() bool {
	return c.NoWire
}

func (c noWireThriftClient) buildTransportRequest(reqBody stream.Enveloper) (*transport.Request, stream.Protocol, error) {
	proto := c.p
	if !c.Enveloping {
		proto = disableEnvelopingNoWireProtocol{
			Protocol: proto,
			Type:     wire.Reply, // we only decode replies with this instance
		}
	}

	treq := transport.Request{
		Caller:    c.cc.Caller(),
		Service:   c.cc.Service(),
		Encoding:  Encoding,
		Procedure: procedure.ToName(c.thriftService, reqBody.MethodName()),
	}

	envType := reqBody.EnvelopeType()
	if envType != wire.Call && envType != wire.OneWay {
		return nil, nil, errors.RequestBodyEncodeError(
			&treq, errUnexpectedEnvelopeType(envType),
		)
	}

	var buffer bytes.Buffer
	sw := proto.Writer(&buffer)
	defer sw.Close()

	if err := sw.WriteEnvelopeBegin(stream.EnvelopeHeader{
		Name:  reqBody.MethodName(),
		Type:  envType,
		SeqID: 1, // don't care
	}); err != nil {
		return nil, nil, errors.RequestBodyEncodeError(&treq, err)
	}

	if err := reqBody.Encode(sw); err != nil {
		return nil, nil, errors.RequestBodyEncodeError(&treq, err)
	}

	if err := sw.WriteEnvelopeEnd(); err != nil {
		return nil, nil, errors.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buffer
	treq.BodySize = buffer.Len()
	return &treq, proto, nil
}
