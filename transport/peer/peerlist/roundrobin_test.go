package peerlist

import (
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
	agent *transporttest.MockPeerAgent,
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

			s.expectedCreateErr = errors.ErrNoPeerToSelect("Test!!")
			s.agent.EXPECT().RetainPeer(s.pids[0], gomock.Any()).Return(nil, s.expectedCreateErr)
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop release error"

			s.agent = transporttest.NewMockAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			expectedErr := errors.ErrAgentHasNoReferenceToPeer{}
			s.agent.EXPECT().ReleasePeer(s.expectedPeers[0], gomock.Any()).Return(expectedErr)

			s.peerListActions = []transporttest.PeerListAction{
				transporttest.StartAction{},
				transporttest.StopAction{
					ExpectedErr: expectedErr,
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

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			allPids := createPeerIDs([]string{"1", "2"})

			s.pids = []transport.PeerIdentifier{allPids[0]}

			s.expectedPeers = createPeers(mockCtrl, s.agent, allPids)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()
				return pl.Add(allPids[1])
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
				{
					peer: s.expectedPeers[0],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start remove"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			s.pids = createPeerIDs([]string{"1", "2"})
			initialPeers := createPeers(mockCtrl, s.agent, s.pids)

			s.expectedPeers = []transport.Peer{initialPeers[1]}

			removedPeers := []transport.Peer{initialPeers[0]}
			preparePeerReleases(s.agent, removedPeers)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()
				return pl.Remove(removedPeers[0])
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start addremoveaddremoveaddremoveaddremoveaddremoveaddremove"
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)

			initialPeerIDs := createPeerIDs([]string{"1", "2", "3", "4"})
			initialPeers := createPeers(mockCtrl, s.agent, initialPeerIDs)

			addedRemovedPeerIDs := createPeerIDs([]string{"5", "6"})
			createPeers(mockCtrl, s.agent, addedRemovedPeerIDs)

			addedPeerIDs := createPeerIDs([]string{"7", "8"})
			addedPeers := createPeers(mockCtrl, s.agent, addedPeerIDs)

			removedPeerIDs := initialPeerIDs[2:]

			allRemovedPeerIDs := append(addedRemovedPeerIDs, removedPeerIDs...)
			preparePeerIdentifierReleases(s.agent, allRemovedPeerIDs)

			s.pids = initialPeerIDs
			s.expectedPeers = append(addedPeers, initialPeers[:2]...)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()

				for _, pid := range addedRemovedPeerIDs {
					err := pl.Add(pid)
					if err != nil {
						return err
					}
				}

				for _, pid := range removedPeerIDs {
					err := pl.Remove(pid)
					if err != nil {
						return err
					}
				}

				for _, pid := range addedPeerIDs {
					err := pl.Add(pid)
					if err != nil {
						return err
					}
				}

				for _, pid := range addedRemovedPeerIDs {
					err := pl.Remove(pid)
					if err != nil {
						return err
					}
				}
				return nil
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: initialPeers[0],
				},
				{
					peer: initialPeers[1],
				},
				{
					peer: addedPeers[0],
				},
				{
					peer: addedPeers[1],
				},
				{
					peer: initialPeers[0],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "add duplicate"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			duplicatePid := s.pids[0]
			s.agent.EXPECT().RetainPeer(duplicatePid, s.pl).Return(s.expectedPeers[0], nil)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()
				return pl.Add(duplicatePid)
			}

			s.expectedError = errors.ErrPeerAlreadyInList{
				Peer:     s.expectedPeers[0],
				PeerList: s.pl,
			}
			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
				{
					peer: s.expectedPeers[0],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "add retain error"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			newPid := createPeerIDs([]string{"3"})[0]
			s.expectedError = errors.ErrInvalidPeerType{
				ExpectedType:   "test",
				PeerIdentifier: newPid,
			}
			s.agent.EXPECT().RetainPeer(newPid, s.pl).Return(nil, s.expectedError)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()
				return pl.Add(newPid)
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
				{
					peer: s.expectedPeers[0],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "remove peer not in list"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.pl, _ = NewRoundRobin(s.pids, s.agent)

			removedPeerID := createPeerIDs([]string{"3"})[0]
			s.expectedError = errors.ErrPeerNotInList{
				PeerIdentifier: removedPeerID,
				PeerList:       s.pl,
			}
			s.agent.EXPECT().ReleasePeer(removedPeerID, s.pl).Return(nil)

			s.appliedFunc = func(pl *roundRobin) error {
				pl.Start()
				return pl.Remove(removedPeerID)
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
				{
					peer: s.expectedPeers[0],
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

			assert.Equal(t, len(tt.expectedPeers), len(pl.peerToNode))
			for _, expectedPeer := range tt.expectedPeers {
				actualPeer := pl.peerToNode[expectedPeer.Identifier()].Value.(transport.Peer)
				assert.Equal(t, expectedPeer, actualPeer)
			}

			assert.Equal(t, tt.expectedStarted, pl.started.Load())
		})
	}
}
