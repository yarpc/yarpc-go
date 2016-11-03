package transport

//go:generate mockgen -destination=transporttest/peeragent.go -package=transporttest go.uber.org/yarpc/transport PeerAgent,PeerSubscriber

// PeerAgent manages Peers across different PeerSubscribers.  A PeerSubscriber will request a Peer for a specific
// PeerIdentifier and the PeerAgent has the ability to create a new Peer or return an existing one.
type PeerAgent interface {
	// Get or create a Peer for the PeerSubscriber
	RetainPeer(PeerIdentifier, PeerSubscriber) (Peer, error)

	// Unallocate a peer from the PeerSubscriber
	ReleasePeer(PeerIdentifier, PeerSubscriber) error
}

// PeerSubscriber listens to changes of a Peer over time.
type PeerSubscriber interface {
	// The Peer Notifies the PeerSubscriber that it can accept requests
	NotifyAvailable(Peer) error

	// The Peer Notifies the PeerSubscriber that it is setting up connections
	NotifyConnecting(Peer) error

	// The Peer Notifies the PeerSubscriber that it is ineligible for requests
	NotifyUnavailable(Peer) error

	// The Peer Notifies the PeerSubscriber when its pending request count changes (maybe to 0).
	NotifyPendingUpdate(Peer)
}
