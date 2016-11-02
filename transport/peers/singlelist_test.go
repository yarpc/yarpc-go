package peers

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

func TestSinglePeerList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		pl          *singlePeerList
		appliedFunc func(transport.PeerList)
		assertFunc  func(*singlePeerList)
	}
	tests := []testStruct{
		func() (s testStruct) {
			msg := "setup"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			s.appliedFunc = func(pl transport.PeerList) {}
			s.assertFunc = func(pl *singlePeerList) {
				assert.Nil(t, pl.peer, msg)
				assert.Equal(t, agent, pl.agent, msg)
				assert.Equal(t, pi, pl.peerID, msg)
				assert.Equal(t, false, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "stop before start"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)
			var err error
			s.appliedFunc = func(pl transport.PeerList) {
				err = pl.Stop()
			}
			s.assertFunc = func(pl *singlePeerList) {
				assert.NotNil(t, err, msg)
				assert.IsType(t, errors.ErrOutboundNotStarted(""), err, msg)
				assert.Equal(t, false, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "choose before start"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)
			var err error
			var peer transport.Peer
			s.appliedFunc = func(pl transport.PeerList) {
				peer, err = pl.ChoosePeer(context.Background(), &transport.Request{})
			}
			s.assertFunc = func(pl *singlePeerList) {
				assert.Nil(t, peer, msg)
				assert.NotNil(t, err, msg)
				assert.IsType(t, errors.ErrOutboundNotStarted(""), err, msg)
				assert.Equal(t, false, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "start and choose"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			expectedPeer := transporttest.NewMockPeer(mockCtrl)
			agent.EXPECT().RetainPeer(pi, s.pl).Return(expectedPeer, nil)

			var startErr error
			var chooseErr error
			var peer transport.Peer
			s.appliedFunc = func(pl transport.PeerList) {
				startErr = pl.Start()
				peer, chooseErr = pl.ChoosePeer(context.Background(), &transport.Request{})
			}

			s.assertFunc = func(pl *singlePeerList) {
				assert.Nil(t, startErr, msg)
				assert.Nil(t, chooseErr, msg)
				assert.Equal(t, expectedPeer, peer, msg)
				assert.Equal(t, true, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "start with agent error"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			expectedErr := fmt.Errorf("test error")
			agent.EXPECT().RetainPeer(pi, s.pl).Return(nil, expectedErr)

			var startErr error
			s.appliedFunc = func(pl transport.PeerList) {
				startErr = pl.Start()
			}

			s.assertFunc = func(pl *singlePeerList) {
				assert.Equal(t, expectedErr, startErr, msg)
				assert.Equal(t, false, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "start twice"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			peer := transporttest.NewMockPeer(mockCtrl)
			agent.EXPECT().RetainPeer(pi, s.pl).Return(peer, nil)

			var startErr error
			s.appliedFunc = func(pl transport.PeerList) {
				_ = pl.Start()
				startErr = pl.Start()
			}

			s.assertFunc = func(pl *singlePeerList) {
				assert.NotNil(t, startErr, msg)
				assert.IsType(t, errors.ErrOutboundAlreadyStarted(""), startErr, msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "start stop"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			peer := transporttest.NewMockPeer(mockCtrl)
			agent.EXPECT().RetainPeer(pi, s.pl).Return(peer, nil)
			agent.EXPECT().ReleasePeer(pi, s.pl).Return(nil)

			var startErr error
			var stopErr error
			s.appliedFunc = func(pl transport.PeerList) {
				startErr = pl.Start()
				stopErr = pl.Stop()
			}

			s.assertFunc = func(pl *singlePeerList) {
				assert.Nil(t, startErr, msg)
				assert.Nil(t, stopErr, msg)
				assert.Equal(t, false, pl.started.Load(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "start stop release failure"
			pi := transporttest.NewMockPeerIdentifier(mockCtrl)
			agent := transporttest.NewMockPeerAgent(mockCtrl)
			s.pl = NewSinglePeerList(pi, agent).(*singlePeerList)

			peer := transporttest.NewMockPeer(mockCtrl)
			agent.EXPECT().RetainPeer(pi, s.pl).Return(peer, nil)
			agent.EXPECT().ReleasePeer(pi, s.pl).Return(errors.ErrAgentHasNoReferenceToPeer{})

			var startErr error
			var stopErr error
			s.appliedFunc = func(pl transport.PeerList) {
				startErr = pl.Start()
				stopErr = pl.Stop()
			}

			s.assertFunc = func(pl *singlePeerList) {
				assert.Nil(t, startErr, msg)
				assert.Equal(t, errors.ErrAgentHasNoReferenceToPeer{}, stopErr, msg)
			}
			return
		}(),
	}

	for _, tt := range tests {
		tt.appliedFunc(tt.pl)

		tt.assertFunc(tt.pl)
	}
}
