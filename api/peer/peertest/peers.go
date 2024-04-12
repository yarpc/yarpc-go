// Copyright (c) 2024 Uber Technologies, Inc.
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

package peertest

import (
	"fmt"
	"sync"

	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc/api/peer"
)

// MockPeerIdentifier is a small wrapper around the PeerIdentifier interfaces for a string
// unfortunately gomock + assert.Equal has difficulty seeing between mock objects of the same type.
type MockPeerIdentifier string

// Identifier returns a unique identifier for MockPeerIDs
func (pid MockPeerIdentifier) Identifier() string {
	return string(pid)
}

// NewLightMockPeer returns a new MockPeer
func NewLightMockPeer(pid MockPeerIdentifier, conStatus peer.ConnectionStatus) *LightMockPeer {
	return &LightMockPeer{
		MockPeerIdentifier: pid,
		PeerStatus: peer.Status{
			ConnectionStatus:    conStatus,
			PendingRequestCount: 0,
		},
	}
}

// LightMockPeer is a small simple wrapper around the Peer interface for mocking and changing
// a peer's attributes
// MockPeer is NOT thread safe
type LightMockPeer struct {
	sync.Mutex

	MockPeerIdentifier

	PeerStatus peer.Status
}

// Status returns the Status Object of the MockPeer
func (p *LightMockPeer) Status() peer.Status {
	return p.PeerStatus
}

// StartRequest is run when a Request starts
func (p *LightMockPeer) StartRequest() {
	p.Lock()
	p.PeerStatus.PendingRequestCount++
	p.Unlock()
}

// EndRequest should be run after a MockPeer request has finished
func (p *LightMockPeer) EndRequest() {
	p.Lock()
	p.PeerStatus.PendingRequestCount--
	p.Unlock()
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

// ExpectPeerRetains registers expectations on a MockTransport to generate peers on the RetainPeer function
func ExpectPeerRetains(
	transport *MockTransport,
	availablePeerStrs []string,
	unavailablePeerStrs []string,
) map[string]*LightMockPeer {
	peers := make(map[string]*LightMockPeer, len(availablePeerStrs)+len(unavailablePeerStrs))
	for _, peerStr := range availablePeerStrs {
		p := NewLightMockPeer(MockPeerIdentifier(peerStr), peer.Available)
		transport.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(p, nil)
		peers[p.Identifier()] = p
	}
	for _, peerStr := range unavailablePeerStrs {
		p := NewLightMockPeer(MockPeerIdentifier(peerStr), peer.Unavailable)
		transport.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(p, nil)
		peers[p.Identifier()] = p
	}
	return peers
}

// ExpectPeerRetainsWithError registers expectations on a MockTransport return errors
func ExpectPeerRetainsWithError(
	transport *MockTransport,
	peerStrs []string,
	err error, // Will be returned from the MockTransport on the Retains of these Peers
) {
	for _, peerStr := range peerStrs {
		transport.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(nil, err)
	}
}

// ExpectPeerReleases registers expectations on a MockTransport to release peers through the ReleasePeer function
func ExpectPeerReleases(
	transport *MockTransport,
	peerStrs []string,
	err error,
) {
	for _, peerStr := range peerStrs {
		transport.EXPECT().ReleasePeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(err)
	}
}
