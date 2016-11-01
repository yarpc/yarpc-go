package transport

//go:generate mockgen -destination=transporttest/peeragent.go -package=transporttest go.uber.org/yarpc/transport PeerAgent

/*
PeerAgent represents an interface for managing Peers across different outbounds.  A PeerList will
request a Peer for a specific PeerIdentifier and the PeerAgent has the ability to create a new Peer or
return an existing one.
*/
type PeerAgent interface {
	RetainPeer(PeerIdentifier, PeerSubscriber) (Peer, error) // Get or create a Peer for the PeerSubscriber
	ReleasePeer(PeerIdentifier, PeerSubscriber) error        // Unallocate a peer from the PeerSubscriber
}
