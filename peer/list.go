// Copyright (c) 2016 Uber Technologies, Inc.
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

	"go.uber.org/yarpc/transport"
)

//go:generate mockgen -destination=peertest/list.go -package=peertest go.uber.org/yarpc/peer Chooser,ChangeListener

// Chooser is a collection of Peers.  Outbounds request peers from the peer.Chooser to determine where to send requests
type Chooser interface {
	// Notify the PeerList that it will start receiving requests
	Start() error

	// Notify the PeerList that it will stop receiving requests
	Stop() error

	// Choose a Peer for the next call, block until a peer is available (or timeout)
	ChoosePeer(context.Context, *transport.Request) (Peer, error)
}

// ChangeListener listens to adds and removes of Peers from a PeerProvider
// A List will implement the PeerChangeListener interface in order to receive
// updates to the list of Peers it is keeping track of
type ChangeListener interface {
	// Add a peer to the List (Called directly from a PeerProvider)
	Add(Identifier) error

	// Remove a peer from the List (Called directly from a PeerProvider)
	Remove(Identifier) error
}
