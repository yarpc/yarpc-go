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

package transport

//go:generate mockgen -destination=transporttest/peer.go -package=transporttest go.uber.org/yarpc/transport PeerIdentifier,Peer

// PeerConnectionStatus maintains information about the Peer's connection state
type PeerConnectionStatus int

const (
	// PeerUnavailable indicates the Peer is unavailable for requests
	PeerUnavailable PeerConnectionStatus = iota

	// PeerConnecting indicates the Peer is in the process of connecting
	PeerConnecting

	// PeerAvailable indicates the Peer is available for requests
	PeerAvailable
)

// PeerStatus holds all the information about a peer's state that would be useful to PeerSubscribers
type PeerStatus struct {
	// Current number of pending requests on this peer
	PendingRequestCount int

	// Current status of the Peer's connection
	ConnectionStatus PeerConnectionStatus
}

// PeerIdentifier is able to uniquely identify a peer (e.g. hostport)
type PeerIdentifier interface {
	Identifier() string
}

// Peer is a level on top of PeerIdentifier.  It should be created by a PeerAgent so we
// can maintain multiple references to the same downstream peer (e.g. hostport).  This is
// useful for load balancing requests to downstream services.
type Peer interface {
	PeerIdentifier

	// Get the status of the Peer
	Status() PeerStatus

	// Tell the peer that a request is starting/ending
	// The callsite should look like:
	//   done := peer.StartRequest()
	//   defer done()
	//   // Do request
	StartRequest() (finish func())
}
