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

var defaultConfig = agentConfig{keepAlive: 30 * time.Second}

// Agent keeps track of http peers and the associated client with which the peer will call into.
type Agent struct {
	sync.Mutex

	client    *http.Client
	peerNodes map[string]*peerNode
}

// peerNode keeps track of a HostPortPeer and any subscribers referencing it
type peerNode struct {
	peer       *hostport.Peer
	references map[transport.PeerSubscriber]struct{}
}

// NewDefaultAgent creates an http agent with the default parameters
func NewDefaultAgent() *Agent {
	return NewAgent(&defaultConfig)
}

// NewAgent creates a new http agent for managing peers and sending requests
func NewAgent(cfg *agentConfig) *Agent {
	return &Agent{
		client:    buildClient(cfg),
		peerNodes: make(map[string]*peerNode),
	}
}

// GetClient gets the http client that should be used for making requests
func (a *Agent) GetClient() *http.Client {
	return a.client
}

// RetainPeer gets or creates a Peer for the specificed PeerList
func (a *Agent) RetainPeer(pid transport.PeerIdentifier, sub transport.PeerSubscriber) (transport.Peer, error) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	hppid, ok := pid.(hostport.PeerIdentifier)
	if !ok {
		return nil, errors.ErrInvalidPeerType{
			ExpectedType:   "hostport.PeerIdentifier",
			PeerIdentifier: pid,
		}
	}

	node := a.getOrCreatePeerNode(hppid)

	node.references[sub] = struct{}{}

	return node.peer, nil
}

func (a *Agent) getOrCreatePeerNode(pid hostport.PeerIdentifier) *peerNode {
	if node, ok := a.peerNodes[pid.Identifier()]; ok {
		return node
	}

	peer := hostport.NewPeer(pid, a, a)
	node := &peerNode{
		peer:       peer,
		references: make(map[transport.PeerSubscriber]struct{}),
	}
	a.peerNodes[peer.Identifier()] = node

	return node
}

// ReleasePeer releases a peer from the PeerSubscriber and removes that peer from the Agent if nothing is listening to it
func (a *Agent) ReleasePeer(pid transport.PeerIdentifier, sub transport.PeerSubscriber) error {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	node, ok := a.peerNodes[pid.Identifier()]
	if !ok {
		return errors.ErrAgentHasNoReferenceToPeer{
			Agent:          a,
			PeerIdentifier: pid,
		}
	}

	_, ok = node.references[sub]
	if !ok {
		return errors.ErrPeerHasNoReferenceToSubscriber{
			PeerIdentifier: pid,
			PeerSubscriber: sub,
		}
	}

	delete(node.references, sub)

	if len(node.references) == 0 {
		delete(a.peerNodes, pid.Identifier())
	}

	return nil
}

// NotifyStatusChanged Notifies peer subscribers the peer's status changes.
func (a *Agent) NotifyStatusChanged(peer transport.Peer) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	node, ok := a.peerNodes[peer.Identifier()]
	if !ok {
		// The peer has probably been released already and this is a request finishing, ignore
		return
	}

	for sub := range node.references {
		sub.NotifyStatusChanged(peer)
	}
}
