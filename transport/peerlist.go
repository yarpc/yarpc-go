package transport

import "context"

//go:generate mockgen -destination=transporttest/peerlist.go -package=transporttest go.uber.org/yarpc/transport PeerList

// PeerList is a collection of Peers.  Outbounds request peers from the PeerList to determine where to send requests
type PeerList interface {
	// Notify the PeerList that it will start receiving requests
	Start() error

	// Notify the PeerList that it will stop receiving requests
	Stop() error

	// Choose a Peer for the next call, block until a peer is available (or timeout)
	ChoosePeer(context.Context, *Request) (Peer, error)
}
