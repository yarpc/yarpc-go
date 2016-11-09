package peerlist

import (
	"container/ring"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
)

func newPeerRing(length int) *peerRing {
	return &peerRing{
		peerToNode: make(map[string]*peerRingNode, length),
	}
}

type peerRing struct {
	peerToNode map[string]*peerRingNode
	nextNode   *peerRingNode
}

func (pr *peerRing) Add(peer transport.Peer) error {
	if _, ok := pr.peerToNode[peer.Identifier()]; ok {
		// Peer Already exists, ignore the add
		return errors.ErrPeerAlreadyInList{
			Peer: peer,
		}
	}

	newNode := newPeerRingNode(peer)

	pr.peerToNode[peer.Identifier()] = newNode

	if pr.nextNode == nil {
		// Empty List, add the first node
		pr.nextNode = newNode

		return nil
	}

	pr.nextNode.push(newNode)

	return nil
}

func (pr *peerRing) removePeer(pid transport.PeerIdentifier) error {
	node, ok := pr.peerToNode[pid.Identifier()]
	if !ok {
		// Peer doesn't exist in the list
		return errors.ErrPeerNotInList{
			PeerIdentifier: pid,
		}
	}

	pr.popNode(node)

	return nil
}

func (pr *peerRing) popAll() []transport.Peer {
	peers := make([]transport.Peer, 0, len(pr.peerToNode))
	for _, node := range pr.peerToNode {
		peers = append(peers, pr.popNode(node))
	}
	return peers
}

func (pr *peerRing) popNode(node *peerRingNode) transport.Peer {
	p := node.getPeer()

	if node.isLastPeer() {
		// This is the last node, set the nextNode to nil
		pr.nextNode = nil
	} else {
		if node.equals(pr.nextNode) {
			pr.nextNode = pr.nextNode.nextPeer()
		}
		// Unlink one node after the "Prev" node (i.e. the current node)
		node.pop()
	}

	// Remove the node from our node map
	delete(pr.peerToNode, p.Identifier())

	return p
}

func (pr *peerRing) next() transport.Peer {
	if pr.nextNode == nil {
		return nil
	}

	p := pr.nextNode.getPeer()

	pr.nextNode = pr.nextNode.nextPeer()

	return p
}

type peerRingNode struct {
	*ring.Ring
}

func newPeerRingNode(peer transport.Peer) *peerRingNode {
	newNode := &peerRingNode{
		Ring: ring.New(1),
	}
	newNode.Value = peer
	return newNode
}

func (pr *peerRingNode) getPeer() transport.Peer {
	return pr.Value.(transport.Peer)
}

func (pr *peerRingNode) isLastPeer() bool {
	return pr.Ring.Next() == pr.Ring
}

func (pr *peerRingNode) equals(compPR *peerRingNode) bool {
	return pr.Ring == compPR.Ring
}

func (pr *peerRingNode) pop() {
	pr.Prev().Unlink(1)
}

func (pr *peerRingNode) push(newPR *peerRingNode) {
	pr.Prev().Link(newPR.Ring)
}

func (pr *peerRingNode) nextPeer() *peerRingNode {
	return &peerRingNode{
		Ring: pr.Ring.Next(),
	}
}
