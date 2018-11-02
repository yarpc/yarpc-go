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

package yarpcrandpeer

import (
	"context"
	"math/rand"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeerlist"
)

type randomList struct {
	subscribers []*subscriber
	random      *rand.Rand
}

func newRandomList(cap int, source rand.Source) *randomList {
	return &randomList{
		subscribers: make([]*subscriber, 0, cap),
		random:      rand.New(source),
	}
}

var _ yarpcpeerlist.Implementation = (*randomList)(nil)

func (r *randomList) Add(peer yarpc.StatusPeer, _ yarpc.Identifier) yarpc.Subscriber {
	index := len(r.subscribers)
	r.subscribers = append(r.subscribers, &subscriber{
		index: index,
		peer:  peer,
	})
	return r.subscribers[index]
}

func (r *randomList) Remove(peer yarpc.StatusPeer, _ yarpc.Identifier, ps yarpc.Subscriber) {
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

func (r *randomList) Choose(_ context.Context, _ *yarpc.Request) yarpc.StatusPeer {
	if len(r.subscribers) == 0 {
		return nil
	}
	index := r.random.Intn(len(r.subscribers))
	return r.subscribers[index].peer
}

func (r *randomList) Start() error {
	return nil
}

func (r *randomList) Stop() error {
	return nil
}

func (r *randomList) IsRunning() bool {
	return true
}

type subscriber struct {
	index int
	peer  yarpc.StatusPeer
}

var _ yarpc.Subscriber = (*subscriber)(nil)

func (*subscriber) NotifyStatusChanged(yarpc.Identifier) {}
