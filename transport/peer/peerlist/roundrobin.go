package peerlist

import (
	"context"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"github.com/uber-go/atomic"
)

// NewRoundRobin creates a new round robin PeerList using
func NewRoundRobin(peerIDs []transport.PeerIdentifier, agent transport.Agent) (*RoundRobin, error) {
	rr := &RoundRobin{
		pr:             newPeerRing(len(peerIDs)),
		agent:          agent,
		peerAddedEvent: make(chan struct{}, 1),
	}

	err := rr.addMulti(peerIDs)
	return rr, err
}

// RoundRobin is a PeerList which rotates which peers are to be selected in a circle
type RoundRobin struct {
	lock sync.Mutex

	pr *peerRing

	peerAddedEvent chan struct{}

	agent   transport.Agent
	started atomic.Bool
}

func (pl *RoundRobin) addMulti(peerIDs []transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	for _, peerID := range peerIDs {
		p, err := pl.agent.RetainPeer(peerID, pl)
		if err != nil {
			return err
		}

		// TODO add event/log when duplicates are inserted
		pl.pr.Add(p)
	}
	return nil
}

func (pl *RoundRobin) notifyPeerAddedEvent() {
	select {
	case pl.peerAddedEvent <- struct{}{}:
	default:
	}
}

func (pl *RoundRobin) waitForPeerAddedEventOrTimeout(ctx context.Context) error {
	select {
	case <-pl.peerAddedEvent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pl *RoundRobin) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrPeerListAlreadyStarted("RoundRobinList")
	}
	return nil
}

// Stop notifies the RoundRobin that requests will stop coming
func (pl *RoundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("RoundRobinList")
	}
	pl.lock.Lock()
	defer pl.lock.Unlock()
	return pl.clearPeers()
}

func (pl *RoundRobin) clearPeers() error {
	peers := pl.pr.popAll()
	for _, p := range peers {
		err := pl.agent.ReleasePeer(p, pl)
		if err != nil {
			return err
		}
	}
	return nil
}

// ChoosePeer selects the next available peer in the round robin
func (pl *RoundRobin) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("RoundRobinList")
	}
	pl.lock.Lock()
	defer pl.lock.Unlock()

	nextPeer := pl.pr.next()
	if nextPeer == nil {
		return nil, errors.ErrNoPeerToSelect("RoundRobinList")
	}
	return nextPeer, nil
}

func (pl *RoundRobin) Add(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	p, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.pr.Add(p)
}

func (pl *RoundRobin) Remove(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	err := pl.pr.removePeer(pid)
	if err != nil {
		// The peer has already been removed
		return err
	}

	return pl.agent.ReleasePeer(pid, pl)
}

// NotifyStatusChanged when the peer's status changes
func (pl *RoundRobin) NotifyStatusChanged(transport.Peer) {}
