package transport

import "context"

//go:generate mockgen -destination=transporttest/peerlist.go -package=transporttest go.uber.org/yarpc/transport PeerList

// PeerList is a collection of Peers.  Outbounds request peers from the ChoosePeer to determine where to send requests
type PeerList interface {
	PeerSubscriber

	Start() error // Notify the PeerList that it will start receiving requests
	Stop() error  // Notify the PeerList that it will stop receiving requests

	ChoosePeer(context.Context, *Request) (Peer, error) // Choose a Peer for the next call
}
