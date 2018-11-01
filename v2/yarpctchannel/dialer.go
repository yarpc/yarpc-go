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

package yarpctchannel

import (
	"context"
	"fmt"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcbackoff"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/zap"
)

// Dialer is a TChannel connection dialer.
type Dialer struct {
	// Caller is the name of this, the calling service, for purposes of
	// logging and establishing the caller name.
	//
	// This field is required.
	Caller string

	// ConnTimeout specifies the time that TChannel will wait for a connection
	// attempt to any retained peer.
	//
	// The default is half of a second.
	ConnTimeout time.Duration

	// ConnBackoff specifies the connection backoff strategy for delays between
	// connection attempts for each peer.
	//
	// ConnBackoff accepts a function that creates new backoff instances.  The
	// dialer uses this to make referentially independent backoff instances
	// that will not be shared across goroutines.
	//
	// The backoff instance is a function that accepts connection attempts and
	// returns a duration.
	//
	// The default is exponential backoff starting with 10ms fully jittered,
	// doubling each attempt, with a maximum interval of 30s.
	ConnBackoff yarpc.BackoffStrategy

	// Tracer specifies the request tracer used for RPCs passing through the
	// TChannel outbound.
	Tracer opentracing.Tracer

	// Logger sets a logger to use for internal logging.
	//
	// The default is to not write any logs.
	Logger *zap.Logger

	lock            sync.Mutex
	ch              *tchannel.Channel
	peers           map[string]*tchannelPeer
	connectorsGroup sync.WaitGroup
}

var _ yarpc.Dialer = (*Dialer)(nil)

// Start starts the TChannel dialer.
func (d *Dialer) Start(ctx context.Context) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	chopts := tchannel.ChannelOptions{
		Tracer:              d.Tracer,
		OnPeerStatusChanged: d.onPeerStatusChanged,
	}
	ch, err := tchannel.NewChannel(d.Caller, &chopts)
	if err != nil {
		// This should be unreachable since it indicates invalid channel
		// options.
		return err
	}
	d.ch = ch

	d.peers = make(map[string]*tchannelPeer)

	if d.ConnTimeout == 0 {
		d.ConnTimeout = DefaultConnTimeout
	}
	if d.ConnBackoff == nil {
		d.ConnBackoff = yarpcbackoff.DefaultExponential
	}

	return nil
}

// Stop stops the TChannel transport. It starts rejecting incoming requests
// and draining connections before closing them.
// In a future version of YARPC, Stop may block until the underlying channel
// has drained all requests.
func (d *Dialer) Stop(ctx context.Context) error {
	// Release all peers.
	for _, peer := range d.peers {
		peer.Release()
	}
	d.ch.Close()
	d.connectorsGroup.Wait()
	// TODO wait for all inbound requests to drain or context to cancel.
	return nil
}

// RetainPeer adds a peer subscriber (typically a peer chooser) and causes the
// transport to maintain persistent connections with that peer.
//
// RetainPeer must be called while the dialer is running.
func (d *Dialer) RetainPeer(pid yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// TODO checking for a non-nil channel is barely sufficient.
	// The purpose of this check is to surface improper use of the API during development.
	// We have deliberately avoided outright preventing misuse of the API,
	// which would entail initialization checks in every function or an API that only
	// grants access to capability bearing instances when it is safe to use them.
	if d.ch == nil {
		return nil, fmt.Errorf("Dialer must be started to retain peers")
	}

	p := d.getOrCreatePeer(pid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (d *Dialer) getOrCreatePeer(pid yarpc.Identifier) *tchannelPeer {
	addr := pid.Identifier()
	if p, ok := d.peers[addr]; ok {
		return p
	}

	p := newPeer(pid, addr, d)
	d.peers[addr] = p
	// Start a peer connection loop
	d.connectorsGroup.Add(1)
	go p.MaintainConn()

	return p
}

// ReleasePeer releases a peer from the yarpc.Subscriber and removes that peer
// from the Dialer if nothing is listening to it.
func (d *Dialer) ReleasePeer(pid yarpc.Identifier, sub yarpc.Subscriber) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	p, ok := d.peers[pid.Identifier()]
	if !ok {
		// This is only reachable if there is a bug in a peer list.
		return yarpcpeer.ErrDialerHasNoReferenceToPeer{
			DialerName:     "tchannel.Dialer",
			PeerIdentifier: pid.Identifier(),
		}
	}

	if err := p.Unsubscribe(sub); err != nil {
		// This is unreachable. AbstractPeer.Unsubscribe has no error-return path.
		return err
	}

	if p.NumSubscribers() == 0 {
		// Release the peer so that the connection retention loop stops.
		p.Release()
		delete(d.peers, pid.Identifier())
	}

	return nil
}

// onPeerStatusChanged receives notifications from TChannel Channel when any
// peer's status changes.
func (d *Dialer) onPeerStatusChanged(tp *tchannel.Peer) {
	d.lock.Lock()
	defer d.lock.Unlock()

	p, ok := d.peers[tp.HostPort()]
	if !ok {
		return
	}
	p.OnStatusChanged()
}
