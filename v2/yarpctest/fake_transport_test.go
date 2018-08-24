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

package yarpctest_test

import (
	"fmt"
	"sync"
	"testing"

	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctest"
)

type testPeer struct {
	id string
}

type testSubscriber struct{}

func (s testSubscriber) NotifyStatusChanged(yarpcpeer.Identifier) {}

func (p testPeer) Identifier() string {
	return p.id
}

func TestFakeTransport(t *testing.T) {
	trans := yarpctest.NewFakeTransport()

	wait := make(chan struct{}, 0)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			p := &testPeer{id: fmt.Sprintf("foo %d", i%10)}
			<-wait
			trans.Peer(p)

			wg.Done()
		}(i)
	}
	close(wait)
	wg.Wait()
}

func TestRetainReleasePeer(t *testing.T) {
	trans := yarpctest.NewFakeTransport()

	wait := make(chan struct{}, 0)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			p := &testPeer{id: fmt.Sprintf("foo %d", i%10)}
			s := &testSubscriber{}
			<-wait
			myPeer := trans.Peer(p)
			trans.RetainPeer(myPeer, s)
			trans.ReleasePeer(myPeer, s)

			wg.Done()
		}(i)
	}
	close(wait)
	wg.Wait()
}
