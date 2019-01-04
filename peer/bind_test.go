// Copyright (c) 2019 Uber Technologies, Inc.
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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	. "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/x/peerheap"
	"go.uber.org/yarpc/yarpctest"
)

func TestBind(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	list := peertest.NewMockChooserList(mockCtrl)

	chooser := Bind(list, BindPeers([]peer.Identifier{
		hostport.PeerIdentifier("x"),
		hostport.PeerIdentifier("y"),
	}))

	list.EXPECT().IsRunning().Return(false)
	assert.Equal(t, false, chooser.IsRunning(), "chooser should not be running")

	list.EXPECT().Start().Return(nil)
	list.EXPECT().Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.PeerIdentifier("x"),
			hostport.PeerIdentifier("y"),
		},
	})
	assert.NoError(t, chooser.Start(), "start without error")

	list.EXPECT().IsRunning().Return(true)
	assert.Equal(t, true, chooser.IsRunning(), "chooser should be running")

	list.EXPECT().Stop().Return(nil)
	list.EXPECT().Update(peer.ListUpdates{
		Removals: []peer.Identifier{
			hostport.PeerIdentifier("x"),
			hostport.PeerIdentifier("y"),
		},
	})
	assert.NoError(t, chooser.Stop(), "stop without error")

	list.EXPECT().IsRunning().Return(false)
	assert.Equal(t, false, chooser.IsRunning(), "chooser should not be running")
}

func TestBindRealList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	transport := yarpctest.NewFakeTransport()
	list := peerheap.New(transport)

	chooser := Bind(list, BindPeers([]peer.Identifier{
		hostport.PeerIdentifier("x"),
		hostport.PeerIdentifier("y"),
	}))

	assert.False(t, chooser.IsRunning(), "chooser should not be running")
	assert.NoError(t, chooser.Start(), "start without error")
	assert.True(t, chooser.IsRunning(), "chooser should be running")
	assert.NoError(t, chooser.Stop(), "start without error")
	assert.False(t, chooser.IsRunning(), "chooser should not be running")
}
