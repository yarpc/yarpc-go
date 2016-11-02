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

	client *http.Client
	peers  map[string]*peers.HostPortPeer
}

// NewDefaultAgent creates an http agent with the default parameters
func NewDefaultAgent() *Agent {
	return NewAgent(&defaultConfig)
}

// NewAgent creates a new http agent for managing peers and sending requests
func NewAgent(cfg *outboundConfig) *Agent {
	return &Agent{
		client: buildClient(cfg),
		peers:  make(map[string]*peers.HostPortPeer),
	}
}

// GetClient gets the http client that should be used for making requests
func (a *Agent) GetClient() *http.Client {
	return a.client
}

// RetainPeer gets or creates a Peer for the specificed PeerList
func (a *Agent) RetainPeer(id transport.PeerIdentifier, list transport.PeerSubscriber) (transport.Peer, error) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	p := a.getOrCreatePeer(id)

	p.OnRetain(list)
	return p, nil
}

func (a *Agent) getOrCreatePeer(pid transport.PeerIdentifier) *peers.HostPortPeer {
	if p, ok := a.peers[pid.Identifier()]; ok {
		return p
	}

	p := peers.NewPeer(pid, a)

	a.peers[p.Identifier()] = p
	return p
}

// ReleasePeer removes a peer from the PeerList and removes that peer from the Agent if nothing is using it
func (a *Agent) ReleasePeer(id transport.PeerIdentifier, list transport.PeerSubscriber) error {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	p, ok := a.peers[id.Identifier()]
	if !ok {
		return errors.ErrAgentHasNoReferenceToPeer{
			Agent:          a,
			PeerIdentifier: id,
		}
	}

	err := p.OnRelease(list)
	if err != nil {
		return err
	}

	if p.References() == 0 {
		delete(a.peers, id.Identifier())
	}

	return nil
}
