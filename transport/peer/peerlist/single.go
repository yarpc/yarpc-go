package peerlist

import (
	"context"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

type single struct {
	peerID  transport.PeerIdentifier
	peer    transport.Peer
	agent   transport.PeerAgent
	started atomic.Bool
}

// NewSingle creates a static PeerList with a single Peer
func NewSingle(pi transport.PeerIdentifier, agent transport.PeerAgent) transport.PeerList {
	return &single{
		peerID: pi,
		agent:  agent,
	}
}

func (pl *single) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrPeerListAlreadyStarted("single")
	}
	peer, err := pl.agent.RetainPeer(pl.peerID, pl)
	if err != nil {
		pl.started.Swap(false)
		return err
	}
	pl.peer = peer
	return nil
}

func (pl *single) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("single")
	}
	err := pl.agent.ReleasePeer(pl.peerID, pl)
	if err != nil {
		return err
	}

	pl.peer = nil
	return nil
}

func (pl *single) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("single")
	}
	return pl.peer, nil
}

// NotifyPending when the number of Pending requests changes
func (pl *single) NotifyStatusChanged(transport.Peer) {}
