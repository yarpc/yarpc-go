// Copyright (c) 2025 Uber Technologies, Inc.
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

// ConnectionStatus maintains information about the Peer's connection state
type ConnectionStatus int

const (
	// Unavailable indicates the Peer is unavailable for requests
	Unavailable ConnectionStatus = iota

	// Connecting indicates the Peer is in the process of connecting
	Connecting

	// Available indicates the Peer is available for requests
	Available
)

// Status holds all the information about a peer's state that would be useful to Subscribers
type Status struct {
	// Current number of pending requests on this peer
	PendingRequestCount int

	// Current status of the Peer's connection
	ConnectionStatus ConnectionStatus
}

// Identifier is able to uniquely identify a peer (e.g. hostport)
type Identifier interface {
	Identifier() string
}

// StatusPeer captures a concrete peer implementation for a particular
// transport, exposing its Identifier and Status.
// StatusPeer provides observability without mutability.
type StatusPeer interface {
	Identifier

	// Get the status of the Peer
	Status() Status
}

// Peer captures a concrete peer implementation for a particular transport,
// providing both observability (Identifier and Status), along with load change
// notifications (StartRequest) (EndRequest).
// Transports reveal peers to peer lists, which in turn offer them to outbounds
// when they choose a peer.
// Having Start/End request messages allows the outbound to broadcast load
// changes to all subscribed load balancers.
// The peer should be created by a transport so we can maintain multiple
// references to the same peer (e.g., hostport).
type Peer interface {
	StatusPeer

	// Tell the peer that a request is starting
	StartRequest()

	// Tell the peer that a request has finished
	EndRequest()
}
