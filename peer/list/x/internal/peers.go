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
	"fmt"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/peertest"

	"github.com/golang/mock/gomock"
)

// MockPeerIdentifier is a small wrapper around the PeerIdentifier interfaces for a string
// unfortunately gomock + assert.Equal has difficulty seeing between mock objects of the same type.
type MockPeerIdentifier string

// Identifier returns a unique identifier for MockPeerIDs
func (pid MockPeerIdentifier) Identifier() string {
	return string(pid)
}

// NewMockPeer returns a new MockPeer
func NewMockPeer(pid MockPeerIdentifier, conStatus peer.ConnectionStatus) *MockPeer {
	return &MockPeer{
		MockPeerIdentifier: pid,
		PeerStatus: peer.Status{
			ConnectionStatus:    conStatus,
			PendingRequestCount: 0,
		},
	}
}

// MockPeer is a small simple wrapper around the Peer interface for mocking and changing
// a peer's attributes
// MockPeer is NOT thread safe
type MockPeer struct {
	MockPeerIdentifier

	PeerStatus peer.Status
}

// Status returns the Status Object of the MockPeer
func (p *MockPeer) Status() peer.Status {
	return p.PeerStatus
}

// StartRequest is run when a Request starts
func (p *MockPeer) StartRequest() func() {
	p.PeerStatus.PendingRequestCount++
	return p.endRequest
}

// endRequest should be run after a MockPeer request has finished
func (p *MockPeer) endRequest() {
	p.PeerStatus.PendingRequestCount--
}

// PeerIdentifierMatcher is used to match a Peer/PeerIdentifier by comparing
// The peer's .Identifier function with the Matcher string
type PeerIdentifierMatcher string

// Matches returns true of got is equivalent to the PeerIdentifier Matching string
func (pim PeerIdentifierMatcher) Matches(got interface{}) bool {
	gotPID, ok := got.(peer.Identifier)
	if !ok {
		return false
	}
	return gotPID.Identifier() == string(pim)
}

// String returns a description of the matcher
func (pim PeerIdentifierMatcher) String() string {
	return fmt.Sprintf("PeerIdentifierMatcher(%s)", string(pim))
}

// CreatePeerIDs takes a slice of peerID strings and returns a slice of PeerIdentifiers
func CreatePeerIDs(peerIDStrs []string) []peer.Identifier {
	pids := make([]peer.Identifier, 0, len(peerIDStrs))
	for _, id := range peerIDStrs {
		pids = append(pids, MockPeerIdentifier(id))
	}
	return pids
}

// ExpectPeerRetains registers expectations on a MockAgent to generate peers on the RetainPeer function
func ExpectPeerRetains(
	agent *peertest.MockAgent,
	availablePeerStrs []string,
	unavailablePeerStrs []string,
) map[string]*MockPeer {
	peers := make(map[string]*MockPeer, len(availablePeerStrs)+len(unavailablePeerStrs))
	for _, peerStr := range availablePeerStrs {
		peer := NewMockPeer(MockPeerIdentifier(peerStr), peer.Available)
		agent.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(peer, nil)
		peers[peer.Identifier()] = peer
	}
	for _, peerStr := range unavailablePeerStrs {
		peer := NewMockPeer(MockPeerIdentifier(peerStr), peer.Unavailable)
		agent.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(peer, nil)
		peers[peer.Identifier()] = peer
	}
	return peers
}

// ExpectPeerRetainsWithError registers expectations on a MockAgent return errors
func ExpectPeerRetainsWithError(
	agent *peertest.MockAgent,
	peerStrs []string,
	err error, // Will be returned from the MockAgent on the Retains of these Peers
) {
	for _, peerStr := range peerStrs {
		agent.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(nil, err)
	}
}

// ExpectPeerReleases registers expectations on a MockAgent to release peers through the ReleasePeer function
func ExpectPeerReleases(
	agent *peertest.MockAgent,
	peerStrs []string,
	err error,
) {
	for _, peerStr := range peerStrs {
		agent.EXPECT().ReleasePeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(err)
	}
}
