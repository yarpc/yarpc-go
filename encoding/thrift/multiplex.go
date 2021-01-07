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
	"io"
	"strings"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

// multiplexedOutboundProtocol is a Protocol for outbound requests that adds
// the name of the service to the envelope name for outbound requests and
// strips it away for inbound responses.
type multiplexedOutboundProtocol struct {
	protocol.Protocol

	// Name of the Thrift service
	Service string
}

func (m multiplexedOutboundProtocol) EncodeEnveloped(e wire.Envelope, w io.Writer) error {
	e.Name = m.Service + ":" + e.Name
	return m.Protocol.EncodeEnveloped(e, w)
}

func (m multiplexedOutboundProtocol) DecodeEnveloped(r io.ReaderAt) (wire.Envelope, error) {
	e, err := m.Protocol.DecodeEnveloped(r)
	e.Name = strings.TrimPrefix(e.Name, m.Service+":")
	return e, err
}
