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
	"io/ioutil"

	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/internal/meta"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"golang.org/x/net/context"
)

// thriftHandler wraps a Thrift Handler into a transport.Handler
type thriftHandler struct {
	Handler  Handler
	Protocol protocol.Protocol
}

func (t thriftHandler) Handle(ctx context.Context, _ transport.Options, treq *transport.Request, rw transport.ResponseWriter) error {
	treq.Encoding = Encoding
	// TODO(abg): Should we fail requests if Rpc-Encoding does not match?

	body, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	envelope, err := t.Protocol.DecodeEnveloped(bytes.NewReader(body))
	if err != nil {
		return encoding.RequestBodyDecodeError(treq, err)
	}

	if envelope.Type != wire.Call {
		return encoding.RequestBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}
	// TODO(abg): Support oneway

	reqMeta := meta.FromTransportRequest(ctx, treq)
	res, err := t.Handler.Handle(reqMeta, envelope.Value)
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
		_ = meta.ToTransportResponseWriter(resMeta, rw)
		// TODO(abg): propagate response context
	}

	err = t.Protocol.EncodeEnveloped(wire.Envelope{
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
