package hostport

import (
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

// PeerIdentifier uniquely references a host:port combination using a common interface
type PeerIdentifier string

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p PeerIdentifier) Identifier() string {
	return string(p)
}

// NewPeer creates a new hostport.Peer from a hostport.PeerIdentifier, transport.Agent, and transport.PeerSubscriber
func NewPeer(pid PeerIdentifier, agent transport.Agent) *Peer {
	return &Peer{
		PeerIdentifier:   pid,
		agent:            agent,
		connectionStatus: transport.PeerUnavailable,
	}
}

// Peer keeps a subscriber to send status updates to it, and the PeerAgent that created it
type Peer struct {
	PeerIdentifier

	agent            transport.Agent
	pending          atomic.Int32
	connectionStatus transport.PeerConnectionStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the transport.Peer to a *hostport.Peer and run this function
func (p *Peer) HostPort() string {
	return string(p.PeerIdentifier)
}

// Status returns the current status of the hostport.Peer
func (p *Peer) Status() transport.PeerStatus {
	return transport.PeerStatus{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    p.connectionStatus,
	}
}

// SetStatus sets the status of the Peer (to be used by the Agent)
func (p *Peer) SetStatus(status transport.PeerConnectionStatus) {
	p.connectionStatus = status
}

// Agent returns the Agent that is in charge of this hostport.Peer (and should be the one to handle requests)
func (p *Peer) Agent() transport.Agent {
	return p.agent
}

// StartRequest runs at the beginning of a request and returns a callback for when the request finished
func (p *Peer) StartRequest() func() {
	p.pending.Inc()
	p.agent.NotifyStatusChanged(p)

	return p.endRequest
}

// endRequest should be run after a request has finished
func (p *Peer) endRequest() {
	p.pending.Dec()
	p.agent.NotifyStatusChanged(p)
}
