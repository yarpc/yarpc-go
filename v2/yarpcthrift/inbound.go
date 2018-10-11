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

package yarpcthrift

import (
	"bytes"
	"context"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcencoding"
)

var _ yarpc.UnaryTransportHandler = (*unaryTransportHandler)(nil)

// unaryTransportHandler wraps a Thrift Handler into a yarpc.UnaryTransportHandler.
type unaryTransportHandler struct {
	ThriftHandler Handler
	Protocol      protocol.Protocol
	Enveloping    bool
}

// Handle implements yarpc.UnaryTransportHandler.
func (t unaryTransportHandler) Handle(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	ctx, call := yarpc.NewInboundCall(ctx)

	reqValue, responder, err := decodeRequest(call, req, reqBuf, t.Protocol, t.Enveloping)
	if err != nil {
		return nil, nil, err
	}

	thriftRes, err := t.ThriftHandler(ctx, reqValue)
	if err != nil {
		return nil, nil, err
	}

	if resType := thriftRes.Body.EnvelopeType(); resType != wire.Reply {
		return nil, nil, yarpcencoding.ResponseBodyEncodeError(
			req, errUnexpectedEnvelopeType(resType))
	}

	value, err := thriftRes.Body.ToWire()
	if err != nil {
		return nil, nil, err
	}

	res, resBuf := &yarpc.Response{}, &yarpc.Buffer{}

	if thriftRes.IsApplicationError {
		res.ApplicationError = true
	}

	call.WriteToResponse(res)

	if err = responder.EncodeResponse(value, wire.Reply, resBuf); err != nil {
		return nil, nil, yarpcencoding.ResponseBodyEncodeError(req, err)
	}

	return res, resBuf, nil
}

// decodeRequest is a utility shared by Unary handlers, to decode
// the request, regardless of enveloping.
func decodeRequest(
	// call is an inboundCall populated from the transport request and context.
	call *yarpc.InboundCall,
	req *yarpc.Request,
	reqBuf *yarpc.Buffer,
	// proto is the encoding protocol (e.g., Binary) or an
	// EnvelopeAgnosticProtocol (e.g., EnvelopeAgnosticBinary)
	proto protocol.Protocol,
	// enveloping indicates that requests must be enveloped, used only if the
	// protocol is not envelope agnostic.
	enveloping bool,
) (
	// the wire representation of the decoded request.
	// decodeRequest does not surface the envelope.
	wire.Value,
	// how to encode the response, with the enveloping
	// strategy corresponding to the request.
	protocol.Responder,
	error,
) {
	if err := yarpcencoding.ExpectEncodings(req, Encoding); err != nil {
		return wire.Value{}, nil, err
	}

	if err := call.ReadFromRequest(req); err != nil {
		// not reachable
		return wire.Value{}, nil, err
	}

	// TODO(mhp): This will be unnecessary when `yarpc.Buffer` implements `io.ReaderAt`
	reader := bytes.NewReader(reqBuf.Bytes())

	// Discover or choose the appropriate envelope
	if agnosticProto, ok := proto.(protocol.EnvelopeAgnosticProtocol); ok {
		return agnosticProto.DecodeRequest(wire.Call, reader)
	}
	if enveloping {
		return decodeEnvelopedRequest(req, proto, reader)
	}
	return decodeUnenvelopedRequest(proto, reader)
}

func decodeEnvelopedRequest(
	req *yarpc.Request,
	proto protocol.Protocol,
	reader *bytes.Reader,
) (wire.Value, protocol.Responder, error) {
	var envelope wire.Envelope
	envelope, err := proto.DecodeEnveloped(reader)
	if err != nil {
		return wire.Value{}, nil, err
	}
	if envelope.Type != wire.Call {
		err := yarpcencoding.RequestBodyDecodeError(req, errUnexpectedEnvelopeType(envelope.Type))
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
