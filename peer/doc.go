// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package peer contains components for managing peers.
//
// The `go.uber.org/yarpc/api/peer` package provides the corresponding
// interfaces that these components implement.
//
// The HTTP and TChannel `NewOutbound` methods accept a `peer.Chooser`.
// To bind an outbound to a single peer, pass a `peer.NewSingle(id)`.
//
// To bind an outbound to multiple peers, you will need a peer list, like
// `roundrobin` (to pick the least recently chosen peer) or `peerheap` (to
// choose the peer with the fewest pending requests).
// Assuming you choose `peerheap`, to bind multiple peers you would pass
// `peer.Bind(peerheap.New(transport), peer.BindPeers(ids))`.
//
// Each transport can define its own domain of peer identifiers, but most will
// use a host:port address for the remote listening socket.
// The `"go.uber.org/yarpc/peer/hostport"` package implements the common
// `hostport.PeerIdentifier` and `hostport.Peer`.
package peer
