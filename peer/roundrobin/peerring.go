// Copyright (c) 2019 Uber Technologies, Inc.
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

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/peerlist/v2"
	"go.uber.org/yarpc/yarpcerrors"
)

var errUnavailable = yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "no peer available in round-robin peer list")

// newPeerRing creates a new peerRing with an initial capacity
func newPeerRing() *peerRing {
	return &peerRing{}
}

type subscriber struct {
	peer peer.StatusPeer
	node *ring.Ring
}

func (s *subscriber) NotifyStatusChanged(pid peer.Identifier) {
}

// peerRing provides a safe way to interact (Add/Remove/Get) with a potentially
// changing list of peer objects
// peerRing is NOT Thread-safe, make sure to only call peerRing functions with a lock
type peerRing struct {
	nextNode *ring.Ring
}

var _ peerlist.Implementation = (*peerRing)(nil)

// Add a peer.StatusPeer to the end of the peerRing, if the ring is empty it
// initializes the nextNode marker
func (pr *peerRing) Add(p peer.StatusPeer, _ peer.Identifier) peer.Subscriber {
	sub := &subscriber{peer: p}
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
func (pr *peerRing) Remove(p peer.StatusPeer, _ peer.Identifier, s peer.Subscriber) {
	sub, ok := s.(*subscriber)
	if !ok {
		// Don't panic.
		return
	}

	node := sub.node
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
	return rNode.Value.(*subscriber).peer
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

func (pr *peerRing) Start() error {
	return nil
}

func (pr *peerRing) Stop() error {
	return nil
}

func (pr *peerRing) IsRunning() bool {
	return true
}
