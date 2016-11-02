package peers

import (
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

// NewPeerIdentifier creates a new HostPortPeerIdentifier
func NewPeerIdentifier(hostport string) *HostPortPeerIdentifier {
	return &HostPortPeerIdentifier{
		hostport: hostport,
	}
}

// HostPortPeerIdentifier uniquely references a host:port combination using a common interface
type HostPortPeerIdentifier struct {
	hostport string
}

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p *HostPortPeerIdentifier) Identifier() string {
	return p.hostport
}

// NewPeer creates a new HostPortPeer from a PeerIdentifier and an Agent
func NewPeer(pid transport.PeerIdentifier, agent transport.PeerAgent) *HostPortPeer {
	hppid := pid.(*HostPortPeerIdentifier)
	return &HostPortPeer{
		HostPortPeerIdentifier: HostPortPeerIdentifier{
			hostport: hppid.hostport,
		},
		agent:      agent,
		references: make(map[transport.PeerSubscriber]bool),
		pending:    atomic.NewInt32(0),
		status:     transport.PeerAvailable,
	}
}

// HostPortPeer keeps a list of references to PeerSubscribers that reference it, and the PeerAgent that created it
// as well
type HostPortPeer struct {
	HostPortPeerIdentifier

	agent      transport.PeerAgent
	references map[transport.PeerSubscriber]bool
	pending    *atomic.Int32
	status     transport.PeerStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the Peer to a HostPortPeer and run this function
func (p *HostPortPeer) HostPort() string {
	return p.hostport
}

// GetStatus returns the current status (Available,Connecting,Unavailable) of the Peer
func (p *HostPortPeer) GetStatus() transport.PeerStatus {
	return p.status
}

// GetAgent returns the Agent that is in charge of this Peer (and should be the one to handle requests)
func (p *HostPortPeer) GetAgent() transport.PeerAgent {
	return p.agent
}

// OnRetain informs the peer that a new PeerSubscriber has begun listening to the events on this peer
func (p *HostPortPeer) OnRetain(sub transport.PeerSubscriber) error {
	p.references[sub] = true
	return nil
}

// OnRelease informs the peer that a PeerSubscriber has stopped listening to the events on this peer
func (p *HostPortPeer) OnRelease(sub transport.PeerSubscriber) error {
	_, ok := p.references[sub]
	if !ok {
		return errors.ErrPeerHasNoReferenceToSubscriber{
			Peer:           p,
			PeerSubscriber: sub,
		}
	}
	delete(p.references, sub)
	return nil
}

// References returns the number of subscribers currently referencing this Peer
func (p *HostPortPeer) References() int {
	return len(p.references)
}

// Pending returns the number of pending requests going to this peer
func (p *HostPortPeer) Pending() int {
	return int(p.pending.Load())
}

// IncPending increments the number of Pending requests on this peer
func (p *HostPortPeer) IncPending() {
	p.pending.Inc()
}

// DecPending decrements the number of Pending requests on this peer
func (p *HostPortPeer) DecPending() {
	p.pending.Dec()
}
