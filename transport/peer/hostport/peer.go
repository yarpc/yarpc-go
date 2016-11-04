package hostport

import (
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

// NewPeerIdentifier creates a new hostport.PeerIdentifier
func NewPeerIdentifier(hostport string) PeerIdentifier {
	return PeerIdentifier(hostport)
}

// PeerIdentifier uniquely references a host:port combination using a common interface
type PeerIdentifier string

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p PeerIdentifier) Identifier() string {
	return string(p)
}

// NewPeer creates a new hostport.Peer from a hostport.PeerIdentifier, transport.Agent, and transport.PeerSubscriber
func NewPeer(pid PeerIdentifier, agent transport.PeerAgent, subscriber transport.PeerSubscriber) *Peer {
	return &Peer{
		PeerIdentifier:   pid,
		agent:            agent,
		subscriber:       subscriber,
		connectionStatus: transport.PeerAvailable,
	}
}

// Peer keeps a subscriber to send status updates to it, and the PeerAgent that created it
type Peer struct {
	PeerIdentifier

	agent            transport.PeerAgent
	subscriber       transport.PeerSubscriber
	pending          atomic.Int32
	connectionStatus transport.PeerConnectionStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the transport.Peer to a *hostport.Peer and run this function
func (p *Peer) HostPort() string {
	return string(p.PeerIdentifier)
}

// GetStatus returns the current status of the hostport.Peer
func (p *Peer) GetStatus() transport.PeerStatus {
	return transport.PeerStatus{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    p.connectionStatus,
	}
}

// GetAgent returns the Agent that is in charge of this hostport.Peer (and should be the one to handle requests)
func (p *Peer) GetAgent() transport.PeerAgent {
	return p.agent
}

// StartRequest runs at the beginning of a request and returns a callback for when the request finished
func (p *Peer) StartRequest() func() {
	p.pending.Inc()
	p.subscriber.NotifyStatusChanged(p)

	return p.endRequest
}

// endRequest should be run after a request has finished
func (p *Peer) endRequest() {
	p.pending.Dec()
	p.subscriber.NotifyStatusChanged(p)
}
