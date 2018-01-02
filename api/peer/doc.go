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

// Package peer contains interfaces pertaining to peers, peer lists, peer list
// updaters, and generally how to choose a peer for an outbound request.
//
// The `go.uber.org/yarpc/peer` package tree provides the corresponding
// implementations.
//
// Outbounds for some transports support selecting a different peer from a peer
// list for each individual request, for example load balancers and for pinning
// requests to a consistent peer.
// A peer instance models the host and port of a remote listening socket for a
// particular transport protocol, like HTTP and TChannel.
// For example, YARPC might have a TChannel peer instance to track connections
// to 127.0.0.1:4040.
//
// Some transports, like HTTP and TChannel support peer selection on their
// outbounds.  A `peer.Chooser` allows an outbound to obtain a peer for a given
// request and also manages the lifecycle of all components it uses.
// A peer chooser is typically also a `peer.List`, thus a `peer.ChooserList`.
//
// Peer list updaters send `Update` message to a `peer.List` to add and remove
// peers.
// Peer list updaters have no specific interface, but must in practice
// implement `transport.Lifecycle` to bookend when they start and stop sending
// updates.
// A `peer.Binder` is a function that binds a `peer.List` to a peer list
// updater and returns the `transport.Lifecycle` of the peer list updater.
//
// Not all `transport.Transport` instances support peer lists.
// To support peer selection, they must also implement `peer.Transport`, which
// has the `RetainPeer` and `ReleasePeer` methods, so a peer list can
// hold strong references to peers maintained by a transport, and receive
// notifications when peers start connecting, become available, become
// unavailable, and when their pending request count changes.
//
// A peer list must provide a `peer.Subscriber` for each peer it retains or
// releases.
// The subscriber may be the peer list itself, though most peer lists contain
// an internal representation of each of their retained peers for book-keeping
// and sorting which benefits from receiving notifications for state changes
// directly.
package peer
