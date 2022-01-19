// Copyright (c) 2022 Uber Technologies, Inc.
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

// Subscriber listens to changes of a Peer over time.
type Subscriber interface {
	// The Peer Notifies the Subscriber when its status changes (e.g.
	// connections status, pending requests)
	//
	// NotifyStatusChanged may call methods of the corresponding Peer,
	// particularly Status.
	// So, subscriber methods must not be called while holding a lock on the
	// Transport.
	NotifyStatusChanged(Identifier)
}

// Transport manages Peers across different Subscribers.
//
// A Subscriber will request a Peer for a specific PeerIdentifier and the
// Transport has the ability to create a new Peer or return an existing one.
//
// RetainPeer and ReleasePeer must not call or synchronize with goroutines that
// call methods of the Subscriber.
type Transport interface {
	// Get or create a Peer for the Subscriber
	RetainPeer(Identifier, Subscriber) (Peer, error)

	// Unallocate a peer from the Subscriber
	ReleasePeer(Identifier, Subscriber) error
}
