// Copyright (c) 2016 Uber Technologies, Inc.
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
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"

	"github.com/opentracing/opentracing-go"
)

type transportConfig struct {
	keepAlive time.Duration
	tracer    opentracing.Tracer
}

var defaultTransportConfig = transportConfig{keepAlive: 30 * time.Second}

// TransportOption customizes the behavior of an HTTP transport.
type TransportOption func(*transportConfig)

// KeepAlive specifies the keep-alive period for the network connection. If
// zero, keep-alives are disabled.
//
// Defaults to 30 seconds.
func KeepAlive(t time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.keepAlive = t
	}
}

// Tracer configures a tracer for the transport and all its inbounds and
// outbounds.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(c *transportConfig) {
		c.tracer = tracer
	}
}

// NewTransport creates a new http transport for managing peers and sending requests
func NewTransport(opts ...TransportOption) *Transport {
	cfg := defaultTransportConfig
	cfg.tracer = opentracing.GlobalTracer()
	for _, o := range opts {
		o(&cfg)
	}

	return &Transport{
		client: buildClient(&cfg),
		peers:  make(map[string]*hostport.Peer),
		tracer: cfg.tracer,
	}
}

func buildClient(cfg *transportConfig) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// options lifted from https://golang.org/src/net/http/transport.go
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: cfg.keepAlive,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// Transport keeps track of http peers and the associated client with which the peer will call into.
type Transport struct {
	lock sync.Mutex

	client *http.Client
	peers  map[string]*hostport.Peer

	tracer opentracing.Tracer
}

// RetainPeer gets or creates a Peer for the specified peer.Subscriber (usually a peer.Chooser)
func (a *Transport) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	hppid, ok := pid.(hostport.PeerIdentifier)
	if !ok {
		return nil, peer.ErrInvalidPeerType{
			ExpectedType:   "hostport.PeerIdentifier",
			PeerIdentifier: pid,
		}
	}

	p, err := a.getOrCreatePeer(hppid)
	p.Subscribe(sub)
	return p, err
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Transport) getOrCreatePeer(pid hostport.PeerIdentifier) *hostport.Peer {
	if p, ok := a.peers[pid.Identifier()]; ok {
		return pl
	}

	p := hostport.NewPeer(pid, a)
	p.SetStatus(peer.Available)

	a.peers[p.Identifier()] = p

	return p
}

// ReleasePeer releases a peer from the peer.Subscriber and removes that peer from the Transport if nothing is listening to it
func (a *Transport) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	p, ok := a.peers[pid.Identifier()]
	if !ok {
		return peer.ErrTransportHasNoReferenceToPeer{
			TransportName:  "http.Transport",
			PeerIdentifier: pid.Identifier(),
		}
	}

	if err := p.Unsubscribe(sub); err != nil {
		return err
	}

	if p.NumSubscribers() == 0 {
		delete(a.peers, pid.Identifier())
	}

	return nil
}
