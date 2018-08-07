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

package randpending

import (
	"context"
	"math/rand"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	peerlist "go.uber.org/yarpc/peer/peerlist/v2"
)

type randPendingList struct {
	subscribers []*subscriber
	random      *rand.Rand
}

func newRandPendingList(cap int, source rand.Source) *randPendingList {
	return &randPendingList{
		subscribers: make([]*subscriber, 0, cap),
		random:      rand.New(source),
	}
}

var _ peerlist.Implementation = (*randPendingList)(nil)

func (r *randPendingList) Add(peer peer.StatusPeer, _ peer.Identifier) peer.Subscriber {
	index := len(r.subscribers)
	r.subscribers = append(r.subscribers, &subscriber{
		index: index,
		peer:  peer,
	})
	return r.subscribers[index]
}

func (r *randPendingList) Remove(peer peer.StatusPeer, _ peer.Identifier, ps peer.Subscriber) {
	sub, ok := ps.(*subscriber)
	if !ok || len(r.subscribers) == 0 {
		return
	}
	index := sub.index
	last := len(r.subscribers) - 1
	r.subscribers[index] = r.subscribers[last]
	r.subscribers[index].index = index
	r.subscribers = r.subscribers[0:last]
}

func (r *randPendingList) Choose(_ context.Context, _ *transport.Request) peer.StatusPeer {
	numSubs := len(r.subscribers)
	if numSubs == 0 {
		return nil
	}
	if numSubs == 1 {
		return r.subscribers[0].peer
	}
	i := r.random.Intn(numSubs)
	j := i + 1 + r.random.Intn(numSubs-1)
	if j >= numSubs {
		j -= numSubs
	}
	if r.pending(i) > r.pending(j) {
		i = j
	}
	return r.subscribers[i].peer
}

func (r *randPendingList) pending(index int) int {
	return r.subscribers[index].peer.Status().PendingRequestCount
}

func (r *randPendingList) Start() error {
	return nil
}

func (r *randPendingList) Stop() error {
	return nil
}

func (r *randPendingList) IsRunning() bool {
	return true
}

type subscriber struct {
	index int
	peer  peer.StatusPeer
}

var _ peer.Subscriber = (*subscriber)(nil)

func (*subscriber) NotifyStatusChanged(peer.Identifier) {}
