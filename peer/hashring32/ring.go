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

package hashring32

import (
	"strconv"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/abstractlist"
	"go.uber.org/yarpc/peer/hashring32/internal/farmhashring"
	"go.uber.org/yarpc/peer/hashring32/internal/hashring32"
	"go.uber.org/zap"
)

// NewImplementation creates a new hashring32 abstractlist.Implementation.
//
// Use this constructor instead of NewList, when wanting to do custom peer
// connection management.
func NewImplementation(opts ...Option) abstractlist.Implementation {
	options := options{
		logger: zap.NewNop(),
	}
	for _, o := range opts {
		o.apply(&options)
	}

	return newPeerRing(
		farmhashring.Fingerprint32,
		options.offsetHeader,
		options.peerOverrideHeader,
		options.logger,
		options.peerRingOptions...,
	)
}

// newPeerRing creates a new peerRing with an initial capacity
func newPeerRing(hashFunc hashring32.HashFunc32, offsetHeader, peerOverrideHeader string, logger *zap.Logger, option ...hashring32.Option) *peerRing {
	return &peerRing{
		ring:               hashring32.New(hashFunc, option...),
		subscribers:        make(map[string]*subscriber),
		offsetHeader:       offsetHeader,
		peerOverrideHeader: peerOverrideHeader,
		logger:             logger,
	}
}

type subscriber struct {
	peer peer.StatusPeer
}

func (s *subscriber) UpdatePendingRequestCount(int) {}

// peerRing provides a safe way to interact (Add/Remove/Get) with a potentially
// changing list of peer objects
// peerRing is NOT Thread-safe, make sure to only call peerRing functions with a lock
type peerRing struct {
	ring               *hashring32.Hashring32
	subscribers        map[string]*subscriber
	offsetHeader       string
	peerOverrideHeader string
	logger             *zap.Logger
}

var _ abstractlist.Implementation = (*peerRing)(nil)

// shardIdentifier is the interface for an identifier that have a shard property
type shardIdentifier interface {
	Identifier() string
	Shard() string
}

// Add a string to the end of the peerRing, if the ring is empty
// it initializes the ring marker
func (pr *peerRing) Add(p peer.StatusPeer, pid peer.Identifier) abstractlist.Subscriber {
	sub := &subscriber{peer: p}
	shardID := getShardID(pid)
	pr.ring.Add(shardID)
	pr.subscribers[shardID] = sub

	return sub
}

// Remove the peer from the ring. Use the subscriber to address the node of the
// ring directly.
func (pr *peerRing) Remove(p peer.StatusPeer, pid peer.Identifier, s abstractlist.Subscriber) {
	sub, ok := s.(*subscriber)
	if !ok {
		// Don't panic.
		return
	}
	// Peerlist's responsibility to make sure this never happens.
	if sub.peer.Identifier() != p.Identifier() {
		return
	}

	shardID := getShardID(pid)
	pr.ring.Add(shardID)
	pr.subscribers[shardID] = sub

	// validate that given peer is already in the subscriber
	pr.ring.Remove(shardID)
	// Peerlist's responsibility to make sure this is thread-safe.
	delete(pr.subscribers, shardID)
}

func (pr *peerRing) getPeerOverride(req *transport.Request) peer.StatusPeer {
	dest, ok := req.Headers.Get(pr.peerOverrideHeader)
	if !ok {
		return nil
	}
	sub, ok := pr.subscribers[dest]
	if !ok {
		return nil
	}
	return sub.peer
}

// Choose returns the assigned peer in the ring
func (pr *peerRing) Choose(req *transport.Request) peer.StatusPeer {
	// Client may want this request to go to a specific destination.
	overridePeer := pr.getPeerOverride(req)
	if overridePeer != nil {
		return overridePeer
	}

	// Hashring choose can return an error eg if there are no peer available.
	var n int
	var err error
	key, ok := req.Headers.Get(pr.offsetHeader)
	if ok {
		n, err = strconv.Atoi(key)
		if err != nil {
			pr.logger.Error("yarpc/hashring32: offset header is not a valid integer", zap.String("offsetHeader", key), zap.Error(err))
		}
	}

	ids, err := pr.ring.Choose(hashring32.Shard{
		Key: req.ShardKey,
		N:   n,
	})
	if err != nil {
		return nil
	}
	if len(ids) <= n {
		return nil
	}
	sub, ok := pr.subscribers[ids[n]]
	if !ok {
		return nil
	}

	return sub.peer
}

// getShardID returns the shardID from a StatusPeer.
func getShardID(p peer.Identifier) string {
	sp, ok := p.(shardIdentifier)
	var id string
	if !ok {
		id = p.Identifier()
	} else {
		id = sp.Shard()
	}
	return id
}
