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

package yarpctworandomchoices

import (
	"context"
	"math/rand"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeerlist"
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

var _ yarpcpeerlist.Implementation = (*twoRandomChoicesList)(nil)

func (l *twoRandomChoicesList) Add(peer yarpc.StatusPeer, _ yarpc.Identifier) yarpc.Subscriber {
	index := len(l.subscribers)
	l.subscribers = append(l.subscribers, &subscriber{
		index: index,
		peer:  peer,
	})
	return l.subscribers[index]
}

func (l *twoRandomChoicesList) Remove(peer yarpc.StatusPeer, _ yarpc.Identifier, ps yarpc.Subscriber) {
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

func (l *twoRandomChoicesList) Choose(_ context.Context, _ *yarpc.Request) yarpc.StatusPeer {
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

type subscriber struct {
	index int
	peer  yarpc.StatusPeer
}

var _ yarpc.Subscriber = (*subscriber)(nil)

func (*subscriber) NotifyStatusChanged(yarpc.Identifier) {}
