package roundrobin

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc/transport/internal/errors"
	. "go.uber.org/yarpc/transport/peer/x/peerlist/roundrobin/internal"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobinList(t *testing.T) {
	type testStruct struct {
		msg                string
		inputPeerIDs       []string
		retainedPeerIDs    []string
		releasedPeerIDs    []string
		errRetainedPeerIDs []string
		retainErr          error
		errReleasedPeerIDs []string
		releaseErr         error
		peerListActions    []PeerListAction
		expectedCreateErr  error
		expectedRingPeers  []string
		expectedStarted    bool
	}
	tests := []testStruct{
		{
			msg:               "setup",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1"},
			expectedRingPeers: []string{"1"},
		},
		{
			msg:               "start",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1"},
			expectedRingPeers: []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedStarted: true,
		},
		{
			msg:             "start stop",
			inputPeerIDs:    []string{"1", "2", "3", "4", "5", "6"},
			retainedPeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			releasedPeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{},
				ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
		},
		{
			msg:               "start many and choose",
			inputPeerIDs:      []string{"1", "2", "3", "4", "5", "6"},
			retainedPeerIDs:   []string{"1", "2", "3", "4", "5", "6"},
			expectedRingPeers: []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "3"},
				ChooseAction{ExpectedPeer: "4"},
				ChooseAction{ExpectedPeer: "5"},
				ChooseAction{ExpectedPeer: "6"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:               "start twice",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1"},
			expectedRingPeers: []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				StartAction{
					ExpectedErr: errors.ErrPeerListAlreadyStarted("RoundRobinList"),
				},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedStarted: true,
		},
		{
			msg:               "stop no start",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1"},
			expectedRingPeers: []string{"1"},
			peerListActions: []PeerListAction{
				StopAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
		},
		{
			msg:                "start retain error",
			inputPeerIDs:       []string{"1"},
			errRetainedPeerIDs: []string{"1"},
			retainErr:          errors.ErrNoPeerToSelect("Test!!"),
			expectedCreateErr:  errors.Errors{errors.ErrNoPeerToSelect("Test!!")},
		},
		{
			msg:                "start retain multiple errors",
			inputPeerIDs:       []string{"1", "2", "3"},
			retainedPeerIDs:    []string{"2"},
			errRetainedPeerIDs: []string{"1", "3"},
			retainErr:          errors.ErrNoPeerToSelect("Test!!"),
			expectedCreateErr:  errors.Errors{errors.ErrNoPeerToSelect("Test!!"), errors.ErrNoPeerToSelect("Test!!")},
			expectedRingPeers:  []string{"2"},
		},
		{
			msg:                "start stop release error",
			inputPeerIDs:       []string{"1"},
			retainedPeerIDs:    []string{"1"},
			errReleasedPeerIDs: []string{"1"},
			releaseErr:         errors.ErrAgentHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{
					ExpectedErr: errors.Errors{errors.ErrAgentHasNoReferenceToPeer{}},
				},
			},
			expectedStarted: false,
		},
		{
			msg:                "start stop release multiple errors",
			inputPeerIDs:       []string{"1", "2", "3"},
			retainedPeerIDs:    []string{"1", "2", "3"},
			releasedPeerIDs:    []string{"2"},
			errReleasedPeerIDs: []string{"1", "3"},
			releaseErr:         errors.ErrAgentHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{
					ExpectedErr: errors.Errors{
						errors.ErrAgentHasNoReferenceToPeer{},
						errors.ErrAgentHasNoReferenceToPeer{},
					},
				},
			},
			expectedStarted: false,
		},
		{
			msg:               "choose before start",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1"},
			expectedRingPeers: []string{"1"},
			peerListActions: []PeerListAction{
				ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
				ChooseAction{
					ExpectedErr: errors.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
		},
		{
			msg: "start choose no peers",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					ExpectedErr: errors.ErrNoPeerToSelect("RoundRobinList"),
				},
				ChooseAction{
					ExpectedErr: errors.ErrNoPeerToSelect("RoundRobinList"),
				},
			},
			expectedStarted: true,
		},
		{
			msg:               "start add",
			inputPeerIDs:      []string{"1"},
			retainedPeerIDs:   []string{"1", "2"},
			expectedRingPeers: []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{InputPeerID: "2"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:               "start remove",
			inputPeerIDs:      []string{"1", "2"},
			retainedPeerIDs:   []string{"1", "2"},
			expectedRingPeers: []string{"2"},
			releasedPeerIDs:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{InputPeerID: "1"},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedStarted: true,
		},
		{
			msg:               "start add many and remove many",
			inputPeerIDs:      []string{"1", "2", "3-r", "4-r"},
			retainedPeerIDs:   []string{"1", "2", "3-r", "4-r", "5-a-r", "6-a-r", "7-a", "8-a"},
			releasedPeerIDs:   []string{"3-r", "4-r", "5-a-r", "6-a-r"},
			expectedRingPeers: []string{"1", "2", "7-a", "8-a"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{InputPeerID: "5-a-r"},
				AddAction{InputPeerID: "6-a-r"},
				AddAction{InputPeerID: "7-a"},
				AddAction{InputPeerID: "8-a"},
				RemoveAction{InputPeerID: "5-a-r"},
				RemoveAction{InputPeerID: "6-a-r"},
				RemoveAction{InputPeerID: "3-r"},
				RemoveAction{InputPeerID: "4-r"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "7-a"},
				ChooseAction{ExpectedPeer: "8-a"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                "add retain error",
			inputPeerIDs:       []string{"1", "2"},
			retainedPeerIDs:    []string{"1", "2"},
			expectedRingPeers:  []string{"1", "2"},
			errRetainedPeerIDs: []string{"3"},
			retainErr:          errors.ErrInvalidPeerType{},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{
					InputPeerID: "3",
					ExpectedErr: errors.ErrInvalidPeerType{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:               "add duplicate peer",
			inputPeerIDs:      []string{"1", "2"},
			retainedPeerIDs:   []string{"1", "2", "2"},
			expectedRingPeers: []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{
					InputPeerID: "2",
					ExpectedErr: errors.ErrPeerAlreadyInList("2"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:               "remove peer not in list",
			inputPeerIDs:      []string{"1", "2"},
			retainedPeerIDs:   []string{"1", "2"},
			expectedRingPeers: []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{
					InputPeerID: "3",
					ExpectedErr: errors.ErrPeerNotInList("3"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                "remove release error",
			inputPeerIDs:       []string{"1", "2"},
			retainedPeerIDs:    []string{"1", "2"},
			errReleasedPeerIDs: []string{"2"},
			releaseErr:         errors.ErrAgentHasNoReferenceToPeer{},
			expectedRingPeers:  []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{
					InputPeerID: "2",
					ExpectedErr: errors.ErrAgentHasNoReferenceToPeer{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			agentMockCtrl := gomock.NewController(t)
			defer agentMockCtrl.Finish()
			peerMockCtrl := gomock.NewController(t)
			defer peerMockCtrl.Finish()

			pids := CreatePeerIDs(tt.inputPeerIDs)
			agent := transporttest.NewMockAgent(agentMockCtrl)

			// Healthy Agent Retain/Release
			ExpectPeerRetains(peerMockCtrl, agent, tt.retainedPeerIDs, nil)
			ExpectPeerReleases(agent, tt.releasedPeerIDs, nil)

			// Unhealthy Agent Retain/Release
			ExpectPeerRetains(peerMockCtrl, agent, tt.errRetainedPeerIDs, tt.retainErr)
			ExpectPeerReleases(agent, tt.errReleasedPeerIDs, tt.releaseErr)

			pl, err := New(pids, agent)
			assert.Equal(t, tt.expectedCreateErr, err)

			ApplyPeerListActions(t, pl, tt.peerListActions)

			assert.Equal(t, len(tt.expectedRingPeers), len(pl.pr.peerToNode))
			for _, expectedRingPeer := range tt.expectedRingPeers {
				node, ok := pl.pr.peerToNode[expectedRingPeer]
				assert.True(t, ok, fmt.Sprintf("expected peer: %s was not in peerlist", expectedRingPeer))

				actualPeer := node.getPeer()
				assert.Equal(t, expectedRingPeer, actualPeer.Identifier())
			}

			assert.Equal(t, tt.expectedStarted, pl.started.Load())
		})
	}
}
