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

package grpc

import (
	"net"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/pkg/lifecycle"
)

// Transport is a grpc transport.Transport.
//
// This currently does not have any additional functionality over creating
// an Inbound or Outbound separately, but may in the future.
type Transport struct {
	lock             sync.Mutex
	once             *lifecycle.Once
	transportOptions *transportOptions
	peers            map[string]*grpcPeer
}

// NewTransport returns a new Transport.
func NewTransport(options ...TransportOption) *Transport {
	return &Transport{
		sync.Mutex{},
		lifecycle.NewOnce(),
		newTransportOptions(options),
		make(map[string]*grpcPeer),
	}
}

// Start implements transport.Lifecycle#Start.
func (t *Transport) Start() error {
	return t.once.Start(nil)
}

// Stop implements transport.Lifecycle#Stop.
func (t *Transport) Stop() error {
	// TODO release all remaining retained peers (maybe?)
	return t.once.Stop(nil)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (t *Transport) IsRunning() bool {
	return t.once.IsRunning()
}

// NewInbound returns a new Inbound for the given listener.
func (t *Transport) NewInbound(listener net.Listener, options ...InboundOption) *Inbound {
	return newInbound(t, listener, options...)
}

func (t *Transport) NewOutbound(pc peer.Chooser) *Outbound {
	return newOutbound(t, pc)
}

// NewSingleOutbound returns a new Outbound for the given adrress.
func (t *Transport) NewSingleOutbound(address string, options ...OutboundOption) *Outbound {
	return newSingleOutbound(t, address, options...)
}

func (t *Transport) RetainPeer(pid peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	address := pid.Identifier()
	peer, err := t.getPeer(address)
	if err != nil {
		return nil, err
	}
	peer.Subscribe(ps)
	return peer, nil
}

func (t *Transport) ReleasePeer(pid peer.Identifier, ps peer.Subscriber) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// TODO unsubscribe, then remove from map if no further subscribers, and of
	// course close the client conn before you leave
	return nil
}

func (t *Transport) getPeer(address string) (*grpcPeer, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	peer, ok := t.peers[address]
	if !ok {
		peer, err := newPeer(address, t)
		if err != nil {
			return nil, err
		}
		t.peers[address] = peer
	}

	return peer, nil
}
