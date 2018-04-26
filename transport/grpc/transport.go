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

package grpc

import (
	"net"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/pkg/lifecycle"
)

// Transport is a grpc transport.Transport.
//
// This currently does not have any additional functionality over creating
// an Inbound or Outbound separately, but may in the future.
type Transport struct {
	lock          sync.Mutex
	once          *lifecycle.Once
	options       *transportOptions
	addressToPeer map[string]*grpcPeer
}

// NewTransport returns a new Transport.
func NewTransport(options ...TransportOption) *Transport {
	return newTransport(newTransportOptions(options))
}

func newTransport(transportOptions *transportOptions) *Transport {
	return &Transport{
		once:          lifecycle.NewOnce(),
		options:       transportOptions,
		addressToPeer: make(map[string]*grpcPeer),
	}
}

// Start implements transport.Lifecycle#Start.
func (t *Transport) Start() error {
	return t.once.Start(nil)
}

// Stop implements transport.Lifecycle#Stop.
func (t *Transport) Stop() error {
	return t.once.Stop(func() error {
		t.lock.Lock()
		defer t.lock.Unlock()
		for _, grpcPeer := range t.addressToPeer {
			grpcPeer.stop()
		}
		var err error
		for _, grpcPeer := range t.addressToPeer {
			err = multierr.Append(err, grpcPeer.wait())
		}
		return err
	})
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (t *Transport) IsRunning() bool {
	return t.once.IsRunning()
}

// NewInbound returns a new Inbound for the given listener.
func (t *Transport) NewInbound(listener net.Listener, options ...InboundOption) *Inbound {
	return newInbound(t, listener, options...)
}

// NewSingleOutbound returns a new Outbound for the given adrress.
func (t *Transport) NewSingleOutbound(address string, options ...OutboundOption) *Outbound {
	return newSingleOutbound(t, address, options...)
}

// NewOutbound returns a new Outbound for the given peer.Chooser.
func (t *Transport) NewOutbound(peerChooser peer.Chooser, options ...OutboundOption) *Outbound {
	return newOutbound(t, peerChooser, options...)
}

// RetainPeer retains the peer.
func (t *Transport) RetainPeer(peerIdentifier peer.Identifier, peerSubscriber peer.Subscriber) (peer.Peer, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	address := peerIdentifier.Identifier()
	p, ok := t.addressToPeer[address]
	if !ok {
		var err error
		p, err = newPeer(address, t)
		if err != nil {
			return nil, err
		}
		t.addressToPeer[address] = p
	}
	p.Subscribe(peerSubscriber)
	return p, nil
}

// ReleasePeer releases the peer.
func (t *Transport) ReleasePeer(peerIdentifier peer.Identifier, peerSubscriber peer.Subscriber) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	address := peerIdentifier.Identifier()
	p, ok := t.addressToPeer[address]
	if !ok {
		return peer.ErrTransportHasNoReferenceToPeer{
			TransportName:  "grpc.Transport",
			PeerIdentifier: address,
		}
	}
	if err := p.Unsubscribe(peerSubscriber); err != nil {
		return err
	}
	if p.NumSubscribers() == 0 {
		delete(t.addressToPeer, address)
		p.stop()
		return p.wait()
	}
	return nil
}
