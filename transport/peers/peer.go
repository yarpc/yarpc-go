package peers

import (
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

// NewPeer creates a new HostPortPeer from a PeerIdentifier, Agent, and Subscriber
func NewPeer(pid transport.PeerIdentifier, agent transport.PeerAgent, subscriber transport.PeerSubscriber) *HostPortPeer {
	hppid := pid.(*HostPortPeerIdentifier)
	return &HostPortPeer{
		HostPortPeerIdentifier: HostPortPeerIdentifier{
			hostport: hppid.hostport,
		},
		agent:            agent,
		subscriber:       subscriber,
		pending:          atomic.NewInt32(0),
		connectionStatus: transport.PeerAvailable,
	}
}

// HostPortPeer keeps a subscriber to send status updates to it, and the PeerAgent that created it
type HostPortPeer struct {
	HostPortPeerIdentifier

	agent            transport.PeerAgent
	subscriber       transport.PeerSubscriber
	pending          *atomic.Int32
	connectionStatus transport.PeerConnectionStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the Peer to a HostPortPeer and run this function
func (p *HostPortPeer) HostPort() string {
	return p.hostport
}

// GetStatus returns the current status (Available,Connecting,Unavailable) of the Peer
func (p *HostPortPeer) GetStatus() transport.PeerStatus {
	return transport.PeerStatus{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    p.connectionStatus,
	}
}

// GetAgent returns the Agent that is in charge of this Peer (and should be the one to handle requests)
func (p *HostPortPeer) GetAgent() transport.PeerAgent {
	return p.agent
}

// StartRequest runs at the beginning of a request and returns a callback for when the request finished
func (p *HostPortPeer) StartRequest() (finish func()) {
	p.pending.Inc()
	p.subscriber.NotifyPendingUpdate(p)

	finish = p.endRequest
	return finish
}

// endRequest should be run after a request has finished
func (p *HostPortPeer) endRequest() {
	p.pending.Dec()
	p.subscriber.NotifyPendingUpdate(p)
}
