// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
)

// Dependencies are passed through PeerListActions' Apply methods in order
// to allow the PeerListAction to modify state other than just the PeerList
type Dependencies struct {
	Peers map[string]*MockPeer
}

// PeerListAction defines actions that can be applied to a PeerList
type PeerListAction interface {
	// Apply runs a function on the PeerList and asserts the result
	Apply(*testing.T, transport.PeerList, Dependencies)
}

// StartAction is an action for testing PeerList.Start
type StartAction struct {
	ExpectedErr error
}

// Apply runs "Start" on the peerList and validates the error
func (a StartAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	err := pl.Start()
	assert.Equal(t, a.ExpectedErr, err)
}

// StopAction is an action for testing PeerList.Stop
type StopAction struct {
	ExpectedErr error
}

// Apply runs "Stop" on the peerList and validates the error
func (a StopAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	err := pl.Stop()
	assert.Equal(t, a.ExpectedErr, err)
}

// ChooseMultiAction will run ChoosePeer multiple times on the PeerList
// It will assert if there are ANY failures
type ChooseMultiAction struct {
	ExpectedPeers []string
}

// Apply runs "ChoosePeer" on the peerList for every ExpectedPeer
func (a ChooseMultiAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	for _, expectedPeer := range a.ExpectedPeers {
		action := ChooseAction{
			ExpectedPeer: expectedPeer,
		}
		action.Apply(t, pl, deps)
	}
}

// ChooseAction is an action for choosing a peer from the peerlist
type ChooseAction struct {
	InputContext        context.Context
	InputContextTimeout time.Duration
	InputRequest        *transport.Request
	ExpectedPeer        string
	ExpectedErr         error
}

// Apply runs "ChoosePeer" on the peerList and validates the peer && error
func (a ChooseAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	ctx := a.InputContext
	if ctx == nil {
		ctx = context.Background()
	}
	if a.InputContextTimeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.InputContextTimeout)
		defer cancel()
	}

	peer, err := pl.ChoosePeer(ctx, a.InputRequest)

	if a.ExpectedErr != nil {
		// Note that we're not verifying anything about ExpectedPeer here because
		// it being non-empty means that the test itself was invalid. If anything,
		// that should cause a panic, not a test failure. But that validation can
		// be done before you start asserting expectations.
		assert.Nil(t, peer)
		assert.Equal(t, a.ExpectedErr, err)
		return
	}

	if assert.NoError(t, err) && assert.NotNil(t, peer) {
		assert.Equal(t, a.ExpectedPeer, peer.Identifier())
	}
}

// AddAction is an action for adding a peer to the peerlist
type AddAction struct {
	InputPeerID string
	ExpectedErr error
}

// Apply runs "Add" on the peerList after casting it to a PeerChangeListener
// and validates the error
func (a AddAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	changeListener := pl.(transport.PeerChangeListener)

	err := changeListener.Add(MockPeerIdentifier(a.InputPeerID))
	assert.Equal(t, a.ExpectedErr, err)
}

// RemoveAction is an action for adding a peer to the peerlist
type RemoveAction struct {
	InputPeerID string
	ExpectedErr error
}

// Apply runs "Remove" on the peerList after casting it to a PeerChangeListener
// and validates the error
func (a RemoveAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	changeListener := pl.(transport.PeerChangeListener)

	err := changeListener.Remove(MockPeerIdentifier(a.InputPeerID))
	assert.Equal(t, a.ExpectedErr, err)
}

// ConcurrentAction will run a series of actions in parallel
type ConcurrentAction struct {
	Actions []PeerListAction
	Wait    time.Duration
}

// Apply runs all the ConcurrentAction's actions in goroutines with a delay of MSWait
// between each action and uses a WaitGroup to make sure all goroutines finish before continuing
func (a ConcurrentAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	var wg sync.WaitGroup

	wg.Add(len(a.Actions))
	for _, action := range a.Actions {
		go func() {
			defer wg.Done()
			action.Apply(t, pl, deps)
		}()

		time.Sleep(a.Wait)
	}

	wg.Wait()
}

// NotifyStatusChangeAction will run the NotifyStatusChange function on a PeerList
// with a specified Peer after changing the peer's ConnectionStatus
type NotifyStatusChangeAction struct {
	// PeerID is a unique identifier to the Peer we want use in the notification
	PeerID string

	// NewConnectionStatus is the new ConnectionStatus of the Peer
	NewConnectionStatus transport.PeerConnectionStatus
}

// Apply will run the NotifyStatusChanged function on the PeerList with the provided Peer
func (a NotifyStatusChangeAction) Apply(t *testing.T, pl transport.PeerList, deps Dependencies) {
	deps.Peers[a.PeerID].StatusObj.ConnectionStatus = a.NewConnectionStatus

	plSub := pl.(transport.PeerSubscriber)

	plSub.NotifyStatusChanged(deps.Peers[a.PeerID])
}

// ApplyPeerListActions runs all the PeerListActions on the PeerList
func ApplyPeerListActions(t *testing.T, pl transport.PeerList, actions []PeerListAction, deps Dependencies) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, pl, deps)
		})
	}
}
