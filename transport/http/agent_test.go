package http

import (
	"testing"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/peers"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/golang/mock/gomock"
)

func TestAgent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		agent       *Agent
		appliedFunc func(transport.PeerAgent)
		assertFunc  func(*Agent)
	}
	tests := []testStruct{
		func() (s testStruct) {
			msg := "one retain"
			s.agent = NewDefaultAgent()
			pid := peers.NewPeerIdentifier("localhost:1234")
			s.appliedFunc = func(a transport.PeerAgent) {
				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				a.RetainPeer(pid, sub)
			}
			s.assertFunc = func(a *Agent) {
				assert.Len(t, a.peers, 1, msg)

				peer := a.peers[pid.Identifier()]
				assert.Equal(t, pid.Identifier(), peer.Identifier(), msg)
				assert.Equal(t, 1, peer.References(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "three retains"
			s.agent = NewDefaultAgent()
			pid := peers.NewPeerIdentifier("localhost:1234")
			s.appliedFunc = func(a transport.PeerAgent) {
				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub2 := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub3 := transporttest.NewMockPeerSubscriber(mockCtrl)
				a.RetainPeer(pid, sub)
				a.RetainPeer(pid, sub2)
				a.RetainPeer(pid, sub3)
			}
			s.assertFunc = func(a *Agent) {
				assert.Len(t, a.peers, 1, msg)
				peer := a.peers[pid.Identifier()]
				assert.Equal(t, 3, peer.References(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "three retains, one release"
			s.agent = NewDefaultAgent()
			pid := peers.NewPeerIdentifier("localhost:1234")
			s.appliedFunc = func(a transport.PeerAgent) {
				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub2 := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub3 := transporttest.NewMockPeerSubscriber(mockCtrl)
				a.RetainPeer(pid, sub)
				a.RetainPeer(pid, sub2)
				a.ReleasePeer(pid, sub)
				a.RetainPeer(pid, sub3)
			}
			s.assertFunc = func(a *Agent) {
				assert.Len(t, a.peers, 1, msg)
				peer := a.peers[pid.Identifier()]
				assert.Equal(t, 2, peer.References(), msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "three retains, three release"
			s.agent = NewDefaultAgent()
			pid := peers.NewPeerIdentifier("localhost:1234")
			s.appliedFunc = func(a transport.PeerAgent) {
				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub2 := transporttest.NewMockPeerSubscriber(mockCtrl)
				sub3 := transporttest.NewMockPeerSubscriber(mockCtrl)
				a.RetainPeer(pid, sub)
				a.RetainPeer(pid, sub2)
				a.ReleasePeer(pid, sub)
				a.RetainPeer(pid, sub3)
				a.ReleasePeer(pid, sub2)
				a.ReleasePeer(pid, sub3)
			}
			s.assertFunc = func(a *Agent) {
				assert.Len(t, a.peers, 0, msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "no retains, one release"
			s.agent = NewDefaultAgent()
			pid := peers.NewPeerIdentifier("localhost:1234")
			var err error
			s.appliedFunc = func(a transport.PeerAgent) {
				sub := transporttest.NewMockPeerSubscriber(mockCtrl)
				err = a.ReleasePeer(pid, sub)
			}
			s.assertFunc = func(a *Agent) {
				assert.NotNil(t, err, msg)

				errAgent, ok := err.(errors.ErrAgentHasNoReferenceToPeer)
				assert.True(t, ok, msg)
				assert.Equal(t, a, errAgent.Agent, msg)
				assert.Equal(t, pid, errAgent.PeerIdentifier, msg)
			}
			return
		}(),
		func() (s testStruct) {
			msg := "one retains, one release (from different subscriber)"
			s.agent = NewDefaultAgent()
			var err error
			pid := peers.NewPeerIdentifier("localhost:1234")
			invalidSub := transporttest.NewMockPeerSubscriber(mockCtrl)
			s.appliedFunc = func(a transport.PeerAgent) {
				sub1 := transporttest.NewMockPeerSubscriber(mockCtrl)
				a.RetainPeer(pid, sub1)
				err = a.ReleasePeer(pid, invalidSub)
			}
			s.assertFunc = func(a *Agent) {
				assert.NotNil(t, err, msg)

				errRef, ok := err.(errors.ErrPeerHasNoReferenceToSubscriber)
				assert.True(t, ok, msg)
				assert.Equal(t, a.peers[pid.Identifier()], errRef.Peer, msg)
				assert.Equal(t, invalidSub, errRef.PeerSubscriber, msg)
			}
			return
		}(),
	}

	for _, tt := range tests {
		agent := tt.agent

		tt.appliedFunc(agent)

		tt.assertFunc(agent)
	}
}
