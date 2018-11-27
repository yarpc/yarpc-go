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

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

type thriftCodec struct {
	protocol   protocol.Protocol
	enveloping bool

	responder protocol.Responder
}

func newCodec(protocol protocol.Protocol, enveloping bool) *thriftCodec {
	return &thriftCodec{
		protocol:   protocol,
		enveloping: enveloping,
	}
}

// setResponder checks whether a responder has been set and sets it to one.
//
// An instance of a thrift codec should not set a responder more than once; if
// it does, it may not encode a response properly. We use this instead of
// directly setting a responder to ensure that we only set it once.
func (c *thriftCodec) setResponder(responder protocol.Responder) error {
	if c.responder != nil {
		return yarpcerror.InternalErrorf("tried to overwrite a responder for thrift codec")
	}
	c.responder = responder
	return nil
}

func (c *thriftCodec) Decode(req *yarpc.Buffer) (interface{}, error) {
	// TODO(mhp): This will be unnecessary when `yarpc.Buffer` implements `io.ReaderAt`
	reader := bytes.NewReader(req.Bytes())

	// Discover or choose the appropriate envelope
	if agnosticProto, ok := c.protocol.(protocol.EnvelopeAgnosticProtocol); ok {
		reqValue, responder, err := agnosticProto.DecodeRequest(wire.Call, reader)
		if err != nil {
			return nil, err
		}
		if err := c.setResponder(responder); err != nil {
			return nil, err
		}
		return reqValue, nil
	}

	if c.enveloping {
		envelope, err := c.protocol.DecodeEnveloped(reader)
		if err != nil {
			return nil, err
		}
		if envelope.Type != wire.Call {
			return nil, errUnexpectedEnvelopeType(envelope.Type)
		}
		if err := c.setResponder(protocol.EnvelopeV1Responder{
			Name:  envelope.Name,
			SeqID: envelope.SeqID,
		}); err != nil {
			return nil, err
		}
		return envelope.Value, nil
	}
	reqValue, err := c.protocol.Decode(reader, wire.TStruct)
	if err != nil {
		return nil, err
	}
	if err := c.setResponder(protocol.NoEnvelopeResponder); err != nil {
		return nil, err
	}
	return reqValue, nil
}

func (c *thriftCodec) encode(res interface{}) (*yarpc.Buffer, error) {
	resBuf := &yarpc.Buffer{}
	if resValue, ok := res.(wire.Value); ok {
		err := c.responder.EncodeResponse(resValue, wire.Reply, resBuf)
		return resBuf, err
	}
	return nil, yarpcerror.InternalErrorf("tried to encode a non-wire.Value in thrift codec")
}

func (c *thriftCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	return c.encode(res)
}

func (c *thriftCodec) EncodeError(err error) (*yarpc.Buffer, error) {
	details := yarpcerror.ExtractDetails(err)
	if details == nil {
		return nil, nil
	}
	return c.encode(details)
}
