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

package tworandomchoices

import (
	"context"
	"math/rand"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	peerlist "go.uber.org/yarpc/peer/peerlist/v2"
)

type twoRandomChoicesList struct {
	subscribers []*subscriber
	random      *rand.Rand
}

func newTwoRandomChoicesList(cap int, source rand.Source) *twoRandomChoicesList {
	return &twoRandomChoicesList{
		subscribers: make([]*subscriber, 0, cap),
		random:      rand.New(source),
	}
}

var _ peerlist.Implementation = (*twoRandomChoicesList)(nil)

func (l *twoRandomChoicesList) Add(peer peer.StatusPeer, _ peer.Identifier) peer.Subscriber {
	index := len(l.subscribers)
	l.subscribers = append(l.subscribers, &subscriber{
		index: index,
		peer:  peer,
	})
	return l.subscribers[index]
}

func (l *twoRandomChoicesList) Remove(peer peer.StatusPeer, _ peer.Identifier, ps peer.Subscriber) {
	sub, ok := ps.(*subscriber)
	if !ok || len(l.subscribers) == 0 {
		return
	}
	index := sub.index
	last := len(l.subscribers) - 1
	l.subscribers[index] = l.subscribers[last]
	l.subscribers[index].index = index
	l.subscribers = l.subscribers[0:last]
}

func (l *twoRandomChoicesList) Choose(_ context.Context, _ *transport.Request) peer.StatusPeer {
	numSubs := len(l.subscribers)
	if numSubs == 0 {
		return nil
	}
	if numSubs == 1 {
		return l.subscribers[0].peer
	}
	i := l.random.Intn(numSubs)
	j := i + 1 + l.random.Intn(numSubs-1)
	if j >= numSubs {
		j -= numSubs
	}
	if l.pending(i) > l.pending(j) {
		i = j
	}
	return l.subscribers[i].peer
}

func (l *twoRandomChoicesList) pending(index int) int {
	return l.subscribers[index].peer.Status().PendingRequestCount
}

func (l *twoRandomChoicesList) Start() error {
	return nil
}

func (l *twoRandomChoicesList) Stop() error {
	return nil
}

func (l *twoRandomChoicesList) IsRunning() bool {
	return true
}

type subscriber struct {
	index int
	peer  peer.StatusPeer
}

var _ peer.Subscriber = (*subscriber)(nil)

func (*subscriber) NotifyStatusChanged(peer.Identifier) {}
