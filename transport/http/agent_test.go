package http

import (
	"testing"
	"time"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	. "go.uber.org/yarpc/peer/peertest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

type peerExpectation struct {
	id          string
	subscribers []string
}

func createPeerIdentifierMap(ids []string) map[string]peer.Identifier {
	pids := make(map[string]peer.Identifier, len(ids))
	for _, id := range ids {
		pids[id] = hostport.PeerIdentifier(id)
	}
	return pids
}

func TestAgent(t *testing.T) {
	type testStruct struct {
		msg string

		// identifiers defines all the Identifiers that will be used in
		// the actions up from so they can be generated and passed as deps
		identifiers []string

		// subscriberDefs defines all the Subscribers that will be used in
		// the actions up from so they can be generated and passed as deps
		subscriberDefs []SubscriberDefinition

		// actions are the actions that will be applied against the agent
		actions []AgentAction

		// expectedPeers are a list of peers (and those peer's subscribers)
		// that are expected on the agent after the actions
		expectedPeers []peerExpectation
	}
	tests := []testStruct{
		{
			msg:         "one retain",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "one retain one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
			},
		},
		{
			msg:         "three retains",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2", "s3"}},
			},
		},
		{
			msg:         "three retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2r"},
				{ID: "s3"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2r", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2r"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s3"}},
			},
		},
		{
			msg:         "three retains, three release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s3"},
			},
		},
		{
			msg:         "no retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []AgentAction{
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s1",
					ExpectedErrType:   peer.ErrAgentHasNoReferenceToPeer{},
				},
			},
		},
		{
			msg:         "one retains, one release (from different subscriber)",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
			},
			actions: []AgentAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s2",
					ExpectedErrType:   peer.ErrPeerHasNoReferenceToSubscriber{},
				},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "multi peer retain/release",
			identifiers: []string{"i1", "i2", "i3", "i4r", "i5r"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
				{ID: "s4"},
				{ID: "s5rnd"},
				{ID: "s6rnd"},
				{ID: "s7rnd"},
			},
			actions: []AgentAction{
				// Retains/Releases of i1 (Retain/Release the random peers at the end)
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd"},

				// Retains/Releases of i2 (Retain then Release then Retain again)
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s3"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},

				// Retains/Releases of i3 (Retain/Release unrelated sub, then retain two)
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd", ExpectedPeerID: "i3"},
				ReleaseAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s3", ExpectedPeerID: "i3"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s4", ExpectedPeerID: "i3"},

				// Retain/Release i4r on random subscriber
				RetainAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd", ExpectedPeerID: "i4r"},
				ReleaseAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd"},

				// Retain/Release i5r on already used subscriber
				RetainAction{InputIdentifierID: "i5r", InputSubscriberID: "s3", ExpectedPeerID: "i5r"},
				ReleaseAction{InputIdentifierID: "i5r", InputSubscriberID: "s3"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2"}},
				{id: "i2", subscribers: []string{"s2", "s3"}},
				{id: "i3", subscribers: []string{"s3", "s4"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			agent := NewAgent()

			deps := AgentDeps{
				PeerIdentifiers: createPeerIdentifierMap(tt.identifiers),
				Subscribers:     CreateSubscriberMap(mockCtrl, tt.subscriberDefs),
			}
			ApplyAgentActions(t, agent, tt.actions, deps)

			assert.Len(t, agent.peers, len(tt.expectedPeers))
			for _, expectedPeerNode := range tt.expectedPeers {
				p, ok := agent.peers[expectedPeerNode.id]
				assert.True(t, ok)

				if assert.NotNil(t, p) {
					assert.Equal(t, expectedPeerNode.id, p.Identifier())

					// We can't look at the hostport subscribers directly so we'll
					// attempt to remove subscribers and be sure that it doesn't error
					assert.Len(t, expectedPeerNode.subscribers, p.NumSubscribers())
					for _, sub := range expectedPeerNode.subscribers {
						err := p.RemoveSubscriber(deps.Subscribers[sub])
						assert.NoError(t, err, "peer %s did not have reference to subscriber %s", p.Identifier(), sub)
					}
				}
			}
		})
	}
}

func TestAgentRetainWithInvalidPeerIdentifierType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	agent := NewAgent()
	pid := NewMockIdentifier(mockCtrl)

	expectedErr := peer.ErrInvalidPeerType{
		ExpectedType:   "hostport.PeerIdentifier",
		PeerIdentifier: pid,
	}

	_, err := agent.RetainPeer(pid, NewMockSubscriber(mockCtrl))

	assert.Equal(t, expectedErr, err, "did not return error on invalid peer identifier")
}

func TestAgentClient(t *testing.T) {
	agent := NewAgent()

	assert.NotNil(t, agent.client)
}

func TestAgentClientWithKeepAlive(t *testing.T) {
	// Unfortunately the KeepAlive is obfuscated in the client, so we can't really
	// assert this worked
	agent := NewAgent(KeepAlive(time.Second))

	assert.NotNil(t, agent.client)
}
