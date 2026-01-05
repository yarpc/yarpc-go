// Copyright (c) 2026 Uber Technologies, Inc.
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

package api

import (
	"testing"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	peerchooser "go.uber.org/yarpc/peer"
)

// ClientStreamRequestOpts are configuration options for a yarpc stream request.
type ClientStreamRequestOpts struct {
	Port          uint16
	GiveRequest   *transport.StreamRequest
	StreamActions []ClientStreamAction
	WantErrMsgs   []string
	NewChooser    func(peer.Identifier, peer.Transport) (peer.Chooser, error)
}

// NewClientStreamRequestOpts initializes a ClientStreamRequestOpts struct.
func NewClientStreamRequestOpts() ClientStreamRequestOpts {
	return ClientStreamRequestOpts{
		GiveRequest: &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Caller:   "unknown",
				Encoding: transport.Encoding("raw"),
			},
		},
		NewChooser: func(id peer.Identifier, transport peer.Transport) (peer.Chooser, error) {
			return peerchooser.NewSingle(id, transport), nil
		},
	}
}

// ClientStreamRequestOption can be used to configure a request.
type ClientStreamRequestOption interface {
	ApplyClientStreamRequest(*ClientStreamRequestOpts)
}

// ClientStreamRequestOptionFunc converts a function into a ClientStreamRequestOption.
type ClientStreamRequestOptionFunc func(*ClientStreamRequestOpts)

// ApplyClientStreamRequest implements ClientStreamRequestOption.
func (f ClientStreamRequestOptionFunc) ApplyClientStreamRequest(opts *ClientStreamRequestOpts) {
	f(opts)
}

// ClientStreamAction is an action applied to a ClientStream.
type ClientStreamAction interface {
	ApplyClientStream(testing.TB, *transport.ClientStream)
}

// ClientStreamActionFunc converts a function into a StreamAction.
type ClientStreamActionFunc func(testing.TB, *transport.ClientStream)

// ApplyClientStream implements ClientStreamAction.
func (f ClientStreamActionFunc) ApplyClientStream(t testing.TB, c *transport.ClientStream) { f(t, c) }
