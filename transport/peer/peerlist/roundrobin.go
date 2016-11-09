package peerlist

import (
	"context"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"github.com/uber-go/atomic"
)

// NewRoundRobin creates a new round robin PeerList using
func NewRoundRobin(peerIDs []transport.PeerIdentifier, agent transport.PeerAgent) (transport.PeerList, error) {
	rr := &roundRobin{
		peerToNode: make(map[string]*roundRobinNode, len(peerIDs)),
		agent:      agent,
		started:    atomic.NewBool(false),
		nextNode:   nil,
	}

	err := rr.createPeers(peerIDs)
	if err != nil {
		return nil, err
	}

	return rr, nil
}

type roundRobin struct {
	sync.Mutex

	peerToNode map[string]*roundRobinNode
	agent      transport.PeerAgent
	started    *atomic.Bool
	nextNode   *roundRobinNode
}

type roundRobinNode struct {
	peer         transport.Peer
	nextNode     *roundRobinNode
	previousNode *roundRobinNode
}

func (pl *roundRobin) createPeers(peerIDs []transport.PeerIdentifier) error {
	pl.Lock()
	defer pl.Unlock()
	for _, peerID := range peerIDs {
		peer, err := pl.agent.RetainPeer(peerID, pl)
		if err != nil {
			return err
		}

		// TODO add event/log when duplicates are inserted
		pl.addPeer(peer)
	}
	return nil
}

func (pl *roundRobin) addPeer(peer transport.Peer) error {
	_, ok := pl.peerToNode[peer.Identifier()]
	if ok {
		// Peer Already exists, ignore the add
		return errors.ErrPeerAlreadyInList{
			Peer:     peer,
			PeerList: pl,
		}
	}

	next := pl.nextNode
	if next == nil {
		// Empty List, add the first node
		newNode := &roundRobinNode{
			peer: peer,
		}
		newNode.nextNode = newNode
		newNode.previousNode = newNode
		pl.peerToNode[peer.Identifier()] = newNode

		pl.nextNode = newNode
		return nil
	}

	previous := next.previousNode

	newNode := &roundRobinNode{
		peer:         peer,
		nextNode:     next,
		previousNode: previous,
	}

	previous.nextNode = newNode
	next.previousNode = newNode

	pl.peerToNode[peer.Identifier()] = newNode
	return nil
}

func (pl *roundRobin) removePeer(pid transport.PeerIdentifier) error {
	node, ok := pl.peerToNode[pid.Identifier()]
	if !ok {
		// Peer doesn't exist in the list
		return errors.ErrPeerNotInList{
			PeerIdentifier: pid,
			PeerList:       pl,
		}
	}

	if node.previousNode == node {
		// This is the last node, set the nextNode to this
		pl.nextNode = nil
		delete(pl.peerToNode, pid.Identifier())
		return nil
	}

	if pl.nextNode == node {
		pl.nextNode = node.nextNode
	}

	// Switch the previous node's 'next' and the next node's 'previous'
	node.previousNode.nextNode, node.nextNode.previousNode = node.nextNode, node.previousNode

	// Remove the node from our node map
	delete(pl.peerToNode, pid.Identifier())

	return nil
}

func (pl *roundRobin) nextPeer() (transport.Peer, error) {
	if pl.nextNode == nil {
		return nil, errors.ErrNoPeerToSelect("RoundRobinList")
	}

	peer := pl.nextNode.peer
	pl.nextNode = pl.nextNode.nextNode
	return peer, nil
}

func (pl *roundRobin) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrPeerListAlreadyStarted("RoundRobinList")
	}
	return nil
}

func (pl *roundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("RoundRobinList")
	}
	return pl.clearPeers()
}

func (pl *roundRobin) clearPeers() error {
	pl.Lock()
	defer pl.Unlock()

	for _, node := range pl.peerToNode {
		peer := node.peer

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

func (pl *roundRobin) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	pl.Lock()
	defer pl.Unlock()

	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("RoundRobinList")
	}
	return pl.nextPeer()
}

func (pl *roundRobin) Add(pid transport.PeerIdentifier) error {
	pl.Lock()
	defer pl.Unlock()

	peer, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.addPeer(peer)
}

func (pl *roundRobin) Remove(pid transport.PeerIdentifier) error {
	pl.Lock()
	defer pl.Unlock()

	err := pl.agent.ReleasePeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.removePeer(pid)
}

// NotifyStatusChanged when the peer's status changes
func (pl *roundRobin) NotifyStatusChanged(transport.Peer) {}
