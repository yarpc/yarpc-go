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

// NoWireCall contains all of the required objects needed for an underlying
// Handle needs to unpack any given request.
type NoWireCall struct {
	Reader        io.Reader
	RequestReader stream.RequestReader
	EnvelopeType  wire.EnvelopeType
}

// NoWireHandler is implemented by the generated code for each method that the
// server needs to implement.  It is responsible for unpacking the request,
// executing it, and returning a NoWireResponse that contains information about
// how to construct a response as well as any relevant metadata while executing
// the request.
type NoWireHandler interface {
	Handle(context.Context, *NoWireCall) (NoWireResponse, error)
}

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

var (
	_ transport.OnewayHandler = (*thriftNoWireHandler)(nil)
	_ transport.UnaryHandler  = (*thriftNoWireHandler)(nil)
)

func (t thriftNoWireHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	ctx, call := encodingapi.NewInboundCall(ctx)
	defer closeReader(treq.Body)

	res, err := t.decodeAndHandle(ctx, call, treq, rw, wire.Call)
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

	_, err := t.decodeAndHandle(ctx, call, treq, nil, wire.OneWay)
	return err
}

// decodeAhdnHandle is a shared utility between the implementations of
// transport.UnaryHandler and transport.OnewayHandler, to decode and execute
// the request regardless of enveloping, via the "nowire" implementation in
// ThriftRW.
func (t thriftNoWireHandler) decodeAndHandle(
	ctx context.Context,
	call *encodingapi.InboundCall,
	treq *transport.Request,
	rw transport.ResponseWriter,
	reqEnvelopeType wire.EnvelopeType,
) (NoWireResponse, error) {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return _emptyResponse, err
	}

	if err := call.ReadFromRequest(treq); err != nil {
		return _emptyResponse, err
	}

	nwc := NoWireCall{
		Reader:       treq.Body,
		EnvelopeType: reqEnvelopeType,
	}

	if reqReader, ok := t.Protocol.(stream.RequestReader); ok {
		nwc.RequestReader = reqReader
	} else {
		nwc.RequestReader = &reqReaderProto{
			Protocol:   t.Protocol,
			treq:       treq,
			enveloping: t.Enveloping,
		}
	}
	return t.NoWireHandler.Handle(ctx, &nwc)
}

// reqReaderProto is an implementation of ThriftRW's stream.RequestReader in
// case the provided stream.Protocol does not implement the necessary
// `ReadRequest` to discover the correct enveloping.
type reqReaderProto struct {
	stream.Protocol

	treq       *transport.Request
	enveloping bool
}

var _ stream.RequestReader = (*reqReaderProto)(nil)

func (p *reqReaderProto) ReadRequest(
	ctx context.Context,
	et wire.EnvelopeType,
	r io.Reader,
	body stream.BodyReader,
) (stream.ResponseWriter, error) {
	sr := p.Protocol.Reader(r)
	defer sr.Close()

	if p.enveloping {
		eh, err := sr.ReadEnvelopeBegin()
		if err != nil {
			return nil, errors.RequestBodyDecodeError(p.treq, err)
		}

		if eh.Type != et {
			return nil, errors.RequestBodyDecodeError(p.treq, errUnexpectedEnvelopeType(eh.Type))
		}

		if err := body.Decode(sr); err != nil {
			return nil, errors.RequestBodyDecodeError(p.treq, err)
		}

		if err := sr.ReadEnvelopeEnd(); err != nil {
			return nil, errors.RequestBodyDecodeError(p.treq, err)
		}
		return binary.EnvelopeV1Responder{
			Name:  eh.Name,
			SeqID: eh.SeqID,
		}, nil
	}

	if err := body.Decode(sr); err != nil {
		return nil, errors.RequestBodyDecodeError(p.treq, err)
	}
	return binary.NoEnvelopeResponder, nil
}
