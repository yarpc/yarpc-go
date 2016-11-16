package peerlist

import (
	"container/ring"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
)

// NewPeerRing creates a new PeerRing with initial size of "length"
func NewPeerRing(length int) *PeerRing {
	return &PeerRing{
		peerToNode: make(map[string]*peerRingNode, length),
	}
}

// PeerRing provides a safe way to interact (Add/Remove/Get) with a potentially
// changing list of peer objects
type PeerRing struct {
	lock sync.Mutex

	peerToNode map[string]*peerRingNode
	nextNode   *peerRingNode
}

// Add a transport.Peer to the end of the PeerRing, if the ring is empty
// it initializes the nextNode marker
func (pr *PeerRing) Add(peer transport.Peer) error {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	if _, ok := pr.peerToNode[peer.Identifier()]; ok {
		// Peer Already in ring, ignore the add
		return errors.ErrPeerAlreadyInList{
			Peer: peer,
		}
	}

	newNode := newPeerRingNode(peer)
	pr.peerToNode[peer.Identifier()] = newNode

	if pr.nextNode == nil {
		// Empty ring, add the first node
		pr.nextNode = newNode
	} else {
		// Push the node to the ring
		pr.nextNode.push(newNode)
	}
	return nil
}

// Remove a peer PeerIdentifier from the PeerRing, if the PeerID is not
// in the ring return an error
func (pr *PeerRing) Remove(pid transport.PeerIdentifier) error {
	pr.lock.Lock()
	defer pr.lock.Unlock()

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

// RemoveAll pops all the peers from the ring and returns them in a list
func (pr *PeerRing) RemoveAll() []transport.Peer {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	peers := make([]transport.Peer, 0, len(pr.peerToNode))
	for _, node := range pr.peerToNode {
		peers = append(peers, pr.popNode(node))
	}
	return peers
}

func (pr *PeerRing) popNode(node *peerRingNode) transport.Peer {
	p := node.getPeer()

	if node.isLastNode() {
		pr.nextNode = nil
	} else {
		if pr.isNextNode(node) {
			pr.nextNode = pr.nextNode.nextRingNode()
		}
		node.pop()
	}

	// Remove the node from our node map
	delete(pr.peerToNode, p.Identifier())

	return p
}

func (pr *PeerRing) isNextNode(node *peerRingNode) bool {
	return node.equals(pr.nextNode)
}

// Next returns the next peer in the ring, or nil if there is no peer in the ring
// after it has the next peer, it increments the nextPeer marker in the ring
func (pr *PeerRing) Next() transport.Peer {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	if pr.nextNode == nil {
		return nil
	}

	p := pr.nextNode.getPeer()

	pr.nextNode = pr.nextNode.nextRingNode()

	return p
}

func newPeerRingNode(peer transport.Peer) *peerRingNode {
	newNode := &peerRingNode{
		Ring: ring.New(1),
	}
	newNode.Value = peer
	return newNode
}

type peerRingNode struct {
	*ring.Ring
}

func (prn *peerRingNode) getPeer() transport.Peer {
	return prn.Value.(transport.Peer)
}

func (prn *peerRingNode) isLastNode() bool {
	return prn.Ring.Next() == prn.Ring
}

func (prn *peerRingNode) equals(compPR *peerRingNode) bool {
	return prn.Ring == compPR.Ring
}

func (prn *peerRingNode) pop() {
	prn.Prev().Unlink(1)
}

func (prn *peerRingNode) push(newPR *peerRingNode) {
	prn.Prev().Link(newPR.Ring)
}

func (prn *peerRingNode) nextRingNode() *peerRingNode {
	return &peerRingNode{
		Ring: prn.Ring.Next(),
	}
}
