// Copyright (c) 2018 Uber Technologies, Inc.
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

package peer

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Chooser is a collection of Peers. Outbounds request peers from the
// peer.Chooser to determine where to send requests.
// The chooser is responsible for managing the lifecycle of any retained peers.
type Chooser interface {
	transport.Lifecycle

	// Choose a Peer for the next call, block until a peer is available (or timeout)
	Choose(context.Context, *transport.Request) (peer Peer, onFinish func(error), err error)
}

// List listens to adds and removes of Peers from a peer list updater.
// A Chooser will implement the List interface in order to receive
// updates to the list of Peers it is keeping track of.
type List interface {
	// Update performs the additions and removals to the Peer List
	Update(updates ListUpdates) error
}

// ListUpdates specifies the updates to be made to a List
type ListUpdates struct {
	// Additions are the identifiers that should be added to the list
	Additions []Identifier

	// Removals are the identifiers that should be removed to the list
	Removals []Identifier
}

// ChooserList is both a Chooser and a List, useful for expressing both
// capabilities of a single instance.
type ChooserList interface {
	Chooser
	List
}

// ListImplementation is a collection of available peers, with its own
// subscribers for peer status change notifications.
// The available peer list encapsulates the logic for selecting from among
// available peers, whereas a ChooserList is responsible for retaining,
// releasing, and monitoring peer availability.
// Use "go.uber.org/yarpc/peer/peerlist".List in conjunction with a
// ListImplementation to produce a "go.uber.org/yarpc/api/peer".List.
//
// peerlist.List and ListImplementation compose well with sharding schemes the
// degenerate to returning the only available peer.
//
// Deprecated in favor of "go.uber.org/yarpc/peer/peerlist/v2".Implementation.
type ListImplementation interface {
	transport.Lifecycle

	Add(StatusPeer) Subscriber
	Remove(StatusPeer, Subscriber)
	// Choose must return an available peer under a list read lock, so must
	// not block.
	Choose(context.Context, *transport.Request) StatusPeer
}

// Binder is a callback for peer.Bind that accepts a peer list and binds it to
// a peer list updater for the duration of the returned peer list updater.
// The peer list updater must implement the lifecycle interface, and start and
// stop updates over that lifecycle.
// The binder must not block on updating the list, because update may block
// until the peer list has started.
// The binder must return a peer list updater that will begin updating when it
// starts, and stop updating when it stops.
type Binder func(List) transport.Lifecycle
