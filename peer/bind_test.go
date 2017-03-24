// Copyright (c) 2017 Uber Technologies, Inc.
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

package peer_test

import (
	"testing"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	intsync "go.uber.org/yarpc/internal/sync"
	. "go.uber.org/yarpc/peer"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	list := peertest.NewMockChooserList(mockCtrl)
	life := &lowlife{once: intsync.Once()}

	list.EXPECT().Start().Return(nil)
	list.EXPECT().Update(peer.ListUpdates{})
	list.EXPECT().Stop().Return(nil)

	binder := func(cl peer.List) transport.Lifecycle {
		cl.Update(peer.ListUpdates{})
		return life
	}

	chooser := Bind(list, binder)
	assert.Equal(t, false, life.IsRunning(), "binder should not be running")
	chooser.Start()
	assert.Equal(t, true, life.IsRunning(), "binder should be running")
	chooser.Stop()
	assert.Equal(t, false, life.IsRunning(), "binder should not be running")
}

type lowlife struct {
	once intsync.LifecycleOnce
}

func (ll *lowlife) Start() error {
	return ll.once.Start(nil)
}

func (ll *lowlife) Stop() error {
	return ll.once.Stop(nil)
}

func (ll *lowlife) IsRunning() bool {
	return ll.once.IsRunning()
}
