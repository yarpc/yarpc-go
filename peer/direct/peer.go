package direct

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
)

func newPeerIdentifier(identifier string) peer.Identifier {
	return hostport.Identify(identifier)
}

type peerSubscriber struct{}

func newPeerSubscriber() peer.Subscriber {
	return &peerSubscriber{}
}

func (d *peerSubscriber) NotifyStatusChanged(_ peer.Identifier) {}
