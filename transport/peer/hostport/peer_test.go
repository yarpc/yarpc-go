package hostport

import (
	"testing"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPeerIdenfier(t *testing.T) {
	tests := []struct {
		hostport           string
		expectedIdentifier string
	}{
		{
			"localhost:12345",
			"localhost:12345",
		},
		{
			"123.123.123.123:12345",
			"123.123.123.123:12345",
		},
	}

	for _, tt := range tests {
		pi := PeerIdentifier(tt.hostport)

		assert.Equal(t, tt.expectedIdentifier, pi.Identifier())
	}
}

func TestPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		msg                string
		pid                PeerIdentifier
		agent              transport.Agent
		appliedFunc        func(*Peer)
		expectedIdentifier string
		expectedHostPort   string
		expectedStatus     transport.PeerStatus
		expectedAgent      transport.Agent
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "create"
			s.pid = PeerIdentifier("localhost:12345")
			s.agent = transporttest.NewMockAgent(mockCtrl)

			s.appliedFunc = func(p *Peer) {}

			s.expectedIdentifier = "localhost:12345"
			s.expectedHostPort = "localhost:12345"
			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerUnavailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start request"
			agent := transporttest.NewMockAgent(mockCtrl)
			agent.EXPECT().NotifyStatusChanged(gomock.Any()).Times(1)
			s.agent = agent

			s.appliedFunc = func(p *Peer) {
				p.StartRequest()
			}

			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 1,
				ConnectionStatus:    transport.PeerUnavailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start request stop request"
			agent := transporttest.NewMockAgent(mockCtrl)
			agent.EXPECT().NotifyStatusChanged(gomock.Any()).Times(2)
			s.agent = agent

			s.appliedFunc = func(p *Peer) {
				done := p.StartRequest()
				done()
			}

			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerUnavailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start 5 stop 2"
			agent := transporttest.NewMockAgent(mockCtrl)
			agent.EXPECT().NotifyStatusChanged(gomock.Any()).Times(7)
			s.agent = agent

			s.appliedFunc = func(p *Peer) {
				done1 := p.StartRequest()
				p.StartRequest()
				p.StartRequest()
				done2 := p.StartRequest()
				done1()
				p.StartRequest()
				done2()
			}

			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 3,
				ConnectionStatus:    transport.PeerUnavailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start 5 stop 5"
			agent := transporttest.NewMockAgent(mockCtrl)
			agent.EXPECT().NotifyStatusChanged(gomock.Any()).Times(10)
			s.agent = agent

			s.appliedFunc = func(p *Peer) {
				for i := 0; i < 5; i++ {
					done := p.StartRequest()
					defer done()
				}
			}

			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerUnavailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "set status"

			s.appliedFunc = func(p *Peer) {
				p.SetStatus(transport.PeerAvailable)
			}

			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
	}

	for _, tt := range tests {
		if tt.pid == PeerIdentifier("") {
			tt.pid = PeerIdentifier("localhost:12345")
			tt.expectedIdentifier = "localhost:12345"
			tt.expectedHostPort = "localhost:12345"
		}
		if tt.agent == nil {
			tt.agent = transporttest.NewMockAgent(mockCtrl)
			tt.expectedAgent = tt.agent
		}

		peer := NewPeer(tt.pid, tt.agent)

		tt.appliedFunc(peer)

		assert.Equal(t, tt.expectedIdentifier, peer.Identifier(), tt.msg)
		assert.Equal(t, tt.expectedHostPort, peer.HostPort(), tt.msg)
		assert.Equal(t, tt.expectedAgent, peer.Agent(), tt.msg)
		assert.Equal(t, tt.expectedStatus, peer.Status(), tt.msg)
	}
}
