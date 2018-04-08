package randpeer

import (
	"context"
	"math/rand"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
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

var _ peer.ListImplementation = (*randomList)(nil)

func (r *randomList) Add(peer peer.StatusPeer) peer.Subscriber {
	index := len(r.subscribers)
	r.subscribers = append(r.subscribers, &subscriber{
		index: index,
		peer:  peer,
	})
	return r.subscribers[index]
}

func (r *randomList) Remove(peer peer.StatusPeer, sub peer.Subscriber) {
	if sub, ok := sub.(*subscriber); ok && len(r.subscribers) > 0 {
		index := sub.index
		last := len(r.subscribers) - 1
		r.subscribers[index] = r.subscribers[last]
		r.subscribers[index].index = index
		r.subscribers = r.subscribers[0:last]
	}
}

func (r *randomList) Choose(_ context.Context, _ *transport.Request) peer.StatusPeer {
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
	peer  peer.StatusPeer
}

var _ peer.Subscriber = (*subscriber)(nil)

func (s *subscriber) NotifyStatusChanged(peer.Identifier) {}
