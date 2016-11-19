package list

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/peertest"
	"go.uber.org/yarpc/transport"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

func TestSingleList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type expectedChooseResult struct {
		peer peer.Peer
		err  error
	}

	type testStruct struct {
		msg                   string
		pid                   peer.Identifier
		agent                 *peertest.MockAgent
		appliedFunc           func(*single) error
		expectedPeerID        peer.Identifier
		expectedPeer          peer.Peer
		expectedAgent         peer.Agent
		expectedStarted       bool
		expectedErr           error
		expectedChooseResults []expectedChooseResult
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "setup"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.appliedFunc = func(pl *single) error {
				return nil
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "stop before start"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.appliedFunc = func(pl *single) error {
				return pl.Stop()
			}

			s.expectedErr = peer.ErrPeerListNotStarted("single")
			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "choose before start"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.appliedFunc = func(pl *single) error {
				return nil
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			s.expectedChooseResults = []expectedChooseResult{{
				peer: nil,
				err:  peer.ErrPeerListNotStarted("single"),
			}}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start and choose"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.expectedPeer = peertest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.appliedFunc = func(pl *single) error {
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
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.expectedErr = fmt.Errorf("test error")
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(nil, s.expectedErr)

			s.appliedFunc = func(pl *single) error {
				return pl.Start()
			}

			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start twice"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.expectedPeer = peertest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.appliedFunc = func(pl *single) error {
				pl.Start()
				return pl.Start()
			}

			s.expectedErr = peer.ErrPeerListAlreadyStarted("single")
			s.expectedPeerID = s.pid
			s.expectedAgent = s.agent
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop"
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			p := peertest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(p, nil)
			s.agent.EXPECT().ReleasePeer(s.pid, gomock.Any()).Return(nil)

			s.appliedFunc = func(pl *single) error {
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
			s.pid = peertest.NewMockIdentifier(mockCtrl)
			s.agent = peertest.NewMockAgent(mockCtrl)

			s.expectedPeer = peertest.NewMockPeer(mockCtrl)
			s.agent.EXPECT().RetainPeer(s.pid, gomock.Any()).Return(s.expectedPeer, nil)

			s.expectedErr = peer.ErrAgentHasNoReferenceToPeer{}
			s.agent.EXPECT().ReleasePeer(s.pid, gomock.Any()).Return(s.expectedErr)

			s.appliedFunc = func(pl *single) error {
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
		pl := NewSingle(tt.pid, tt.agent).(*single)

		err := tt.appliedFunc(pl)

		assert.Equal(t, tt.expectedErr, err, tt.msg)
		assert.Equal(t, tt.expectedAgent, pl.agent, tt.msg)
		assert.Equal(t, tt.expectedPeerID, pl.initialPeerID, tt.msg)
		assert.Equal(t, tt.expectedPeer, pl.p, tt.msg)
		assert.Equal(t, tt.expectedStarted, pl.started, tt.msg)

		for _, expectedResult := range tt.expectedChooseResults {
			p, err := pl.ChoosePeer(context.Background(), &transport.Request{})

			assert.Equal(t, expectedResult.peer, p, tt.msg)
			assert.True(t, expectedResult.peer == p, tt.msg)
			assert.Equal(t, expectedResult.err, err, tt.msg)
		}
	}
}
