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

package http

import (
	"go.uber.org/yarpc/api/peer"
)

// NewDialer creates a transport that is decorated to optionally isolate
// retained peers from other dialers.
func (t *Transport) NewDialer() *Dialer {
	return &Dialer{trans: t}
}

// Dialer is a decorator for an HTTP transport that can isolate its peers,
// and therefore connections, from other dialers.
type Dialer struct {
	trans           *Transport
	connectionScope *connectionScope
}

var _ peer.Transport = (*Dialer)(nil)

// WithConnectionIsolation returns a copy of the Dialer whose peers, and
// therefore connections or connection pools, are not shared with other
// dialers retaining the same address.
//
// Callers should create one isolated Dialer per logical outbound. Requests
// within that outbound continue to share peers normally.
func (d *Dialer) WithConnectionIsolation() *Dialer {
	isolated := *d
	isolated.connectionScope = new(connectionScope)
	return &isolated
}

// RetainPeer retains the identified peer.
func (d *Dialer) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	return d.trans.retainPeer(id, d.connectionScope, ps)
}

// ReleasePeer releases the identified peer.
func (d *Dialer) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return d.trans.releasePeer(id, d.connectionScope, ps)
}
