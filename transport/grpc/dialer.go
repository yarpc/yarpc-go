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

package grpc

import (
	"go.uber.org/yarpc/api/peer"
)

// NewDialer creates a transport that is decorated to retain peers with
// additional gRPC dial options.
func (t *Transport) NewDialer(options ...DialOption) *Dialer {
	return &Dialer{trans: t, options: newDialOptions(options)}
}

// Dialer is a decorator for a gRPC transport that threads dial options for
// every retained peer.
type Dialer struct {
	trans   *Transport
	options *dialOptions
}

var _ peer.Transport = (*Dialer)(nil)

// RetainPeer retains the identified peer, passing dial options.
func (d *Dialer) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	return d.trans.retainPeer(id, d.options, ps)
}

// ReleasePeer releases the identified peer.
func (d *Dialer) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return d.trans.ReleasePeer(id, ps)
}
