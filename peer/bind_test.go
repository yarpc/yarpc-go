package peer_test

import (
	"testing"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	. "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
