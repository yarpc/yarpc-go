package transporttest

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossdock/crossdock-go/assert"
	"go.uber.org/yarpc/transport"
)

// PeerListAction is an interface for defining actions that can be applied
// to a PeerList
type PeerListAction interface {
	// ApplyAndAssert runs a function on the PeerList and asserts the result
	ApplyAndAssert(*testing.T, transport.PeerList)
}

// StartAction is an action for testing PeerList.Start
type StartAction struct {
	ExpectedErr error
}

// ApplyAndAssert runs "Start" on the peerList and validates the error
func (a StartAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	err := pl.Start()

	assert.Equal(t, a.ExpectedErr, err)
}

// StopAction is an action for testing PeerList.Stop
type StopAction struct {
	ExpectedErr error
}

// ApplyAndAssert runs "Stop" on the peerList and validates the error
func (a StopAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	err := pl.Stop()

	assert.Equal(t, a.ExpectedErr, err)
}

// ChooseAction is an action for
type ChooseAction struct {
	InputContext context.Context
	InputRequest *transport.Request
	ExpectedPeer transport.Peer
	ExpectedErr  error
}

// ApplyAndAssert runs "ChoosePeer" on the peerList and validates the peer && error
func (a ChooseAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	peer, err := pl.ChoosePeer(a.InputContext, a.InputRequest)

	assert.Equal(t, a.ExpectedPeer, peer)
	assert.True(
		t,
		a.ExpectedPeer == peer,
		fmt.Sprintf("%v was not the same instance as %v", peer, a.ExpectedPeer),
	)
	assert.Equal(t, a.ExpectedErr, err)
}

// ApplyPeerListActions runs all the PeerListActions on the PeerList
func ApplyPeerListActions(t *testing.T, pl transport.PeerList, actions []PeerListAction) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.ApplyAndAssert(t, pl)
		})
	}
}
