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
