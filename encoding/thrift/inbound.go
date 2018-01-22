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
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	if _, err := buf.ReadFrom(treq.Body); err != nil {
		return err
	}

	// We disable enveloping if either the client or the transport requires it.
	proto := t.Protocol
	if !t.Enveloping {
		proto = disableEnvelopingProtocol{
			Protocol: proto,
			Type:     wire.Call, // we only decode requests
		}
	}

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return errors.RequestBodyDecodeError(treq, err)
	}

	if envelope.Type != wire.Call {
		return errors.RequestBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}

	res, err := t.UnaryHandler(ctx, envelope.Value)
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
		return err
	}

	err = proto.EncodeEnveloped(wire.Envelope{
		Name:  res.Body.MethodName(),
		Type:  res.Body.EnvelopeType(),
		SeqID: envelope.SeqID,
		Value: value,
	}, rw)
	if err != nil {
		return errors.ResponseBodyEncodeError(treq, err)
	}

	return nil
}

// TODO(apb): reduce commonality between Handle and HandleOneway

func (t thriftOnewayHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	if _, err := buf.ReadFrom(treq.Body); err != nil {
		return err
	}

	// We disable enveloping if either the client or the transport requires it.
	proto := t.Protocol
	if !t.Enveloping {
		proto = disableEnvelopingProtocol{
			Protocol: proto,
			Type:     wire.OneWay, // we only decode oneway requests
		}
	}

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return errors.RequestBodyDecodeError(treq, err)
	}

	if envelope.Type != wire.OneWay {
		return errors.RequestBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}

	return t.OnewayHandler(ctx, envelope.Value)
}
