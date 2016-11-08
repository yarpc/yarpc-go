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
