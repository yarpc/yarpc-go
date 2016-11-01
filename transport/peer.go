package transport

//go:generate mockgen -destination=transporttest/peer.go -package=transporttest go.uber.org/yarpc/transport PeerIdentifier,Peer,SubscribablePeer,PeerSubscriber

// PeerStatus maintains information about the Peer's connection state
type PeerStatus string

const (
	// PeerAvailable indicates the Peer is available for requests
	PeerAvailable PeerStatus = "available"
	// PeerConnecting indicates the Peer is in the process of connecting
	PeerConnecting PeerStatus = "connecting"
	// PeerUnavailable indicates the Peer is unavailable for requests
	PeerUnavailable PeerStatus = "unavailable"
)

/*
PeerIdentifier is able to uniquely identify a peer (e.g. hostport)
*/
type PeerIdentifier interface {
	Identifier() string
}

/*
Peer is a level on top of PeerIdentifier.  It should be created by a PeerAgent so we
can maintain multiple references to the same downstream peer (e.g. hostport).  This is
useful for load balancing requests to downstream services.
*/
type Peer interface {
	PeerIdentifier

	GetStatus() PeerStatus // Get the status of the Peer
	GetAgent() PeerAgent   // Get the PeerAgent which owns this Peer
	Pending() int          // The number of pending requests on this peer
	IncPending()           // Increment the number of pending requests
	DecPending()           // Decrement the number of pending requests
}

/*
SubscribablePeer (Name Pending) is the "Full" interface for a Peer, it should only be used by the PeerAgent.
It adds functions for adding/removing PeerSubscriber references.  If a Peer has no subscriptions/references
then it should be safe to delete.
*/
type SubscribablePeer interface {
	Peer

	OnRetain(PeerSubscriber) error  // Tell Peer it is being tracked by PeerSubscriber
	OnRelease(PeerSubscriber) error // Tell Peer it is no longer being tracked by PeerSubscriber
	References() int                // The number of PeerSubscribers referencing this peer
}

/*
PeerSubscriber is an interface for listening to changes on a Peer Over time.
*/
type PeerSubscriber interface {
	NotifyAvailable(Peer) error   // The Peer Notifies the PeerSubscriber that it can accept requests
	NotifyConnecting(Peer) error  // The Peer Notifies the PeerSubscriber that it is setting up connections
	NotifyUnavailable(Peer) error // The Peer Notifies the PeerSubscriber that it is ineligible for requests
	NotifyPending(Peer)           // The Peer Notifies the PeerSubscriber when its pending request count changes (maybe to 0).
}
