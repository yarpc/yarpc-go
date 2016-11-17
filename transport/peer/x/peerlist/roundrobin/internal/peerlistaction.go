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
	"testing"

	"go.uber.org/yarpc/transport"

	"github.com/crossdock/crossdock-go/assert"
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

// ChooseAction is an action for choosing a peer from the peerlist
type ChooseAction struct {
	InputContext context.Context
	InputRequest *transport.Request
	ExpectedPeer string
	ExpectedErr  error
}

// ApplyAndAssert runs "ChoosePeer" on the peerList and validates the peer && error
func (a ChooseAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	peer, err := pl.ChoosePeer(a.InputContext, a.InputRequest)

	if peer != nil {
		assert.Equal(t, a.ExpectedPeer, peer.Identifier())
	} else {
		assert.Equal(t, a.ExpectedPeer, "")
	}
	assert.Equal(t, a.ExpectedErr, err)
}

// AddAction is an action for adding a peer to the peerlist
type AddAction struct {
	InputPeerID string
	ExpectedErr error
}

// ApplyAndAssert runs "Add" on the peerList after casting it to a PeerChangeListener
// and validates the error
func (a AddAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	changeListener := pl.(transport.PeerChangeListener)

	err := changeListener.Add(MockPeerIdentifier(a.InputPeerID))

	assert.Equal(t, a.ExpectedErr, err)
}

// RemoveAction is an action for adding a peer to the peerlist
type RemoveAction struct {
	InputPeerID string
	ExpectedErr error
}

// ApplyAndAssert runs "Remove" on the peerList after casting it to a PeerChangeListener
// and validates the error
func (a RemoveAction) ApplyAndAssert(t *testing.T, pl transport.PeerList) {
	changeListener := pl.(transport.PeerChangeListener)

	err := changeListener.Remove(MockPeerIdentifier(a.InputPeerID))

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
