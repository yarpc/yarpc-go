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
	"net/http"
	"sync"
	"time"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
	"go.uber.org/yarpc/transport/peer/hostport"
)

type agentConfig struct {
	keepAlive time.Duration
}

var defaultAgentConfig = agentConfig{keepAlive: 30 * time.Second}

// AgentOption customizes the behavior of an HTTP agent.
type AgentOption func(*agentConfig)

// KeepAlive specifies the keep-alive period for the network connection. If
// zero, keep-alives are disabled.
//
// Defaults to 30 seconds.
func KeepAlive(t time.Duration) AgentOption {
	return func(c *agentConfig) {
		c.keepAlive = t
	}
}

// NewAgent creates a new http agent for managing peers and sending requests
func NewAgent(opts ...AgentOption) *Agent {
	cfg := defaultAgentConfig
	for _, o := range opts {
		o(&cfg)
	}

	return &Agent{
		client: buildClient(&cfg),
		peers:  make(map[string]*hostport.Peer),
	}
}

// Agent keeps track of http peers and the associated client with which the peer will call into.
type Agent struct {
	lock sync.RWMutex

	client *http.Client
	peers  map[string]*hostport.Peer
}

// RetainPeer gets or creates a Peer for the specified PeerSubscriber (usually a PeerList)
func (a *Agent) RetainPeer(pid transport.PeerIdentifier, sub transport.PeerSubscriber) (transport.Peer, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	hppid, ok := pid.(hostport.PeerIdentifier)
	if !ok {
		return nil, errors.ErrInvalidPeerType{
			ExpectedType:   "hostport.PeerIdentifier",
			PeerIdentifier: pid,
		}
	}

	peer := a.getOrCreatePeer(hppid)
	peer.AddSubscriber(sub)
	return peer, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Agent) getOrCreatePeer(pid hostport.PeerIdentifier) *hostport.Peer {
	if peer, ok := a.peers[pid.Identifier()]; ok {
		return peer
	}

	peer := hostport.NewPeer(pid, a)
	peer.SetStatus(transport.PeerAvailable)

	a.peers[peer.Identifier()] = peer

	return peer
}

// ReleasePeer releases a peer from the PeerSubscriber and removes that peer from the Agent if nothing is listening to it
func (a *Agent) ReleasePeer(pid transport.PeerIdentifier, sub transport.PeerSubscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	peer, ok := a.peers[pid.Identifier()]
	if !ok {
		return errors.ErrAgentHasNoReferenceToPeer{
			Agent:          a,
			PeerIdentifier: pid,
		}
	}

	if err := peer.RemoveSubscriber(sub); err != nil {
		return err
	}

	if peer.NumSubscribers() == 0 {
		delete(a.peers, pid.Identifier())
	}

	return nil
}
