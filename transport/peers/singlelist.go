package peers

import (
	"context"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

type singlePeerList struct {
	peerID  transport.PeerIdentifier
	peer    transport.Peer
	agent   transport.PeerAgent
	started atomic.Bool
}

// NewSinglePeerList creates a static PeerList with a single Peer
func NewSinglePeerList(pi transport.PeerIdentifier, agent transport.PeerAgent) transport.PeerList {
	return &singlePeerList{
		peerID: pi,
		agent:  agent,
	}
}

func (pl *singlePeerList) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrOutboundAlreadyStarted("SinglePeerList")
	}
	peer, err := pl.agent.RetainPeer(pl.peerID, pl)
	if err != nil {
		pl.started.Swap(false)
		return err
	}
	pl.peer = peer
	return nil
}

func (pl *singlePeerList) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrOutboundNotStarted("SinglePeerList")
	}
	err := pl.agent.ReleasePeer(pl.peerID, pl)
	if err != nil {
		return err
	}

	pl.peer = nil
	return nil
}

func (pl *singlePeerList) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrOutboundNotStarted("peerlist was not started")
	}
	return pl.peer, nil
}

// NotifyPending when the number of Pending requests changes
func (pl *singlePeerList) NotifyStatusChanged(transport.Peer) {}
