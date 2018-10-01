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
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENd. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package grpc

import (
	"context"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeer"
)

var _ yarpc.Dialer = (*Dialer)(nil)

// Dialer is a decorator for a gRPC transport that threads dial options for
// every retained peer.
type Dialer struct {
	lock          sync.Mutex
	options       *dialOptions
	addressToPeer map[string]*grpcPeer
}

// Start starts the gRPC dialer.
func (d *Dialer) Start(_ context.Context) error {
	d.addressToPeer = make(map[string]*grpcPeer)
	// init options?
	return nil
}

// Stop stops the gRPC dialer.
func (d *Dialer) Stop() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, grpcPeer := range d.addressToPeer {
		grpcPeer.stop()
	}
	var err error
	for _, grpcPeer := range d.addressToPeer {
		err = multierr.Append(err, grpcPeer.wait())
	}
	return err
}

// RetainPeer retains the identified peer, passing dial options.
func (d *Dialer) RetainPeer(id yarpc.Identifier, ps yarpc.Subscriber) (yarpc.Peer, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	address := id.Identifier()
	p, ok := d.addressToPeer[address]
	if !ok {
		var err error
		p, err = d.newPeer(address, d.options)
		if err != nil {
			return nil, err
		}
		d.addressToPeer[address] = p
	}
	p.Subscribe(ps)
	return p, nil
}

// ReleasePeer releases the identified peer.
func (d *Dialer) ReleasePeer(id yarpc.Identifier, ps yarpc.Subscriber) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	address := id.Identifier()
	p, ok := d.addressToPeer[address]
	if !ok {
		return yarpcpeer.ErrDialerHasNoReferenceToPeer{
			TransportName:  "grpc.Transport",
			PeerIdentifier: address,
		}
	}
	if err := p.Unsubscribe(ps); err != nil {
		return err
	}
	if p.NumSubscribers() == 0 {
		delete(d.addressToPeer, address)
		p.stop()
		return p.wait()
	}
	return nil
}
