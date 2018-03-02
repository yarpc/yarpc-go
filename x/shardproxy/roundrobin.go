package shardproxy

import (
	"container/ring"
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

// newPeerRing creates a new peerRing with an initial capacity
func newPeerRing() *peerRing {
	return &peerRing{}
}

type node struct {
	peer *peerThunk
	node *ring.Ring
}

// peerRing provides a safe way to interact (Add/Remove/Get) with a potentially
// changing list of peer objects
// peerRing is NOT Thread-safe, make sure to only call peerRing functions with a lock
type peerRing struct {
	nextNode *ring.Ring
}

// Add a peer.StatusPeer to the end of the peerRing, if the ring is empty it
// initializes the nextNode marker
func (pr *peerRing) Add(p *peerThunk) *node {
	sub := &node{peer: p}
	newNode := ring.New(1)
	newNode.Value = sub
	sub.node = newNode

	if pr.nextNode == nil {
		// Empty ring, add the first node
		pr.nextNode = newNode
	} else {
		// Push the node to the ring
		pushBeforeNode(pr.nextNode, newNode)
	}
	return sub
}

// Remove the peer from the ring. Use the subscriber to address the node of the
// ring directly.
func (pr *peerRing) Remove(n *node) {
	node := n.node
	if isLastRingNode(node) {
		pr.nextNode = nil
	} else {
		if pr.isNextNode(node) {
			pr.nextNode = pr.nextNode.Next()
		}
		popNodeFromRing(node)
	}
}

func (pr *peerRing) isNextNode(node *ring.Ring) bool {
	return pr.nextNode == node
}

// Choose returns the next peer in the ring, or nil if there is no peer in the ring
// after it has the next peer, it increments the nextPeer marker in the ring
func (pr *peerRing) Choose(_ context.Context, _ *transport.Request) peer.StatusPeer {
	if pr.nextNode == nil {
		return nil
	}

	p := getPeerForRingNode(pr.nextNode)
	pr.nextNode = pr.nextNode.Next()

	return p
}

func getPeerForRingNode(rNode *ring.Ring) peer.StatusPeer {
	return rNode.Value.(*node).peer
}

func isLastRingNode(rNode *ring.Ring) bool {
	return rNode.Next() == rNode
}

func popNodeFromRing(rNode *ring.Ring) {
	rNode.Prev().Unlink(1)
}

func pushBeforeNode(curNode, newNode *ring.Ring) {
	curNode.Prev().Link(newNode)
}
