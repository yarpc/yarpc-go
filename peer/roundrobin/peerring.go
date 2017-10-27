// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package roundrobin

import (
	"container/ring"
	"context"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

// newPeerRing creates a new peerRing with an initial capacity
func newPeerRing(capacity int) *peerRing {
	return &peerRing{
		peerToNode: make(map[string]*ring.Ring, capacity),
	}
}

// peerRing provides a safe way to interact (Add/Remove/Get) with a potentially
// changing list of peer objects
// peerRing is NOT Thread-safe, make sure to only call peerRing functions with a lock
type peerRing struct {
	peerToNode map[string]*ring.Ring
	nextNode   *ring.Ring
}

func (pr *peerRing) Update(updates peer.ListUpdates) error {
	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	var errs error
	for _, p := range updates.Removals {
		errs = multierr.Append(errs, pr.Remove(p))
	}
	for _, p := range updates.Additions {
		errs = multierr.Append(errs, pr.Add(p))
	}
	return errs
}

// Add a string to the end of the peerRing, if the ring is empty
// it initializes the nextNode marker
func (pr *peerRing) Add(p peer.Identifier) error {
	if _, ok := pr.peerToNode[p.Identifier()]; ok {
		// Peer Already in ring, ignore the add
		return peer.ErrPeerAddAlreadyInList(p.Identifier())
	}

	newNode := newPeerRingNode(p)
	pr.peerToNode[p.Identifier()] = newNode

	if pr.nextNode == nil {
		// Empty ring, add the first node
		pr.nextNode = newNode
	} else {
		// Push the node to the ring
		pushBeforeNode(pr.nextNode, newNode)
	}
	return nil
}

func newPeerRingNode(p peer.Identifier) *ring.Ring {
	newNode := ring.New(1)
	newNode.Value = p
	return newNode
}

// Remove a peer.Identifier from the peerRing, if the PeerID is not
// in the ring return an error
func (pr *peerRing) Remove(p peer.Identifier) error {
	node, ok := pr.peerToNode[p.Identifier()]
	if !ok {
		// Peer doesn't exist in the list
		return peer.ErrPeerRemoveNotInList(p.Identifier())
	}

	pr.popNode(node)

	return nil
}

func (pr *peerRing) popNode(node *ring.Ring) peer.Identifier {
	p := getPeerForRingNode(node)

	if isLastRingNode(node) {
		pr.nextNode = nil
	} else {
		if pr.isNextNode(node) {
			pr.nextNode = pr.nextNode.Next()
		}
		popNodeFromRing(node)
	}

	// Remove the node from our node map
	delete(pr.peerToNode, p.Identifier())

	return p
}

func (pr *peerRing) isNextNode(node *ring.Ring) bool {
	return pr.nextNode == node
}

// Choose returns the next peer in the ring, or nil if there is no peer in the ring
// after it has the next peer, it increments the nextPeer marker in the ring
func (pr *peerRing) Choose(ctx context.Context, req *transport.Request) peer.Identifier {
	if pr.nextNode == nil {
		return nil
	}

	p := getPeerForRingNode(pr.nextNode)

	pr.nextNode = pr.nextNode.Next()

	return p
}

func getPeerForRingNode(rNode *ring.Ring) peer.Identifier {
	return rNode.Value.(peer.Identifier)
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
