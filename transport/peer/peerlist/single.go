package peerlist

import (
	"context"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
)

type single struct {
	lock sync.RWMutex

	initialPeerID transport.PeerIdentifier
	peer          transport.Peer
	agent         transport.Agent
	started       bool
}

// NewSingle creates a static PeerList with a single Peer
func NewSingle(pid transport.PeerIdentifier, agent transport.Agent) transport.PeerList {
	return &single{
		initialPeerID: pid,
		agent:         agent,
		started:       false,
	}
}

func (pl *single) Start() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()
	if pl.started {
		return errors.ErrPeerListAlreadyStarted("single")
	}
	pl.started = true

	peer, err := pl.agent.RetainPeer(pl.initialPeerID, pl)
	if err != nil {
		pl.started = false
		return err
	}
	pl.peer = peer
	return nil
}

func (pl *single) Stop() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if !pl.started {
		return errors.ErrPeerListNotStarted("single")
	}
	pl.started = false

	err := pl.agent.ReleasePeer(pl.initialPeerID, pl)
	if err != nil {
		return err
	}

	pl.peer = nil
	return nil
}

func (pl *single) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	if !pl.started {
		return nil, errors.ErrPeerListNotStarted("single")
	}
	return pl.peer, nil
}

// NotifyStatusChanged when the Peer status changes
func (pl *single) NotifyStatusChanged(transport.Peer) {}
