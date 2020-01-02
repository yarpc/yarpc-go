// Copyright (c) 2020 Uber Technologies, Inc.
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
	"fmt"
	"io"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

type errUnexpectedEnvelopeType wire.EnvelopeType

func (e errUnexpectedEnvelopeType) Error() string {
	return fmt.Sprintf("unexpected envelope type: %v", wire.EnvelopeType(e))
}

// disableEnvelopingProtocol wraps a protocol to not envelope payloads.
type disableEnvelopingProtocol struct {
	protocol.Protocol

	// EnvelopeType to use for decoded envelopes.
	Type wire.EnvelopeType
}

func (ev disableEnvelopingProtocol) EncodeEnveloped(e wire.Envelope, w io.Writer) error {
	return ev.Encode(e.Value, w)
}

func (ev disableEnvelopingProtocol) DecodeEnveloped(r io.ReaderAt) (wire.Envelope, error) {
	value, err := ev.Decode(r, wire.TStruct)
	return wire.Envelope{
		Name:  "", // we don't use the decoded name anywhere
		Type:  ev.Type,
		SeqID: 1,
		Value: value,
	}, err
}
