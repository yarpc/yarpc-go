package transport

//go:generate mockgen -destination=transporttest/peer.go -package=transporttest go.uber.org/yarpc/transport PeerIdentifier,Peer

// PeerConnectionStatus maintains information about the Peer's connection state
type PeerConnectionStatus int

const (
	// PeerAvailable indicates the Peer is available for requests
	PeerAvailable PeerConnectionStatus = iota + 1

	// PeerConnecting indicates the Peer is in the process of connecting
	PeerConnecting

	// PeerUnavailable indicates the Peer is unavailable for requests
	PeerUnavailable
)

// PeerStatus holds all the information about a peer's state that would be useful to PeerSubscribers
type PeerStatus struct {
	// Current number of pending requests on this peer
	PendingRequestCount int

	// Current status of the Peer's connection
	ConnectionStatus PeerConnectionStatus
}

// PeerIdentifier is able to uniquely identify a peer (e.g. hostport)
type PeerIdentifier interface {
	Identifier() string
}

// Peer is a level on top of PeerIdentifier.  It should be created by a PeerAgent so we
// can maintain multiple references to the same downstream peer (e.g. hostport).  This is
// useful for load balancing requests to downstream services.
type Peer interface {
	PeerIdentifier

	// Get the status of the Peer
	GetStatus() PeerStatus

	// Tell the peer that a request is starting/ending
	// The callsite should look like:
	//   done := peer.StartRequest()
	//   defer done()
	//   // Do request
	StartRequest() (finish func())
}
