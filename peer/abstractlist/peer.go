// Copyright (c) 2020 Uber Technologies, Inc.
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

package abstractlist

import (
	"go.uber.org/yarpc/api/peer"
)

var _ peer.Peer = (*peerFacade)(nil)

// peerFacade captures a peer and its corresponding Subscriber, and serves as a
// Peer by proxy.
// This allows a transport to send connection status changes into a peer list
// without the list having to look them up by their identifier every time.
// It also allows the list to retain a single onFinish closure for the lifetime
// of the peer, always returning the same func for the peer whenever it is
// chosen.
type peerFacade struct {
	list       *List
	id         peer.Identifier
	peer       peer.Peer
	status     peer.Status
	subscriber Subscriber
	onFinish   func(error)
}

// StartRequest is vestigial.
//
// The peer.Peer interface requires StartRequest because transports used to
// track the pending request count.
// This responsibility is now handled by the abstract peer list itself.
func (pf *peerFacade) StartRequest() {}

// EndRequest is vestigial.
//
// The peer.Peer interface requires EndRequest because transports used to
// track the pending request count.
// This responsibility is now handled by the abstract peer list itself.
func (pf *peerFacade) EndRequest() {}

// NotifyStatusChanged receives status notifications and adjusts the peer list
// accodingly.
//
// Peers that become unavailable are removed from the implementation of the
// data structure that tracks candidates for selection.
// Peers that become available are restored to that collection.
func (pf *peerFacade) NotifyStatusChanged(pid peer.Identifier) {
	pf.list.lockAndNotifyStatusChanged(pf)
}

func (pf *peerFacade) Identifier() string {
	// pf.peer is lock safe only because we never write it, not even to nil it
	// out, even after releasing the peer.
	return pf.peer.Identifier()
}

func (pf *peerFacade) Status() peer.Status {
	return pf.list.status(pf)
}
