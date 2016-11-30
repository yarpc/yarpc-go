package single

import (
	"testing"

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

		// Actions that will be applied on the PeerChooser
		actions []PeerListAction

		// Expected Peer to be stored in the Single Chooser
		expectedPeer string
	}
	tests := []testStruct{
		{
			msg:            "setup",
			inputPeerID:    "1",
			retainedPeerID: "1",
			expectedPeer:   "1",
		},
		{
			msg:            "choose",
			inputPeerID:    "1",
			retainedPeerID: "1",
			actions: []PeerListAction{
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedPeer: "1",
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

			s := New(MockPeerIdentifier(tt.inputPeerID), agent)

			ApplyPeerListActions(t, s, tt.actions, ListActionDeps{})

			if tt.expectedPeer != "" {
				assert.Equal(t, tt.expectedPeer, s.p.Identifier())
			} else {
				assert.Nil(t, s.p)
			}
		})
	}
}
