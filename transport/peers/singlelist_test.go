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

	type expectedChooseResult struct {
		peer transport.Peer
		err  error
	}

	type testStruct struct {
		msg                   string
		pid                   transport.PeerIdentifier
		agent                 *transporttest.MockPeerAgent
		appliedFunc           func(*singlePeerList) error
		expectedPeerID        transport.PeerIdentifier
		expectedPeer          transport.Peer
		expectedAgent         transport.PeerAgent
		expectedStarted       bool
		expectedErr           error
		expectedChooseResults []expectedChooseResult
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "setup"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.appliedFunc = func(pl *singlePeerList) error {
				return nil
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "stop before start"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.appliedFunc = func(pl *singlePeerList) error {
				return pl.Stop()
			}

			s.expectedErr = errors.ErrOutboundNotStarted("SinglePeerList")
			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "choose before start"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.appliedFunc = func(pl *singlePeerList) error {
				return nil
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			s.expectedChooseResults = []expectedChooseResult{{
				peer: nil,
				err:  errors.ErrOutboundNotStarted("peerlist was not started"),
			}}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start and choose"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.expectedPeer = transporttest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.appliedFunc = func(pl *singlePeerList) error {
				return pl.Start()
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = true
			s.expectedChooseResults = []expectedChooseResult{{
				peer: s.expectedPeer,
				err:  nil,
			}}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start with agent error"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.expectedErr = fmt.Errorf("test error")
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(nil, s.expectedErr)

			s.appliedFunc = func(pl *singlePeerList) error {
				return pl.Start()
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start twice"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.expectedPeer = transporttest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.appliedFunc = func(pl *singlePeerList) error {
				pl.Start()
				return pl.Start()
			}

			s.expectedErr = errors.ErrOutboundAlreadyStarted("SinglePeerList")
			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			peer := transporttest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(peer, nil)
			s.agent.EXPECT().ReleasePeer(s.pid, gomock.Any()).Return(nil)

			s.appliedFunc = func(pl *singlePeerList) error {
				err := pl.Start()
				if err != nil {
					return err
				}
				return pl.Stop()
			}

			s.expectedErr = nil
			s.expectedPeerID = s.pid
			s.expectedPeer = nil
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop release failure"
			s.pid = transporttest.NewMockPeerIdentifier(mockCtrl)
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.expectedPeer = transporttest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.expectedErr = errors.ErrAgentHasNoReferenceToPeer{}
			s.agent.EXPECT().ReleasePeer(s.pid, gomock.Any()).Return(s.expectedErr)

			s.appliedFunc = func(pl *singlePeerList) error {
				err := pl.Start()
				if err != nil {
					return err
				}
				return pl.Stop()
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
	}

	for _, tt := range tests {
		pl := NewSinglePeerList(tt.pid, tt.agent).(*singlePeerList)

		err := tt.appliedFunc(pl)

		assert.Equal(t, tt.expectedErr, err, tt.msg)
		assert.Equal(t, tt.expectedAgent, pl.agent, tt.msg)
		assert.Equal(t, tt.expectedPeerID, pl.peerID, tt.msg)
		assert.Equal(t, tt.expectedPeer, pl.peer, tt.msg)
		assert.Equal(t, tt.expectedStarted, pl.started.Load(), tt.msg)

		for _, expectedResult := range tt.expectedChooseResults {
			peer, err := pl.ChoosePeer(context.Background(), &transport.Request{})

			assert.Equal(t, expectedResult.peer, peer, tt.msg)
			assert.True(t, expectedResult.peer == peer, tt.msg)
			assert.Equal(t, expectedResult.err, err, tt.msg)
		}
	}
}
