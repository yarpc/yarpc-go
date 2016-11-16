package peerlist

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// gomock has difficulty seeing between mock objects of the same type, we need to define
// the peerIdentifiers ourself or gomock will lump all mocking into a single bucket.
type mockPeerIdentifier string

func (p mockPeerIdentifier) Identifier() string {
	return string(p)
}

func createPeerIDs(
	peerIDStrs []string,
) []transport.PeerIdentifier {
	pids := []transport.PeerIdentifier{}
	for _, id := range peerIDStrs {
		pids = append(pids, mockPeerIdentifier(id))
	}
	return pids
}

func createPeers(
	mockCtrl *gomock.Controller,
	agent *transporttest.MockAgent,
	pids []transport.PeerIdentifier,
) []transport.Peer {
	peers := []transport.Peer{}
	for _, pid := range pids {
		peer := transporttest.NewMockPeer(mockCtrl)
		peer.EXPECT().Identifier().Return(pid.Identifier()).AnyTimes()

		agent.EXPECT().RetainPeer(pid, gomock.Any()).Return(peer, nil)

		peers = append(peers, peer)
	}
	return peers
}

func preparePeerReleases(
	agent *transporttest.MockAgent,
	peers []transport.Peer,
) {
	for _, peer := range peers {
		agent.EXPECT().ReleasePeer(transport.PeerIdentifier(peer), gomock.Any()).Return(nil)
	}
}

func preparePeerIdentifierReleases(
	agent *transporttest.MockAgent,
	peerIDs []transport.PeerIdentifier,
) {
	for _, pid := range peerIDs {
		agent.EXPECT().ReleasePeer(pid, gomock.Any()).Return(nil)
	}
}

func TestRoundRobinList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		msg               string
		pids              []transport.PeerIdentifier
		agent             *transporttest.MockAgent
		pl                *RoundRobin
		peerListActions   []transporttest.PeerListAction
		expectedCreateErr error
		expectedPeers     []transport.Peer
		expectedStarted   bool
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "setup"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			return
		}(),
		func() (s testStruct) {
			s.msg = "start"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2", "3", "4", "5", "6"})
			peers := createPeers(mockCtrl, s.agent, s.pids)
			preparePeerReleases(s.agent, peers)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.StopAction{},
				transporttest.ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			}

			s.expectedPeers = []transport.Peer{}
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many and choose"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2", "3", "4", "5", "6"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[2],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[3],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[4],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[5],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start twice"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.StartAction{
					ExpectedErr: errors.ErrPeerListAlreadyStarted("RoundRobinList"),
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "stop no start"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StopAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			}

			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start retain error"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})

			retainErr := errors.ErrNoPeerToSelect("Test!!")
			s.agent.EXPECT().RetainPeer(s.pids[0], gomock.Any()).Return(nil, retainErr)

			s.expectedCreateErr = errors.Errors{retainErr}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop release error"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			peers := createPeers(mockCtrl, s.agent, s.pids)

			expectedErr := errors.ErrAgentHasNoReferenceToPeer{}
			s.agent.EXPECT().ReleasePeer(peers[0], gomock.Any()).Return(expectedErr)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.StopAction{
					ExpectedErr: errors.Errors{expectedErr},
				},
			}

			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "choose before start"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
				transporttest.ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			}

			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start choose no peers"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.ChooseAction{
					ExpectedErr: errors.ErrNoPeerToSelect("RoundRobinList"),
				},
				transporttest.ChooseAction{
					ExpectedErr: errors.ErrNoPeerToSelect("RoundRobinList"),
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start add"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			allPids := createPeerIDs([]string{"1", "2"})

			s.pids = []transport.PeerIdentifier{allPids[0]}

			s.expectedPeers = createPeers(mockCtrl, s.agent, allPids)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.AddAction{
					InputPeerID: allPids[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start remove"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			initialPeers := createPeers(mockCtrl, s.agent, s.pids)

			s.expectedPeers = []transport.Peer{initialPeers[1]}

			removedPeerIDs := []transport.PeerIdentifier{s.pids[0]}
			preparePeerIdentifierReleases(s.agent, removedPeerIDs)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.RemoveAction{
					InputPeerID: s.pids[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start add many and remove many"
			s.agent = transporttest.NewMockAgent(mockCtrl)

			initialPeerIDs := createPeerIDs([]string{"1", "2", "3", "4"})
			initialPeers := createPeers(mockCtrl, s.agent, initialPeerIDs)
			s.pids = initialPeerIDs

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
			}

			addedRemovedPeerIDs := createPeerIDs([]string{"5", "6"})
			createPeers(mockCtrl, s.agent, addedRemovedPeerIDs)

			addedPeerIDs := createPeerIDs([]string{"7", "8"})
			addedPeers := createPeers(mockCtrl, s.agent, addedPeerIDs)
			allAddedPeerIDs := append(addedRemovedPeerIDs, addedPeerIDs...)

			for _, pid := range allAddedPeerIDs {
				s.peerListActions = append(s.peerListActions, transporttest.AddAction{
					InputPeerID: pid,
				})
			}

			removedPeerIDs := initialPeerIDs[2:]
			allRemovedPeerIDs := append(addedRemovedPeerIDs, removedPeerIDs...)
			preparePeerIdentifierReleases(s.agent, allRemovedPeerIDs)

			for _, pid := range allRemovedPeerIDs {
				s.peerListActions = append(s.peerListActions, transporttest.RemoveAction{
					InputPeerID: pid,
				})
			}

			s.expectedPeers = append(addedPeers, initialPeers[:2]...)

			s.peerListActions = append(
				s.peerListActions,
				transporttest.ChooseAction{
					ExpectedPeer: initialPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: initialPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: addedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: addedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: initialPeers[0],
				},
			)

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "add duplicate"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			duplicatePid := s.pids[0]
			s.agent.EXPECT().RetainPeer(duplicatePid, s.pl).Return(s.expectedPeers[0], nil)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.AddAction{
					InputPeerID: duplicatePid,
					ExpectedErr: errors.ErrPeerAlreadyInList{
						Peer: s.expectedPeers[0],
					},
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "add retain error"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			newPid := createPeerIDs([]string{"3"})[0]
			expectedError := errors.ErrInvalidPeerType{
				ExpectedType:   "test",
				PeerIdentifier: newPid,
			}
			s.agent.EXPECT().RetainPeer(newPid, s.pl).Return(nil, expectedError)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.AddAction{
					InputPeerID: newPid,
					ExpectedErr: expectedError,
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "remove peer not in list"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			removedPeerID := createPeerIDs([]string{"3"})[0]

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.RemoveAction{
					InputPeerID: removedPeerID,
					ExpectedErr: errors.ErrPeerNotInList{
						PeerIdentifier: removedPeerID,
					},
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[1],
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "remove release error"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			peers := createPeers(mockCtrl, s.agent, s.pids)
			s.expectedPeers = []transport.Peer{peers[0]}

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			removedPid := s.pids[1]
			expectedError := errors.ErrAgentHasNoReferenceToPeer{
				Agent:          s.agent,
				PeerIdentifier: removedPid,
			}
			s.agent.EXPECT().ReleasePeer(removedPid, s.pl).Return(expectedError)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.RemoveAction{
					InputPeerID: removedPid,
					ExpectedErr: expectedError,
				},
				transporttest.ChooseAction{
					ExpectedPeer: s.expectedPeers[0],
				},
			}

			s.expectedStarted = true
			return
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			var pl *RoundRobin
			var err error
			pl = tt.pl
			if pl == nil {
				pl, err = NewRoundRobin(tt.pids, tt.agent)
				assert.Equal(t, tt.expectedCreateErr, err)

				if pl == nil {
					return
				}
			}

			transporttest.ApplyPeerListActions(t, pl, tt.peerListActions)

			assert.Equal(t, len(tt.expectedPeers), len(pl.pr.peerToNode))
			for _, expectedPeer := range tt.expectedPeers {
				node, ok := pl.pr.peerToNode[expectedPeer.Identifier()]
				assert.True(t, ok, fmt.Sprintf("expected peer: %s was not in peerlist", expectedPeer.Identifier()))
				actualPeer := node.getPeer()
				assert.Equal(t, expectedPeer, actualPeer)
			}

			assert.Equal(t, tt.expectedStarted, pl.started.Load())
		})
	}
}
