package peerlist

import (
	"context"
	"testing"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func createPeers(
	mockCtrl *gomock.Controller,
	peerIDStrs []string,
	forEach func(*transporttest.MockPeerIdentifier, *transporttest.MockPeer, *transporttest.MockPeerAgent),
) (transport.PeerAgent, []transport.PeerIdentifier, []transport.Peer) {
	agent := transporttest.NewMockPeerAgent(mockCtrl)
	pids := []transport.PeerIdentifier{}
	peers := []transport.Peer{}
	for _, id := range peerIDStrs {
		pid := transporttest.NewMockPeerIdentifier(mockCtrl)
		peer := transporttest.NewMockPeer(mockCtrl)

		pid.EXPECT().Identifier().Return(id).AnyTimes()
		peer.EXPECT().Identifier().Return(id).AnyTimes()

		forEach(pid, peer, agent)

		pids = append(pids, pid)
		peers = append(peers, peer)
	}
	return agent, pids, peers
}

func TestRoundRobinList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		msg                    string
		pids                   []transport.PeerIdentifier
		agent                  transport.PeerAgent
		appliedFunc            func(transport.PeerList) error
		expectedInitialPeerIDs []transport.PeerIdentifier
		expectedPeers          []transport.Peer
		expectedNextPeer       transport.Peer
		expectedStarted        bool
		expectedError          error
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "setup"

			s.agent, s.pids, _ = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error { return nil }

			s.expectedInitialPeerIDs = s.pids
			s.expectedPeers = []transport.Peer{}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				return pl.Start()
			}

			s.expectedNextPeer = s.expectedPeers[0]
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop"

			s.agent, s.pids, _ = createPeers(
				mockCtrl,
				[]string{"1", "2", "3", "4", "5", "6"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
					a.EXPECT().ReleasePeer(p, gomock.Any()).Return(nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				err := pl.Start()
				if err != nil {
					return err
				}
				return pl.Stop()
			}

			s.expectedNextPeer = nil
			s.expectedInitialPeerIDs = s.pids
			s.expectedPeers = []transport.Peer{}
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1", "2", "3", "4", "5", "6"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				return pl.Start()
			}

			s.expectedNextPeer = s.expectedPeers[0] // The first inserted peer
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many choose 1"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1", "2", "3", "4", "5", "6"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				err := pl.Start()
				assert.Nil(t, err, s.msg)

				chosenPeer, err := pl.ChoosePeer(context.Background(), &transport.Request{})
				assert.Equal(t, s.expectedPeers[0], chosenPeer)
				return err
			}

			s.expectedNextPeer = s.expectedPeers[1] // The second inserted peer
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many choose 4"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1", "2", "3", "4", "5", "6"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				err := pl.Start()
				assert.Nil(t, err, s.msg)

				for i := 0; i < 4; i++ {
					_, err = pl.ChoosePeer(context.Background(), &transport.Request{})
				}
				return err
			}

			s.expectedNextPeer = s.expectedPeers[4] // The fifth inserted peer
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many choose 7"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1", "2", "3", "4", "5", "6"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				err := pl.Start()
				assert.Nil(t, err, s.msg)

				for i := 0; i < 7; i++ {
					_, err = pl.ChoosePeer(context.Background(), &transport.Request{})
				}
				return err
			}

			s.expectedNextPeer = s.expectedPeers[0] // The first peer, we should be looping
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start twice"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				_ = pl.Start()
				return pl.Start()
			}

			s.expectedError = errors.ErrPeerListAlreadyStarted("RoundRobinList")
			s.expectedNextPeer = s.expectedPeers[0]
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "stop no start"

			s.agent, s.pids, _ = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				return pl.Stop()
			}

			s.expectedError = errors.ErrPeerListNotStarted("RoundRobinList")
			s.expectedNextPeer = nil
			s.expectedInitialPeerIDs = s.pids
			s.expectedPeers = []transport.Peer{}
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start retain error"

			err := errors.ErrNoPeerToSelect("Test!!")
			s.agent, s.pids, _ = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(nil, err)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				return pl.Start()
			}

			s.expectedError = err
			s.expectedNextPeer = nil
			s.expectedInitialPeerIDs = s.pids
			s.expectedPeers = []transport.Peer{}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop release error"

			err := errors.ErrAgentHasNoReferenceToPeer{}
			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
					a.EXPECT().RetainPeer(pid, gomock.Any()).Return(p, nil)
					a.EXPECT().ReleasePeer(p, gomock.Any()).Return(err)
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				_ = pl.Start()
				return pl.Stop()
			}

			s.expectedError = err
			s.expectedNextPeer = s.expectedPeers[0]
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "choose before start"

			s.agent, s.pids, _ = createPeers(
				mockCtrl,
				[]string{"1"},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				_, err := pl.ChoosePeer(context.Background(), &transport.Request{})
				return err
			}

			s.expectedError = errors.ErrPeerListNotStarted("RoundRobinList")
			s.expectedNextPeer = nil
			s.expectedPeers = []transport.Peer{}
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start choose no peers"

			s.agent, s.pids, s.expectedPeers = createPeers(
				mockCtrl,
				[]string{},
				func(pid *transporttest.MockPeerIdentifier, p *transporttest.MockPeer, a *transporttest.MockPeerAgent) {
				},
			)

			s.appliedFunc = func(pl transport.PeerList) error {
				pl.Start()
				_, err := pl.ChoosePeer(context.Background(), &transport.Request{})
				return err
			}

			s.expectedError = errors.ErrNoPeerToSelect("RoundRobinList")
			s.expectedNextPeer = nil
			s.expectedInitialPeerIDs = s.pids
			s.expectedStarted = true
			return
		}(),
	}

	for _, tt := range tests {
		peerList := NewRoundRobin(tt.pids, tt.agent).(*roundRobin)

		err := tt.appliedFunc(peerList)
		assert.Equal(t, tt.expectedError, err, tt.msg)

		assert.Equal(t, len(tt.expectedInitialPeerIDs), len(peerList.initialPeerIDs), tt.msg)
		for _, expectedPeerID := range tt.expectedInitialPeerIDs {
			assert.Equal(t, expectedPeerID, peerList.initialPeerIDs[expectedPeerID.Identifier()], tt.msg)
		}

		assert.Equal(t, len(tt.expectedPeers), len(peerList.peerToNode), tt.msg)
		for _, expectedPeer := range tt.expectedPeers {
			actualPeer := peerList.peerToNode[expectedPeer.Identifier()].peer
			assert.Equal(t, expectedPeer, actualPeer, tt.msg)
		}

		if tt.expectedNextPeer == nil {
			assert.Nil(t, peerList.nextNode, tt.msg)
		} else {
			assert.Equal(t, tt.expectedNextPeer, peerList.nextNode.peer, tt.msg)
		}

		assert.Equal(t, tt.expectedStarted, peerList.started.Load(), tt.msg)
	}
}
