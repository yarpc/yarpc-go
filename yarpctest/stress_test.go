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

package yarpctest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

func TestStress(t *testing.T) {
	test := ListStressTest{
		Workers:  1,
		Duration: time.Second,
		Timeout:  10 * time.Millisecond,
		New: func(t peer.Transport) peer.ChooserList {
			return newMRUList(t)
		},
	}
	report := test.Run(t)
	report.Log(t)
	test.Log(t)
}

var _ peer.ChooserList = (*mruList)(nil)

type mruList struct {
	transport peer.Transport
	peer      peer.Peer
	mu        sync.Mutex
}

func newMRUList(t peer.Transport) *mruList {
	return &mruList{transport: t}
}

func (l *mruList) Start() error {
	return nil
}

func (l *mruList) Stop() error {
	return nil
}

func (l *mruList) IsRunning() bool {
	return true
}

func (l *mruList) Choose(context.Context, *transport.Request) (peer peer.Peer, onFinish func(error), err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.peer == nil {
		return nil, nil, fmt.Errorf("no peer available")
	}

	return l.peer, func(error) {}, nil
}

func (l *mruList) Update(updates peer.ListUpdates) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, pid := range updates.Additions {
		l.update(pid)
	}

	if (len(updates.Additions)+len(updates.Removals))%2 == 0 {
		return fmt.Errorf("can you handle this")
	}
	if len(updates.Additions) == 0 {
		return fmt.Errorf("parting is such sweet sorrow")
	}
	return nil
}

func (l *mruList) update(id peer.Identifier) {
	if peer, err := l.transport.RetainPeer(id, nopSub); err == nil {
		if l.peer != nil {
			_ = l.transport.ReleasePeer(id, nopSub)
		}
		l.peer = peer
	}
}

type _nopSub struct{}

func (_nopSub) NotifyStatusChanged(peer.Identifier) {}

var nopSub = _nopSub{}
