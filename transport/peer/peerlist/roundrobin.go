package peerlist

import (
	"context"
	"sync"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/uber-go/atomic"
)

// NewRoundRobin creates a new round robin PeerList using
func NewRoundRobin(peerIDs []transport.PeerIdentifier, agent transport.PeerAgent) transport.PeerList {
	return &roundRobin{
		initialPeerIDs: peerIDs,
		peerToNode:     make(map[string]*roundRobinNode, len(peerIDs)),
		agent:          agent,
		started:        atomic.NewBool(false),
		nextNode:       nil,
	}
}

type roundRobin struct {
	sync.Mutex

	initialPeerIDs []transport.PeerIdentifier
	peerToNode     map[string]*roundRobinNode
	agent          transport.PeerAgent
	started        *atomic.Bool
	nextNode       *roundRobinNode
}

type roundRobinNode struct {
	peer         transport.Peer
	nextNode     *roundRobinNode
	previousNode *roundRobinNode
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

func (pl *roundRobin) removePeer(peer transport.Peer) error {
	node, ok := pl.peerToNode[peer.Identifier()]
	if !ok {
		// Peer doesn't exist in the list
		return errors.ErrPeerNotInList{
			Peer:     peer,
			PeerList: pl,
		}
	}

	if node.previousNode == node.nextNode {
		// This is the last node, set the nextNode to this
		pl.nextNode = nil
		delete(pl.peerToNode, peer.Identifier())
		return nil
	}

	if pl.nextNode == node {
		pl.nextNode = node.nextNode
	}

	// Switch the previous node's 'next' and the next node's 'previous'
	node.previousNode.nextNode, node.nextNode.previousNode = node.nextNode, node.previousNode

	// Remove the node from our node map
	delete(pl.peerToNode, peer.Identifier())

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
	pl.Lock()
	defer pl.Unlock()

	for _, peerID := range pl.initialPeerIDs {
		peer, err := pl.agent.RetainPeer(peerID, pl)
		if err != nil {
			return err
		}

		// TODO add event/log when duplicates are inserted
		pl.addPeer(peer)
	}
	return nil
}

func (pl *roundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("RoundRobinList")
	}
	pl.Lock()
	defer pl.Unlock()

	for _, node := range pl.peerToNode {
		peer := node.peer

		err := pl.agent.ReleasePeer(peer, pl)
		if err != nil {
			return err
		}

		pl.removePeer(peer)
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

// NotifyPending when the number of Pending requests changes
func (pl *roundRobin) NotifyStatusChanged(transport.Peer) {}
