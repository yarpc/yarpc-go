package http

import (
	"net/http"
	"sync"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/peers"
)

// Agent keeps track of http peers and the associated client with which the peer will call into.
type Agent struct {
	sync.Mutex

	client    *http.Client
	peerNodes map[string]*peerNode
}

// peerNode keeps track of a HostPortPeer and any subscribers referencing it
type peerNode struct {
	peer       *peers.HostPortPeer
	references map[transport.PeerSubscriber]bool
}

// NewDefaultAgent creates an http agent with the default parameters
func NewDefaultAgent() *Agent {
	return NewAgent(&defaultConfig)
}

// NewAgent creates a new http agent for managing peers and sending requests
func NewAgent(cfg *outboundConfig) *Agent {
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

	node := a.getOrCreatePeerNode(pid)

	node.references[sub] = true

	return node.peer, nil
}

func (a *Agent) getOrCreatePeerNode(pid transport.PeerIdentifier) *peerNode {
	if node, ok := a.peerNodes[pid.Identifier()]; ok {
		return node
	}

	peer := peers.NewPeer(pid, a, a)
	node := &peerNode{
		peer:       peer,
		references: make(map[transport.PeerSubscriber]bool),
	}
	a.peerNodes[peer.Identifier()] = node

	return node
}

// ReleasePeer releases a peer from the peersubscriber and removes that peer from the Agent if nothing is listening to it
func (a *Agent) ReleasePeer(id transport.PeerIdentifier, sub transport.PeerSubscriber) error {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	node, ok := a.peerNodes[id.Identifier()]
	if !ok {
		return errors.ErrAgentHasNoReferenceToPeer{
			Agent:          a,
			PeerIdentifier: id,
		}
	}

	_, ok = node.references[sub]
	if !ok {
		return errors.ErrPeerHasNoReferenceToSubscriber{
			Peer:           node.peer,
			PeerSubscriber: sub,
		}
	}

	delete(node.references, sub)

	if len(node.references) == 0 {
		delete(a.peerNodes, id.Identifier())
	}

	return nil
}

func (a *Agent) NotifyAvailable(peer transport.Peer) error {
	return nil
}

func (a *Agent) NotifyConnecting(peer transport.Peer) error {
	return nil
}

// The Peer Notifies the PeerSubscriber that it is ineligible for requests
func (a *Agent) NotifyUnavailable(peer transport.Peer) error {
	return nil
}

// The Peer Notifies the PeerSubscriber when its pending request count changes (maybe to 0).
func (a *Agent) NotifyPendingUpdate(peer transport.Peer) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	node, ok := a.peerNodes[peer.Identifier()]
	if !ok {
		// The peer has probably been released already and this is a request finishing, ignore
		return
	}

	for sub, _ := range node.references {
		sub.NotifyPendingUpdate(peer)
	}
}
