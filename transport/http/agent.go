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
		client:    buildClient(&cfg),
		peerNodes: make(map[string]*peerNode),
	}
}

// Agent keeps track of http peers and the associated client with which the peer will call into.
type Agent struct {
	lock sync.RWMutex

	client    *http.Client
	peerNodes map[string]*peerNode
}

// peerNode keeps track of a HostPortPeer and any subscribers retaining it
type peerNode struct {
	peer        *hostport.Peer
	subscribers map[transport.PeerSubscriber]struct{}
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

	node := a.getOrCreatePeerNode(hppid)
	node.subscribers[sub] = struct{}{}
	return node.peer, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Agent) getOrCreatePeerNode(pid hostport.PeerIdentifier) *peerNode {
	if node, ok := a.peerNodes[pid.Identifier()]; ok {
		return node
	}

	peer := hostport.NewPeer(pid, a)
	peer.SetStatus(transport.PeerAvailable)

	node := &peerNode{
		peer:        peer,
		subscribers: make(map[transport.PeerSubscriber]struct{}),
	}
	a.peerNodes[peer.Identifier()] = node

	return node
}

// ReleasePeer releases a peer from the PeerSubscriber and removes that peer from the Agent if nothing is listening to it
func (a *Agent) ReleasePeer(pid transport.PeerIdentifier, sub transport.PeerSubscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	node, ok := a.peerNodes[pid.Identifier()]
	if !ok {
		return errors.ErrAgentHasNoReferenceToPeer{
			Agent:          a,
			PeerIdentifier: pid,
		}
	}

	if _, ok = node.subscribers[sub]; !ok {
		return errors.ErrPeerHasNoReferenceToSubscriber{
			PeerIdentifier: pid,
			PeerSubscriber: sub,
		}
	}

	delete(node.subscribers, sub)

	if len(node.subscribers) == 0 {
		delete(a.peerNodes, pid.Identifier())
	}

	return nil
}

// NotifyStatusChanged Notifies peer subscribers the peer's status changes.
func (a *Agent) NotifyStatusChanged(peer transport.Peer) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	node, ok := a.peerNodes[peer.Identifier()]
	if !ok {
		// The peer has probably been released already and this is a request finishing, ignore
		return
	}

	for sub := range node.subscribers {
		sub.NotifyStatusChanged(peer)
	}
}
