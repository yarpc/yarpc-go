// Copyright (c) 2017 Uber Technologies, Inc.
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

package yarpctest

// This file provides fake implementations of most YARPC building blocks for
// the purpose of testing configuration using custom transports, choosers, and
// binders..

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/peer/hostport"
)

type FakeTransportOption func(*FakeTransport)

func FakeTransportAddress(addr string) FakeTransportOption {
	return func(t *FakeTransport) {
		t.Address = addr
	}
}

// NewFakeTransport returns a fake transport.
func NewFakeTransport(opts ...FakeTransportOption) *FakeTransport {
	t := &FakeTransport{
		Lifecycle: intsync.NewNopLifecycle(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// FakeTransport is a fake transport.
type FakeTransport struct {
	transport.Lifecycle
	address string
}

// RetainPeer returns a fake peer.
func (t *FakeTransport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	return &FakePeer{id: id.(hostport.PeerIdentifier)}, nil
}

// ReleasePeer does nothing.
func (t *FakeTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return nil
}

// FakePeer is a fake peer with an identifier.
type FakePeer struct {
	id hostport.PeerIdentifier
}

// Identifier returns the fake peer identifier.
func (p *FakePeer) Identifier() string {
	return string(p.id)
}

// Status returns the fake peer status.
func (p *FakePeer) Status() peer.Status {
	return peer.Status{
		ConnectionStatus:    peer.Available,
		PendingRequestCount: 0,
	}
}

// StartRequest does nothing.
func (p *FakePeer) StartRequest() {
}

// EndRequest does nothing.
func (p *FakePeer) EndRequest() {
}
