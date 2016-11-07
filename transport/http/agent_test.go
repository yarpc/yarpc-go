package http

import (
	"testing"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/peer/hostport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

type peerExpectation struct {
	identifier  hostport.PeerIdentifier
	subscribers []*transporttest.MockPeerSubscriber
}

func createPeerExpectations(
	mockCtrl *gomock.Controller,
	hostports []string,
	subscribers int,
) []peerExpectation {
	expectations := make([]peerExpectation, 0, len(hostports))
	for _, hp := range hostports {
		hpid := hostport.NewPeerIdentifier(hp)
		subs := make([]*transporttest.MockPeerSubscriber, 0, subscribers)
		for i := 0; i < subscribers; i++ {
			subs = append(subs, transporttest.NewMockPeerSubscriber(mockCtrl))
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
	res, ok := x.(transport.PeerIdentifier)
	if !ok {
		return false
	}

	return res.Identifier() == hostport.PeerIdentifier(pim).Identifier()
}

// String describes what the matcher matches.
func (pim peerIdentifierMatcher) String() string {
	return string(hostport.PeerIdentifier(pim).Identifier())
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
			s.agent = NewDefaultAgent()
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
			s.msg = "one retain on release"
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.NewPeerIdentifier("localhost:1234")

				sub := transporttest.NewMockPeerSubscriber(mockCtrl)

				_, err := a.RetainPeer(pid, sub)
				if err != nil {
					return err
				}
				err = a.ReleasePeer(pid, sub)

				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "one retain on release using peer"
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.NewPeerIdentifier("localhost:1234")

				sub := transporttest.NewMockPeerSubscriber(mockCtrl)

				peer, err := a.RetainPeer(pid, sub)
				if err != nil {
					return err
				}
				err = a.ReleasePeer(peer, sub)

				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "three retains"
			s.agent = NewDefaultAgent()
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
			s.agent = NewDefaultAgent()
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				2,
			)
			s.appliedFunc = func(a *Agent) error {
				unSub := transporttest.NewMockPeerSubscriber(mockCtrl)
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
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}
			s.appliedFunc = func(a *Agent) error {
				pid := hostport.NewPeerIdentifier("localhost:123")

				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub2 := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub3 := transporttest.NewMockPeerSubscriber(mockCtrl)

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
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}

			pid := hostport.NewPeerIdentifier("localhost:1234")
			s.expectedErr = errors.ErrAgentHasNoReferenceToPeer{
				Agent:          s.agent,
				PeerIdentifier: pid,
			}

			s.appliedFunc = func(a *Agent) error {
				return a.ReleasePeer(pid, transporttest.NewMockPeerSubscriber(mockCtrl))
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "retain with invalid identifier"
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}

			pid := transporttest.NewMockPeerIdentifier(mockCtrl)
			s.expectedErr = errors.ErrInvalidPeerType{
				ExpectedType:   "hostport.PeerIdentifier",
				PeerIdentifier: pid,
			}

			s.appliedFunc = func(a *Agent) error {
				_, err := a.RetainPeer(pid, transporttest.NewMockPeerSubscriber(mockCtrl))
				return err
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "one retains, one release (from different subscriber)"
			s.agent = NewDefaultAgent()
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				1,
			)

			invalidSub := transporttest.NewMockPeerSubscriber(mockCtrl)
			s.expectedErr = errors.ErrPeerHasNoReferenceToSubscriber{
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
			s.agent = NewDefaultAgent()
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234", "localhost:1111", "localhost:2222"},
				2,
			)

			s.appliedFunc = func(a *Agent) error {
				expP1 := s.expectedPeers[0]
				expP2 := s.expectedPeers[1]
				expP3 := s.expectedPeers[2]
				rndP1 := hostport.NewPeerIdentifier("localhost:9888")
				rndP2 := hostport.NewPeerIdentifier("localhost:9883")
				rndSub1 := transporttest.NewMockPeerSubscriber(mockCtrl)
				rndSub2 := transporttest.NewMockPeerSubscriber(mockCtrl)
				rndSub3 := transporttest.NewMockPeerSubscriber(mockCtrl)

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
		func() (s testStruct) {
			s.msg = "notification verification"
			s.agent = NewDefaultAgent()
			s.expectedPeers = createPeerExpectations(
				mockCtrl,
				[]string{"localhost:1234"},
				2,
			)

			s.expectedPeers[0].subscribers[0].EXPECT().NotifyStatusChanged(
				peerIdentifierMatcher(s.expectedPeers[0].identifier),
			).Times(3)
			s.expectedPeers[0].subscribers[1].EXPECT().NotifyStatusChanged(
				peerIdentifierMatcher(s.expectedPeers[0].identifier),
			).Times(3)

			s.appliedFunc = func(a *Agent) error {
				peer, _ := a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[0])
				a.RetainPeer(s.expectedPeers[0].identifier, s.expectedPeers[0].subscribers[1])

				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)

				return nil
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "no notification after release"
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}

			sub := transporttest.NewMockPeerSubscriber(mockCtrl)
			sub.EXPECT().NotifyStatusChanged(gomock.Any()).Times(0)

			pid := hostport.NewPeerIdentifier("localhost:1234")

			s.appliedFunc = func(a *Agent) error {
				peer, _ := a.RetainPeer(pid, sub)
				a.ReleasePeer(pid, sub)

				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)

				return nil
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "notification before versus after release"
			s.agent = NewDefaultAgent()
			s.expectedPeers = []peerExpectation{}

			pid := hostport.NewPeerIdentifier("localhost:1234")

			sub := transporttest.NewMockPeerSubscriber(mockCtrl)
			sub.EXPECT().NotifyStatusChanged(peerIdentifierMatcher(pid)).Times(1)

			s.appliedFunc = func(a *Agent) error {
				peer, _ := a.RetainPeer(pid, sub)

				a.NotifyStatusChanged(peer)

				a.ReleasePeer(pid, sub)

				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)
				a.NotifyStatusChanged(peer)

				return nil
			}
			return
		}(),
	}

	for _, tt := range tests {
		agent := tt.agent

		err := tt.appliedFunc(agent)

		assert.Equal(t, tt.expectedErr, err, tt.msg)
		assert.Equal(t, len(tt.expectedPeers), len(agent.peerNodes), tt.msg)
		for _, expectedPeerNode := range tt.expectedPeers {
			peerNode, ok := agent.peerNodes[expectedPeerNode.identifier.Identifier()]
			assert.True(t, ok, tt.msg)

			assert.Equal(t, expectedPeerNode.identifier.Identifier(), peerNode.peer.Identifier(), tt.msg)

			assert.Equal(t, len(expectedPeerNode.subscribers), len(peerNode.references), tt.msg)
			for _, expectedSubscriber := range expectedPeerNode.subscribers {
				subExists, ok := peerNode.references[expectedSubscriber]
				assert.True(t, ok && subExists, "subscriber (%v) not in list (%v). %s", expectedSubscriber, peerNode.references, tt.msg)
			}
		}
	}
}

func TestAgentClient(t *testing.T) {
	agent := NewDefaultAgent()

	assert.NotNil(t, agent.GetClient())
}
