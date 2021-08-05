// Copyright (c) 2021 Uber Technologies, Inc.
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
	"context"
	"io"

	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
)

var _emptyResponse = NoWireResponse{}

// thriftNoWireHandler is similar to thriftUnaryHandler and thriftOnewayHandler
// except that thriftNoWireHandler implements both transport.UnaryHandler and
// transport.OnewayHandler through a single type, utilizing the "nowire"
// (streaming in the ThriftRW context, bypassing the intermediate "Wire"
// representation) implementation.
//
// The UnaryHandler and OnewayHandler implementations then call into type that
// implements NoWireHandler, which understands how to unpack and invoke the
// request, given the ThriftRW primitives for reading the raw representation.
type thriftNoWireHandler struct {
	NoWireHandler NoWireHandler
	Protocol      stream.Protocol
	Enveloping    bool
}

type NoWireCall struct {
	Reader        io.Reader
	RequestReader stream.RequestReader
	EnvelopeType  wire.EnvelopeType

	StreamReader stream.Reader
}

type NoWireHandler interface {
	Handle(context.Context, *NoWireCall) (NoWireResponse, error)
}

func (t thriftNoWireHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	ctx, call := encodingapi.NewInboundCall(ctx)
	defer closeReader(treq.Body)

	res, err := decodeNoWireRequest(ctx, call, treq, rw, t.NoWireHandler, wire.Call, t.Protocol, t.Enveloping)
	if err != nil {
		return err
	}

	if resType := res.Body.EnvelopeType(); resType != wire.Reply {
		return errors.ResponseBodyEncodeError(
			treq, errUnexpectedEnvelopeType(resType))
	}

	if res.IsApplicationError {
		rw.SetApplicationError()

		if applicationErrorMetaSetter, ok := rw.(transport.ApplicationErrorMetaSetter); ok {
			applicationErrorMetaSetter.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
				Details: res.ApplicationErrorDetails,
				Name:    res.ApplicationErrorName,
				Code:    res.ApplicationErrorCode,
			})
		}
	}

	if err := call.WriteToResponse(rw); err != nil {
		// not reachable
		return err
	}

	if err := res.ResponseWriter.WriteResponse(wire.Reply, rw, res.Body); err != nil {
		return errors.ResponseBodyEncodeError(treq, err)
	}

	return nil
}

func (t thriftNoWireHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	ctx, call := encodingapi.NewInboundCall(ctx)
	defer closeReader(treq.Body)

	_, err := decodeNoWireRequest(ctx, call, treq, nil, t.NoWireHandler, wire.OneWay, t.Protocol, t.Enveloping)
	return err
}

// decodeNoWireRequest is a shared utility between the implementations of
// transport.UnaryHandler and transport.OnewayHandler, to decode and execute the
// request regardless of enveloping, via the "nowire" implementation in
// ThriftRW.
func decodeNoWireRequest(
	ctx context.Context,
	call *encodingapi.InboundCall,
	treq *transport.Request,
	rw transport.ResponseWriter,
	noWireHandler NoWireHandler,
	reqEnvelopeType wire.EnvelopeType,
	proto stream.Protocol,
	enveloping bool,
) (NoWireResponse, error) {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return _emptyResponse, err
	}

	if err := call.ReadFromRequest(treq); err != nil {
		return _emptyResponse, err
	}

	if reqReader, ok := proto.(stream.RequestReader); ok {
		nwc := NoWireCall{
			Reader:        treq.Body,
			RequestReader: reqReader,
			EnvelopeType:  reqEnvelopeType,
		}
		return noWireHandler.Handle(ctx, &nwc)
	}

	streamReader := proto.Reader(treq.Body)
	defer streamReader.Close()

	if enveloping {
		eh, err := streamReader.ReadEnvelopeBegin()
		if err != nil {
			return _emptyResponse, err
		}
		if eh.Type != reqEnvelopeType {
			return _emptyResponse, errors.RequestBodyDecodeError(treq, errUnexpectedEnvelopeType(eh.Type))
		}

		responder := binary.EnvelopeV1Responder{Name: eh.Name, SeqID: eh.SeqID}
		nwc := NoWireCall{StreamReader: streamReader}
		resp, err := noWireHandler.Handle(ctx, &nwc)
		if err != nil {
			return _emptyResponse, err
		}

		resp.ResponseWriter = responder
		return resp, streamReader.ReadEnvelopeEnd()
	}

	responder := binary.NoEnvelopeResponder
	nwc := NoWireCall{StreamReader: streamReader}
	resp, err := noWireHandler.Handle(ctx, &nwc)
	if err != nil {
		return _emptyResponse, err
	}

	resp.ResponseWriter = responder
	return resp, nil
}
