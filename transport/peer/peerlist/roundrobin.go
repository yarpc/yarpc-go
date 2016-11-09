package peerlist

import (
	"container/ring"
	"context"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"github.com/uber-go/atomic"
)

type peerRing struct {
	*ring.Ring
}

func newPeerRing(peer transport.Peer) *peerRing {
	newNode := &peerRing{
		Ring: ring.New(1),
	}
	newNode.Value = peer
	return newNode
}

func (pr *peerRing) getPeer() transport.Peer {
	return pr.Value.(transport.Peer)
}

func (pr *peerRing) isLastPeer() bool {
	return pr.Ring.Next() == pr.Ring
}

func (pr *peerRing) pop() {
	pr.Prev().Unlink(1)
}

func (pr *peerRing) push(newPR *peerRing) {
	pr.Prev().Link(newPR.Ring)
}

func (pr *peerRing) nextPeer() *peerRing {
	return &peerRing{
		Ring: pr.Ring.Next(),
	}
}

// NewRoundRobin creates a new round robin PeerList using
func NewRoundRobin(peerIDs []transport.PeerIdentifier, agent transport.Agent) (*RoundRobin, error) {
	rr := &RoundRobin{
		peerToNode:     make(map[string]*peerRing, len(peerIDs)),
		agent:          agent,
		peerAddedEvent: make(chan struct{}, 1),
	}

	err := rr.addMulti(peerIDs)
	return rr, err
}

// RoundRobin is a PeerList which rotates which peers are to be selected in a circle
type RoundRobin struct {
	lock sync.Mutex

	peerToNode map[string]*peerRing
	nextNode   *peerRing

	peerAddedEvent chan struct{}

	agent   transport.Agent
	started atomic.Bool
}

func (pl *RoundRobin) addMulti(peerIDs []transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	for _, peerID := range peerIDs {
		peer, err := pl.agent.RetainPeer(peerID, pl)
		if err != nil {
			return err
		}

		// TODO add event/log when duplicates are inserted
		pl.addToRing(peer)
	}
	return nil
}

func (pl *RoundRobin) addToRing(peer transport.Peer) error {
	_, ok := pl.peerToNode[peer.Identifier()]
	if ok {
		// Peer Already exists, ignore the add
		return errors.ErrPeerAlreadyInList{
			Peer:     peer,
			PeerList: pl,
		}
	}

	defer pl.notifyPeerAddedEvent()

	newNode := newPeerRing(peer)

	pl.peerToNode[peer.Identifier()] = newNode

	next := pl.nextNode
	if next == nil {

		// Empty List, add the first node
		pl.nextNode = newNode

		return nil
	}

	next.push(newNode)

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

func (pl *RoundRobin) removePeer(pid transport.PeerIdentifier) error {
	node, ok := pl.peerToNode[pid.Identifier()]
	if !ok {
		// Peer doesn't exist in the list
		return errors.ErrPeerNotInList{
			PeerIdentifier: pid,
			PeerList:       pl,
		}
	}

	if node.isLastPeer() {
		// This is the last node, set the nextNode to nil
		pl.nextNode = nil
	} else {
		// Unlink one node after the "Prev" node (i.e. the current node)
		node.pop()
	}

	// Remove the node from our node map
	delete(pl.peerToNode, pid.Identifier())

	return nil
}

func (pl *RoundRobin) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrPeerListAlreadyStarted("RoundRobinList")
	}
	return nil
}

func (pl *RoundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("RoundRobinList")
	}
	return pl.clearPeers()
}

func (pl *RoundRobin) clearPeers() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	for _, node := range pl.peerToNode {
		peer := node.getPeer()

		err := pl.agent.ReleasePeer(peer, pl)
		if err != nil {
			return err
		}

		err = pl.removePeer(peer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pl *RoundRobin) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("RoundRobinList")
	}
	return pl.nextPeer()
}

func (pl *RoundRobin) nextPeer() (transport.Peer, error) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if pl.nextNode == nil {
		return nil, errors.ErrNoPeerToSelect("RoundRobinList")
	}

	peer := pl.nextNode.getPeer()

	pl.nextNode = pl.nextNode.nextPeer()

	return peer, nil
}

func (pl *RoundRobin) Add(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	peer, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.addToRing(peer)
}

func (pl *RoundRobin) Remove(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	err := pl.agent.ReleasePeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.removePeer(pid)
}

// NotifyStatusChanged when the number of Pending requests changes
func (pl *RoundRobin) NotifyStatusChanged(transport.Peer) {}
