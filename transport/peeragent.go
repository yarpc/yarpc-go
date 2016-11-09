package transport

//go:generate mockgen -destination=transporttest/peeragent.go -package=transporttest go.uber.org/yarpc/transport Agent,PeerSubscriber

// PeerSubscriber listens to changes of a Peer over time.
type PeerSubscriber interface {
	// The Peer Notifies the PeerSubscriber when its status changes (e.g. connections status, pending requests)
	NotifyStatusChanged(Peer)
}

// Agent manages Peers across different PeerSubscribers.  A PeerSubscriber will request a Peer for a specific
// PeerIdentifier and the Agent has the ability to create a new Peer or return an existing one.
type Agent interface {
	PeerSubscriber

	// Get or create a Peer for the PeerSubscriber
	RetainPeer(PeerIdentifier, PeerSubscriber) (Peer, error)

	// Unallocate a peer from the PeerSubscriber
	ReleasePeer(PeerIdentifier, PeerSubscriber) error
}
