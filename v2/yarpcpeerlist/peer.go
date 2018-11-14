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

package yarpcpeerlist

import (
	"sync"

	yarpc "go.uber.org/yarpc/v2"
)

// peerThunk captures a peer and its corresponding subscriber,
// and serves as a subscriber by proxy.
type peerThunk struct {
	lock          sync.RWMutex
	list          *List
	id            yarpc.Identifier
	peer          yarpc.Peer
	subscriber    yarpc.Subscriber
	boundOnFinish func(error)
}

func (t *peerThunk) onFinish(error) {
	t.peer.EndRequest()
}

func (t *peerThunk) Identifier() string {
	return t.peer.Identifier()
}

func (t *peerThunk) Status() yarpc.Status {
	return t.peer.Status()
}

func (t *peerThunk) StartRequest() {
	t.peer.StartRequest()
}

func (t *peerThunk) EndRequest() {
	t.peer.EndRequest()
}

// NotifyStatusChanged forwards a status notification to the peer list and to
// the underlying identifier chooser list.
func (t *peerThunk) NotifyStatusChanged(id yarpc.Identifier) {
	t.list.notifyStatusChanged(id)

	if s := t.Subscriber(); s != nil {
		s.NotifyStatusChanged(id)
	}
}

// SetSubscriber assigns a subscriber to the subscriber thunk.
func (t *peerThunk) SetSubscriber(s yarpc.Subscriber) {
	t.lock.Lock()
	t.subscriber = s
	t.lock.Unlock()
}

// Subscriber returns the subscriber.
func (t *peerThunk) Subscriber() yarpc.Subscriber {
	t.lock.RLock()
	s := t.subscriber
	t.lock.RUnlock()
	return s
}
