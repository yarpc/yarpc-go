// Copyright (c) 2018 Uber Technologies, Inc.
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

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/pkg/errors"
)

// thriftUnaryHandler wraps a Thrift Handler into a transport.UnaryHandler
type thriftUnaryHandler struct {
	UnaryHandler UnaryHandler
	Protocol     protocol.Protocol
	Enveloping   bool
}

// thriftOnewayHandler wraps a Thrift Handler into a transport.OnewayHandler
type thriftOnewayHandler struct {
	OnewayHandler OnewayHandler
	Protocol      protocol.Protocol
	Enveloping    bool
}

func (t thriftUnaryHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	ctx, call := encodingapi.NewInboundCall(ctx)

	reqValue, responder, err := decodeRequest(call, buf, treq, wire.Call, t.Protocol, t.Enveloping)
	if err != nil {
		return err
	}

	res, err := t.UnaryHandler(ctx, reqValue)
	if err != nil {
		return err
	}

	if resType := res.Body.EnvelopeType(); resType != wire.Reply {
		return errors.ResponseBodyEncodeError(
			treq, errUnexpectedEnvelopeType(resType))
	}

	value, err := res.Body.ToWire()
	if err != nil {
		return err
	}

	if res.IsApplicationError {
		rw.SetApplicationError()
	}

	if err := call.WriteToResponse(rw); err != nil {
		// not reachable
		return err
	}

	if err = responder.EncodeResponse(value, wire.Reply, rw); err != nil {
		return errors.ResponseBodyEncodeError(treq, err)
	}

	return nil
}

func (t thriftOnewayHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	ctx, call := encodingapi.NewInboundCall(ctx)

	reqValue, _, err := decodeRequest(call, buf, treq, wire.OneWay, t.Protocol, t.Enveloping)
	if err != nil {
		return err
	}

	return t.OnewayHandler(ctx, reqValue)
}

// decodeRequest is a utility shared by Unary and Oneway handlers, to decode
// the request, regardless of enveloping.
func decodeRequest(
	call *encodingapi.InboundCall,
	// buf is a byte buffer from the buffer pool, that will be released back to
	// the buffer pool by the caller after it is finished with the decoded
	// request payload. Thrift requests read sets, maps, and lists lazilly.
	buf *bufferpool.Buffer,
	treq *transport.Request,
	// reqEnvelopeType indicates the expected envelope type, if an envelope is
	// present.
	reqEnvelopeType wire.EnvelopeType,
	// proto is the encoding protocol (e.g., Binary) or an
	// EnvelopeAgnosticProtocol (e.g., EnvelopeAgnosticBinary)
	proto protocol.Protocol,
	// enveloping indicates that requests must be enveloped, used only if the
	// protocol is not envelope agnostic.
	enveloping bool,
) (
	// call is an inboundCall populated from the transport request and context.
	// reqValue is the wire representation of the decoded request.
	// decodeRequest does not surface the envelope.
	wire.Value,
	// responder indicates how to encode the response, with the enveloping
	// strategy corresponding to the request. It is not used for oneway handlers.
	protocol.Responder,
	error,
) {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return wire.Value{}, nil, err
	}

	if err := call.ReadFromRequest(treq); err != nil {
		// not reachable
		return wire.Value{}, nil, err
	}

	if _, err := buf.ReadFrom(treq.Body); err != nil {
		return wire.Value{}, nil, err
	}

	reader := bytes.NewReader(buf.Bytes())

	// Discover or choose the appropriate envelope
	if agnosticProto, ok := proto.(protocol.EnvelopeAgnosticProtocol); ok {
		return agnosticProto.DecodeRequest(reqEnvelopeType, reader)
	} else if enveloping {
		return decodeEnvelopedRequest(treq, reqEnvelopeType, proto, reader)
	} else {
		return decodeUnenvelopedRequest(proto, reader)
	}
}

func decodeEnvelopedRequest(
	treq *transport.Request,
	reqEnvelopeType wire.EnvelopeType,
	proto protocol.Protocol,
	reader *bytes.Reader,
) (wire.Value, protocol.Responder, error) {
	var envelope wire.Envelope
	envelope, err := proto.DecodeEnveloped(reader)
	if err != nil {
		return wire.Value{}, nil, err
	}
	if envelope.Type != reqEnvelopeType {
		err := errors.RequestBodyDecodeError(treq, errUnexpectedEnvelopeType(envelope.Type))
		return wire.Value{}, nil, err
	}
	reqValue := envelope.Value
	responder := protocol.EnvelopeV1Responder{Name: envelope.Name, SeqID: envelope.SeqID}
	return reqValue, responder, nil
}

func decodeUnenvelopedRequest(
	proto protocol.Protocol,
	reader *bytes.Reader,
) (wire.Value, protocol.Responder, error) {
	reqValue, err := proto.Decode(reader, wire.TStruct)
	if err != nil {
		return wire.Value{}, nil, err
	}
	responder := protocol.NoEnvelopeResponder
	return reqValue, responder, err
}
