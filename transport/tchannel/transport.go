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

package tchannel

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

type headerCase int

const (
	canonicalizedHeaderCase headerCase = iota
	originalHeaderCase
)

// Transport is a TChannel transport suitable for use with YARPC's peer
// selection system.
// The transport implements peer.Transport so multiple peer.List
// implementations can retain and release shared peers.
// The transport implements transport.Transport so it is suitable for lifecycle
// management.
type Transport struct {
	lock sync.Mutex
	once *lifecycle.Once

	ch       *tchannel.Channel
	router   transport.Router
	tracer   opentracing.Tracer
	logger   *zap.Logger
	name     string
	addr     string
	listener net.Listener

	connTimeout            time.Duration
	initialConnRetryDelay  time.Duration
	connRetryBackoffFactor int
	connectorsGroup        sync.WaitGroup
	connBackoffStrategy    backoffapi.Strategy
	headerCase             headerCase

	peers map[string]*tchannelPeer
}

// NewTransport is a YARPC transport that facilitates sending and receiving
// YARPC requests through TChannel.
// It uses a shared TChannel Channel for both, incoming and outgoing requests,
// ensuring reuse of connections and other resources.
//
// Either the local service name (with the ServiceName option) or a user-owned
// TChannel (with the WithChannel option) MUST be specified.
func NewTransport(opts ...TransportOption) (*Transport, error) {
	options := newTransportOptions()

	for _, opt := range opts {
		opt(&options)
	}

	if options.ch != nil {
		return nil, fmt.Errorf("NewTransport does not accept WithChannel, use NewChannelTransport")
	}

	return options.newTransport(), nil
}

func (o transportOptions) newTransport() *Transport {
	logger := o.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	headerCase := canonicalizedHeaderCase
	if o.originalHeaders {
		headerCase = originalHeaderCase
	}
	return &Transport{
		once:                lifecycle.NewOnce(),
		name:                o.name,
		addr:                o.addr,
		listener:            o.listener,
		connTimeout:         o.connTimeout,
		connBackoffStrategy: o.connBackoffStrategy,
		peers:               make(map[string]*tchannelPeer),
		tracer:              o.tracer,
		logger:              logger,
		headerCase:          headerCase,
	}
}

// ListenAddr exposes the listen address of the transport.
func (t *Transport) ListenAddr() string {
	return t.addr
}

// RetainPeer adds a peer subscriber (typically a peer chooser) and causes the
// transport to maintain persistent connections with that peer.
func (t *Transport) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	p := t.getOrCreatePeer(pid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (t *Transport) getOrCreatePeer(pid peer.Identifier) *tchannelPeer {
	addr := pid.Identifier()
	if p, ok := t.peers[addr]; ok {
		return p
	}

	p := newPeer(addr, t)
	t.peers[addr] = p
	// Start a peer connection loop
	t.connectorsGroup.Add(1)
	go p.MaintainConn()

	return p
}

// ReleasePeer releases a peer from the peer.Subscriber and removes that peer
// from the Transport if nothing is listening to it.
func (t *Transport) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	p, ok := t.peers[pid.Identifier()]
	if !ok {
		return peer.ErrTransportHasNoReferenceToPeer{
			TransportName:  "tchannel.Transport",
			PeerIdentifier: pid.Identifier(),
		}
	}

	if err := p.Unsubscribe(sub); err != nil {
		return err
	}

	if p.NumSubscribers() == 0 {
		// Release the peer so that the connection retention loop stops.
		p.Release()
		delete(t.peers, pid.Identifier())
	}

	return nil
}

func (t *Transport) peerList() *tchannel.RootPeerList {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.ch == nil {
		return nil
	}

	return t.ch.RootPeers()
}

// Start starts the TChannel transport. This starts making connections and
// accepting inbound requests. All inbounds must have been assigned a router
// to accept inbound requests before this is called.
func (t *Transport) Start() error {
	return t.once.Start(t.start)
}

func (t *Transport) start() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	chopts := tchannel.ChannelOptions{
		Tracer: t.tracer,
		Handler: handler{
			router:            t.router,
			tracer:            t.tracer,
			headerCase:        t.headerCase,
			logger:            t.logger,
			newResponseWriter: newHandlerWriter,
		},
		OnPeerStatusChanged: t.onPeerStatusChanged,
	}
	ch, err := tchannel.NewChannel(t.name, &chopts)
	if err != nil {
		return err
	}
	t.ch = ch

	if t.listener != nil {
		if err := t.ch.Serve(t.listener); err != nil {
			return err
		}
	} else {
		// Default to ListenIP if addr wasn't given.
		addr := t.addr
		if addr == "" {
			listenIP, err := tchannel.ListenIP()
			if err != nil {
				return err
			}

			addr = listenIP.String() + ":0"
			// TODO(abg): Find a way to export this to users
		}

		// TODO(abg): If addr was just the port (":4040"), we want to use
		// ListenIP() + ":4040" rather than just ":4040".
		if err := t.ch.ListenAndServe(addr); err != nil {
			return err
		}
	}

	t.addr = t.ch.PeerInfo().HostPort

	return nil
}

// Stop stops the TChannel transport. It starts rejecting incoming requests
// and draining connections before closing them.
// In a future version of YARPC, Stop will block until the underlying channel
// has closed completely.
func (t *Transport) Stop() error {
	return t.once.Stop(t.stop)
}

func (t *Transport) stop() error {
	t.ch.Close()
	t.connectorsGroup.Wait()
	return nil
}

// IsRunning returns whether the TChannel transport is running.
func (t *Transport) IsRunning() bool {
	return t.once.IsRunning()
}

// onPeerStatusChanged receives notifications from TChannel Channel when any
// peer's status changes.
func (t *Transport) onPeerStatusChanged(tp *tchannel.Peer) {
	t.lock.Lock()
	defer t.lock.Unlock()

	p, ok := t.peers[tp.HostPort()]
	if !ok {
		return
	}
	p.OnStatusChanged()
}
