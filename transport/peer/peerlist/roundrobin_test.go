package peerlist

import (
	"context"
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
	agent *transporttest.MockPeerAgent,
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
	agent *transporttest.MockPeerAgent,
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

	type expectedChooseResult struct {
		peer transport.Peer
		err  error
	}

	type testStruct struct {
		msg                   string
		pids                  []transport.PeerIdentifier
		agent                 *transporttest.MockPeerAgent
		pl                    transport.PeerList
		appliedFunc           func(*roundRobin) error
		expectedCreateErr     error
		expectedPeers         []transport.Peer
		expectedStarted       bool
		expectedError         error
		expectedChooseResults []expectedChooseResult
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "setup"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error { return nil }

			return
		}(),
		func() (s testStruct) {
			s.msg = "start"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				return pl.Start()
			}

			s.expectedChooseResults = []expectedChooseResult{{
				peer: s.expectedPeers[0],
				err:  nil,
			}}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2", "3", "4", "5", "6"})
			peers := createPeers(mockCtrl, s.agent, s.pids)
			preparePeerReleases(s.agent, peers)

			s.appliedFunc = func(pl *roundRobin) error {
				err := pl.Start()
				if err != nil {
					return err
				}
				return pl.Stop()
			}

			s.expectedChooseResults = []expectedChooseResult{{
				peer: nil,
				err:  errors.ErrPeerListNotStarted("RoundRobinList"),
			}}
			s.expectedPeers = []transport.Peer{}
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start many and choose"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1", "2", "3", "4", "5", "6"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				return pl.Start()
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
				{
					peer: s.expectedPeers[2],
				},
				{
					peer: s.expectedPeers[3],
				},
				{
					peer: s.expectedPeers[4],
				},
				{
					peer: s.expectedPeers[5],
				},
				{
					peer: s.expectedPeers[0],
				},
				{
					peer: s.expectedPeers[1],
				},
			}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "start twice"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				_ = pl.Start()
				return pl.Start()
			}

			s.expectedError = errors.ErrPeerListAlreadyStarted("RoundRobinList")
			s.expectedChooseResults = []expectedChooseResult{{
				peer: s.expectedPeers[0],
			}}
			s.expectedStarted = true
			return
		}(),
		func() (s testStruct) {
			s.msg = "stop no start"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				return pl.Stop()
			}

			s.expectedError = errors.ErrPeerListNotStarted("RoundRobinList")
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start retain error"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})

			s.expectedCreateErr = errors.ErrNoPeerToSelect("Test!!")
			s.agent.EXPECT().RetainPeer(s.pids[0], gomock.Any()).Return(nil, s.expectedCreateErr)
			return
		}(),
		func() (s testStruct) {
			s.msg = "start stop release error"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.expectedError = errors.ErrAgentHasNoReferenceToPeer{}
			s.agent.EXPECT().ReleasePeer(s.expectedPeers[0], gomock.Any()).Return(s.expectedError)

			s.appliedFunc = func(pl *roundRobin) error {
				_ = pl.Start()
				return pl.Stop()
			}

			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "choose before start"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{"1"})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				return nil
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					err: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
				{
					err: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			}
			s.expectedStarted = false
			return
		}(),
		func() (s testStruct) {
			s.msg = "start choose no peers"

			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.pids = createPeerIDs([]string{})
			s.expectedPeers = createPeers(mockCtrl, s.agent, s.pids)

			s.appliedFunc = func(pl *roundRobin) error {
				return pl.Start()
			}

			s.expectedChooseResults = []expectedChooseResult{
				{
					err: errors.ErrNoPeerToSelect("RoundRobinList"),
				},
				{
					err: errors.ErrNoPeerToSelect("RoundRobinList"),
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
		var pl transport.PeerList
		var err error
		pl = tt.pl
		if pl == nil {
			pl, err = NewRoundRobin(tt.pids, tt.agent)
			assert.Equal(t, tt.expectedCreateErr, err)

			if pl == nil {
				continue
			}
		}
		peerList, ok := pl.(*roundRobin)
		assert.True(t, ok, tt.msg)

		err = tt.appliedFunc(peerList)
		assert.Equal(t, tt.expectedError, err, tt.msg)

		assert.Equal(t, len(tt.expectedPeers), len(peerList.peerToNode), tt.msg)
		for _, expectedPeer := range tt.expectedPeers {
			actualPeer := peerList.peerToNode[expectedPeer.Identifier()].peer
			assert.Equal(t, expectedPeer, actualPeer, tt.msg)
		}

		assert.Equal(t, tt.expectedStarted, peerList.started.Load(), tt.msg)

		for _, expectedResult := range tt.expectedChooseResults {
			peer, err := peerList.ChoosePeer(context.Background(), &transport.Request{})

			assert.Equal(t, expectedResult.peer, peer, tt.msg)
			assert.True(t, expectedResult.peer == peer, tt.msg)
			assert.Equal(t, expectedResult.err, err, tt.msg)
		}
	}
}
