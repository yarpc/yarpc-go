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

package yarpctest

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/lifecycletest"
)

// FakeTransportOption is an option for NewFakeTransport.
type FakeTransportOption func(*FakeTransport)

// NopTransportOption returns a no-op option for NewFakeTransport.
// The option exists to verify that options work.
func NopTransportOption(nopOption string) FakeTransportOption {
	return func(t *FakeTransport) {
		t.nopOption = nopOption
	}
}

// NewFakeTransport returns a fake transport.
func NewFakeTransport(opts ...FakeTransportOption) *FakeTransport {
	t := &FakeTransport{
		Lifecycle: lifecycletest.NewNop(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// FakeTransport is a fake transport.
type FakeTransport struct {
	transport.Lifecycle
	nopOption string
}

// NopOption returns the configured nopOption. It's fake.
func (t *FakeTransport) NopOption() string {
	return t.nopOption
}

// RetainPeer returns a fake peer.
func (t *FakeTransport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	return &FakePeer{id: id.(hostport.PeerIdentifier)}, nil
}

// ReleasePeer does nothing.
func (t *FakeTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return nil
}
