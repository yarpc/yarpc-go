package chooser

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc/peer"
	. "go.uber.org/yarpc/peer/peertest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

func TestSingleChooser(t *testing.T) {
	type testStruct struct {
		msg string

		// PeerID that will be input into the PeerChooser
		inputPeerID string

		// PeerID that will be returned from the agent's OnRetain
		retainedPeerID string

		// Error that will be returned from the agent's OnRetain
		retainedErr error

		// PeerID that will be released from the agent
		releasedPeerID string

		// Error that will be returned from the agent's OnRelease
		releasedErr error

		// Actions that will be applied on the PeerChooser
		actions []PeerListAction

		// Expected PeerID to be stored in the Single Chooser
		expectedPeerID string

		// Expected Peer to be stored in the Single Chooser
		expectedPeer string

		// Expected state of the started flag in the Chooser
		expectedStarted bool
	}
	tests := []testStruct{
		{
			msg:             "setup",
			inputPeerID:     "1",
			expectedPeerID:  "1",
			expectedStarted: false,
		},
		{
			msg:         "stop before start",
			inputPeerID: "1",
			actions: []PeerListAction{
				StopAction{ExpectedErr: peer.ErrPeerListNotStarted("single")},
			},
			expectedPeerID:  "1",
			expectedStarted: false,
		},
		{
			msg:         "choose before start",
			inputPeerID: "1",
			actions: []PeerListAction{
				ChooseAction{ExpectedErr: peer.ErrPeerListNotStarted("single")},
			},
			expectedPeerID:  "1",
			expectedStarted: false,
		},
		{
			msg:            "start and choose",
			inputPeerID:    "1",
			retainedPeerID: "1",
			actions: []PeerListAction{
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedPeerID:  "1",
			expectedPeer:    "1",
			expectedStarted: true,
		},
		{
			msg:            "start with agent error",
			inputPeerID:    "1",
			retainedPeerID: "1",
			retainedErr:    fmt.Errorf("test error"),
			actions: []PeerListAction{
				StartAction{ExpectedErr: fmt.Errorf("test error")},
			},
			expectedPeerID:  "1",
			expectedStarted: false,
		},
		{
			msg:            "start twice",
			inputPeerID:    "1",
			retainedPeerID: "1",
			actions: []PeerListAction{
				StartAction{},
				StartAction{ExpectedErr: peer.ErrPeerListAlreadyStarted("single")},
			},
			expectedPeer:    "1",
			expectedPeerID:  "1",
			expectedStarted: true,
		},
		{
			msg:            "start stop",
			inputPeerID:    "1",
			retainedPeerID: "1",
			releasedPeerID: "1",
			actions: []PeerListAction{
				StartAction{},
				StopAction{},
			},
			expectedPeerID:  "1",
			expectedStarted: false,
		},
		{
			msg:            "start stop release failure",
			inputPeerID:    "1",
			retainedPeerID: "1",
			releasedPeerID: "1",
			releasedErr:    peer.ErrAgentHasNoReferenceToPeer{},
			actions: []PeerListAction{
				StartAction{},
				StopAction{ExpectedErr: peer.ErrAgentHasNoReferenceToPeer{}},
			},
			expectedPeerID:  "1",
			expectedPeer:    "1",
			expectedStarted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			agent := NewMockAgent(mockCtrl)

			if tt.retainedPeerID != "" {
				if tt.retainedErr != nil {
					ExpectPeerRetainsWithError(agent, []string{tt.retainedPeerID}, tt.retainedErr)
				} else {
					ExpectPeerRetains(agent, []string{tt.retainedPeerID}, []string{})
				}
			}
			if tt.releasedPeerID != "" {
				ExpectPeerReleases(agent, []string{tt.releasedPeerID}, tt.releasedErr)
			}

			pl := NewSingle(MockPeerIdentifier(tt.inputPeerID), agent).(*single)

			ApplyPeerListActions(t, pl, tt.actions, ListActionDeps{})

			assert.Equal(t, agent, pl.agent)
			assert.Equal(t, tt.expectedPeerID, pl.initialPeerID.Identifier())
			if tt.expectedPeer != "" {
				assert.Equal(t, tt.expectedPeer, pl.p.Identifier())
			} else {
				assert.Nil(t, pl.p)
			}
			assert.Equal(t, tt.expectedStarted, pl.started)
		})
	}
}
