package http

import (
	"testing"
	"time"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/peertest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

type peerExpectation struct {
	identifier  hostport.PeerIdentifier
	subscribers []*peertest.MockSubscriber
}

// createPeerExpectations creates a slice of peerExpectation structs for the
// peers that are expected to be contained in the agent.  It will also add a
// number of expected subscribers for each Peer
func createPeerExpectations(
	mockCtrl *gomock.Controller,
	hostports []string,
	subscribers int,
) []peerExpectation {
	expectations := make([]peerExpectation, 0, len(hostports))
	for _, hp := range hostports {
		hpid := hostport.PeerIdentifier(hp)
		subs := make([]*peertest.MockSubscriber, 0, subscribers)
		for i := 0; i < subscribers; i++ {
			subs = append(subs, peertest.NewMockSubscriber(mockCtrl))
		}
		expectations = append(expectations, peerExpectation{
			identifier:  hpid,
			subscribers: subs,
		})
	}
	return expectations
}

// peerIdentifierMatcher allows us to compare peerIdentifiers with Peers through gomock
type peerIdentifierMatcher hostport.PeerIdentifier

// Matches returns whether x is a match.
func (pim peerIdentifierMatcher) Matches(x interface{}) bool {
	res, ok := x.(peer.Identifier)
	if !ok {
		return false
	}

	return res.Identifier() == hostport.PeerIdentifier(pim).Identifier()
}

// String describes what the matcher matches.
func (pim peerIdentifierMatcher) String() string {
	return hostport.PeerIdentifier(pim).Identifier()
}

func TestAgent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		msg           string
		agent         *Agent
		appliedFunc   func(*Agent) error
		expectedPeers []peerExpectation
		expectedErr   error
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "one retain"
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				1,
			)
			s.appliedFunc = func(a *Agent) error {
				_, err := a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[0])
				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "one retain one release"
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.PeerIdentifier("localhost:1234")

				sub := peertest.NewMockSubscriber(mockCtrl)

				if _, err := a.RetainPeer(pid, sub); err != nil {
					return err
				}
				return a.ReleasePeer(pid, sub)
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "one retain one release using peer"
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.PeerIdentifier("localhost:1234")

				sub := peertest.NewMockSubscriber(mockCtrl)

				p, err := a.RetainPeer(pid, sub)
				if err != nil {
					return err
				}
				err = a.ReleasePeer(p, sub)

				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "three retains"
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				3,
			)
			s.appliedFunc = func(a *Agent) error {
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[0])
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[1])
				_, err := a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[2])
				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "three retains, one release"
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				2,
			)
			s.appliedFunc = func(a *Agent) error {
				unSub := peertest.NewMockSubscriber(mockCtrl)
				a.RetainPeer(s.expectedPeers[0].identifier, unSub)
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[0])
				a.ReleasePeer(s.expectedPeers[0].identifier, unSub)
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[1])

				return nil
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "three retains, three release"
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.PeerIdentifier("localhost:123")

				sub := peertest.NewMockSubscriber(mockCtrl)
				sub2 := peertest.NewMockSubscriber(mockCtrl)
				sub3 := peertest.NewMockSubscriber(mockCtrl)

				a.RetainPeer(pid, sub)
				a.RetainPeer(pid, sub2)
				a.ReleasePeer(pid, sub)
				a.RetainPeer(pid, sub3)
				a.ReleasePeer(pid, sub2)
				a.ReleasePeer(pid, sub3)

				return nil
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "no retains, one release"
			s.agent = NewAgent()
			s.expectedPeers = []peerExpectation{}

			pid := hostport.PeerIdentifier("localhost:1234")
			s.expectedErr = peer.ErrAgentHasNoReferenceToPeer{
				Agent:          s.agent,
				PeerIdentifier: pid,
			}

			s.appliedFunc = func(a *Agent) error {
				return a.ReleasePeer(pid, peertest.NewMockSubscriber(mockCtrl))
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "retain with invalid identifier"
			s.expectedPeers = []peerExpectation{}

			pid := peertest.NewMockIdentifier(mockCtrl)
			s.expectedErr = peer.ErrInvalidPeerType{
				ExpectedType:   "hostport.PeerIdentifier",
				PeerIdentifier: pid,
			}

			s.appliedFunc = func(a *Agent) error {
				_, err := a.RetainPeer(pid, peertest.NewMockSubscriber(mockCtrl))
				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "one retains, one release (from different subscriber)"
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				1,
			)

			invalidSub := peertest.NewMockSubscriber(mockCtrl)
			s.expectedErr = peer.ErrPeerHasNoReferenceToSubscriber{
				PeerIdentifier: s.expectedPeers[0].identifier,
				PeerSubscriber: invalidSub,
			}

			s.appliedFunc = func(a *Agent) error {
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[0])
				return a.ReleasePeer(s.expectedPeers[0].identifier, invalidSub)
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "multi peer retain/release"
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234", "localhost:1111", "localhost:2222"},
				2,
			)

			s.appliedFunc = func(a *Agent) error {
				expP1 := s.expectedPeers[0]
				expP2 := s.expectedPeers[1]
				expP3 := s.expectedPeers[2]
				rndP1 := hostport.PeerIdentifier("localhost:9888")
				rndP2 := hostport.PeerIdentifier("localhost:9883")
				rndSub1 := peertest.NewMockSubscriber(mockCtrl)
				rndSub2 := peertest.NewMockSubscriber(mockCtrl)
				rndSub3 := peertest.NewMockSubscriber(mockCtrl)

				// exp1: Defer a bunch of Releases
				a.RetainPeer(expP1.identifier, rndSub1)
				defer a.ReleasePeer(expP1.identifier, rndSub1)
				a.RetainPeer(expP1.identifier, expP1.subscribers[0])
				a.RetainPeer(expP1.identifier, rndSub2)
				defer a.ReleasePeer(expP1.identifier, rndSub2)
				a.RetainPeer(expP1.identifier, expP1.subscribers[1])

				// exp2: Retain a subscriber, release it, then retain it again
				a.RetainPeer(expP2.identifier, expP2.subscribers[0])
				a.RetainPeer(expP2.identifier, expP2.subscribers[1])
				a.ReleasePeer(expP2.identifier, expP2.subscribers[0])
				a.ReleasePeer(expP2.identifier, expP2.subscribers[1])
				a.RetainPeer(expP2.identifier, expP2.subscribers[0])
				a.RetainPeer(expP2.identifier, expP2.subscribers[1])

				// exp3: Retain release a Peer
				a.RetainPeer(expP3.identifier, rndSub3)
				a.ReleasePeer(expP3.identifier, rndSub3)
				a.RetainPeer(expP3.identifier, expP3.subscribers[0])
				a.RetainPeer(expP3.identifier, expP3.subscribers[1])

				// rnd1: retain/release on random sub
				a.RetainPeer(rndP1, rndSub1)
				a.ReleasePeer(rndP1, rndSub1)

				// rnd2: retain/release on already used subscriber
				a.RetainPeer(rndP2, expP1.subscribers[0])
				a.ReleasePeer(rndP2, expP1.subscribers[0])

				return nil
			}
			return
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			agent := tt.agent
			if agent == nil {
				agent = NewAgent()
			}

			err := tt.appliedFunc(agent)

			assert.Equal(t, tt.expectedErr, err)
			assert.Len(t, agent.peers, len(tt.expectedPeers))
			for _, expectedPeerNode := range tt.expectedPeers {
				p, ok := agent.peers[expectedPeerNode.identifier.Identifier()]
				assert.True(t, ok)

				assert.Equal(t, expectedPeerNode.identifier.Identifier(), p.Identifier())

				assert.Equal(t, p.NumSubscribers(), len(expectedPeerNode.subscribers))
			}
		})
	}
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
