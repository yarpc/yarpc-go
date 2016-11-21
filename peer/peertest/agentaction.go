package peertest

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc/peer"

	"github.com/stretchr/testify/assert"
)

// AgentDeps are passed through all the AgentActions in order to pass certain
// state in between Actions
type AgentDeps struct {
	PeerIdentifiers map[string]peer.Identifier
	Subscribers     map[string]peer.Subscriber
}

// AgentAction defines actions that can be applied to an Agent
type AgentAction interface {
	// Apply runs a function on the Agent and asserts the result
	Apply(*testing.T, peer.Agent, AgentDeps)
}

// RetainAction will execute the RetainPeer method on the Agent
type RetainAction struct {
	InputIdentifierID string
	InputSubscriberID string
	ExpectedErr       error
	ExpectedPeerID    string
}

// Apply will execute the RetainPeer method on the Agent
func (a RetainAction) Apply(t *testing.T, agent peer.Agent, deps AgentDeps) {
	peerID := deps.PeerIdentifiers[a.InputIdentifierID]
	sub := deps.Subscribers[a.InputSubscriberID]

	p, err := agent.RetainPeer(peerID, sub)

	if a.ExpectedErr != nil {
		assert.Equal(t, a.ExpectedErr, err)
		assert.Nil(t, p)
		return
	}

	if assert.NoError(t, err) && assert.NotNil(t, p) {
		assert.Equal(t, a.ExpectedPeerID, p.Identifier())
	}
}

// ReleaseAction will execute the ReleasePeer method on the Agent
type ReleaseAction struct {
	InputIdentifierID string
	InputSubscriberID string
	ExpectedErrType   error
}

// Apply will execute the ReleasePeer method on the Agent
func (a ReleaseAction) Apply(t *testing.T, agent peer.Agent, deps AgentDeps) {
	peerID := deps.PeerIdentifiers[a.InputIdentifierID]
	sub := deps.Subscribers[a.InputSubscriberID]

	err := agent.ReleasePeer(peerID, sub)

	if a.ExpectedErrType != nil && assert.Error(t, err) {
		assert.IsType(t, a.ExpectedErrType, err)
	} else {
		assert.Nil(t, err)
	}
}

// ApplyAgentActions runs all the AgentActions on the peer Agent
func ApplyAgentActions(t *testing.T, agent peer.Agent, actions []AgentAction, d AgentDeps) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, agent, d)
		})
	}
}
