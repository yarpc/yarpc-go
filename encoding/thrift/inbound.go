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
	"context"
	"io/ioutil"

	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/meta"
	"go.uber.org/yarpc/transport"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

// thriftHandler wraps a Thrift Handler into a transport.Handler and transport.OnewayHandler
type thriftHandler struct {
	Handler       UnaryHandler
	OnewayHandler OnewayHandler
	Protocol      protocol.Protocol
	Enveloping    bool
}

func (t thriftHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	if err := encoding.Expect(treq, Encoding); err != nil {
		return err
	}

	body, err := ioutil.ReadAll(treq.Body)
	if err != nil {
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

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(body))
	if err != nil {
		return encoding.RequestBodyDecodeError(treq, err)
	}

	if envelope.Type != wire.Call {
		return encoding.RequestBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}

	reqMeta := meta.FromTransportRequest(treq)
	res, err := t.Handler.HandleUnary(ctx, reqMeta, envelope.Value)
	if err != nil {
		return err
	}

	if resType := res.Body.EnvelopeType(); resType != wire.Reply {
		return encoding.ResponseBodyEncodeError(
			treq, errUnexpectedEnvelopeType(resType))
	}

	value, err := res.Body.ToWire()
	if err != nil {
		return err
	}

	if res.IsApplicationError {
		rw.SetApplicationError()
	}

	resMeta := res.Meta
	if resMeta != nil {
		meta.ToTransportResponseWriter(resMeta, rw)
	}

	err = proto.EncodeEnveloped(wire.Envelope{
		Name:  res.Body.MethodName(),
		Type:  res.Body.EnvelopeType(),
		SeqID: envelope.SeqID,
		Value: value,
	}, rw)
	if err != nil {
		return encoding.ResponseBodyEncodeError(treq, err)
	}

	return nil
}

// TODO: reduce commonality between Handle and HandleOneway
func (t thriftHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := encoding.Expect(treq, Encoding); err != nil {
		return err
	}

	body, err := ioutil.ReadAll(treq.Body)
	if err != nil {
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

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(body))
	if err != nil {
		return encoding.RequestBodyDecodeError(treq, err)
	}

	if envelope.Type != wire.OneWay {
		return encoding.RequestBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}

	reqMeta := meta.FromTransportRequest(treq)
	return t.OnewayHandler.HandleOneway(ctx, reqMeta, envelope.Value)
}
