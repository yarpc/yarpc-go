package internal

import (
	"fmt"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
)

// MockPeerIdentifier is a small wrapper around the PeerIdentifier interfaces for a string
// unfortunately gomock + assert.Equal has difficulty seeing between mock objects of the same type.
type MockPeerIdentifier string

// Identifier returns a unique identifier for MockPeerIDs
func (pid MockPeerIdentifier) Identifier() string {
	return string(pid)
}

// PeerIdentifierMatcher is used to match a Peer/PeerIdentifier by comparing
// The peer's .Identifier function with the Matcher string
type PeerIdentifierMatcher string

// Matches returns true of got is equivalent to the PeerIdentifier Matching string
func (pim PeerIdentifierMatcher) Matches(got interface{}) bool {
	gotPID, ok := got.(transport.PeerIdentifier)
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
func CreatePeerIDs(
	peerIDStrs []string,
) []transport.PeerIdentifier {
	pids := make([]transport.PeerIdentifier, 0, len(peerIDStrs))
	for _, id := range peerIDStrs {
		pids = append(pids, MockPeerIdentifier(id))
	}
	return pids
}

// ExpectPeerRetains registers expectations on a MockAgent to generate peers on the RetainPeer function
func ExpectPeerRetains(
	mockCtrl *gomock.Controller,
	agent *transporttest.MockAgent,
	peerStrs []string,
	err error,
) []transport.Peer {
	peers := make([]transport.Peer, 0, len(peerStrs))
	for _, peerStr := range peerStrs {
		peer := transporttest.NewMockPeer(mockCtrl)
		peer.EXPECT().Identifier().Return(peerStr).AnyTimes()

		agent.EXPECT().RetainPeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(peer, err)

		peers = append(peers, peer)
	}
	return peers
}

// ExpectPeerReleases registers expectations on a MockAgent to release peers through the ReleasePeer function
func ExpectPeerReleases(
	agent *transporttest.MockAgent,
	peerStrs []string,
	err error,
) {
	for _, peerStr := range peerStrs {
		agent.EXPECT().ReleasePeer(PeerIdentifierMatcher(peerStr), gomock.Any()).Return(err)
	}
}
