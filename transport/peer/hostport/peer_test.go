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
		pi := NewPeerIdentifier(tt.hostport)

		assert.Equal(t, tt.expectedIdentifier, pi.Identifier())
	}
}

func TestPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testStruct struct {
		msg                string
		pi                 PeerIdentifier
		agent              transport.PeerAgent
		subscriber         transport.PeerSubscriber
		appliedFunc        func(*Peer)
		expectedIdentifier string
		expectedHostPort   string
		expectedStatus     transport.PeerStatus
		expectedAgent      transport.PeerAgent
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.msg = "create"
			s.pi = NewPeerIdentifier("localhost:12345")
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.subscriber = transporttest.NewMockPeerSubscriber(mockCtrl)
			s.appliedFunc = func(p *Peer) {}
			s.expectedIdentifier = "localhost:12345"
			s.expectedHostPort = "localhost:12345"
			s.expectedAgent = s.agent
			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start request"
			subscriber := transporttest.NewMockPeerSubscriber(mockCtrl)
			subscriber.EXPECT().NotifyStatusChanged(gomock.Any()).Times(1)
			s.subscriber = subscriber

			s.appliedFunc = func(p *Peer) {
				p.StartRequest()
			}

			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 1,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start request stop request"
			subscriber := transporttest.NewMockPeerSubscriber(mockCtrl)
			subscriber.EXPECT().NotifyStatusChanged(gomock.Any()).Times(2)
			s.subscriber = subscriber

			s.appliedFunc = func(p *Peer) {
				done := p.StartRequest()
				done()
			}

			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start 5 stop 2"
			subscriber := transporttest.NewMockPeerSubscriber(mockCtrl)
			subscriber.EXPECT().NotifyStatusChanged(gomock.Any()).Times(7)
			s.subscriber = subscriber

			s.appliedFunc = func(p *Peer) {
				done1 := p.StartRequest()
				p.StartRequest()
				p.StartRequest()
				done2 := p.StartRequest()
				done1()
				p.StartRequest()
				done2()
			}

			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 3,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
		func() (s testStruct) {
			s.msg = "start 5 stop 5"
			subscriber := transporttest.NewMockPeerSubscriber(mockCtrl)
			subscriber.EXPECT().NotifyStatusChanged(gomock.Any()).Times(10)
			s.subscriber = subscriber

			s.appliedFunc = func(p *Peer) {
				for i := 0; i < 5; i++ {
					done := p.StartRequest()
					defer done()
				}
			}

			s.expectedStatus = transport.PeerStatus{
				PendingRequestCount: 0,
				ConnectionStatus:    transport.PeerAvailable,
			}
			return
		}(),
	}

	for _, tt := range tests {
		if tt.pi == PeerIdentifier("") {
			tt.pi = NewPeerIdentifier("localhost:12345")
			tt.expectedIdentifier = "localhost:12345"
			tt.expectedHostPort = "localhost:12345"
		}
		if tt.agent == nil {
			tt.agent = transporttest.NewMockPeerAgent(mockCtrl)
			tt.expectedAgent = tt.agent
		}

		peer := NewPeer(tt.pi, tt.agent, tt.subscriber)

		tt.appliedFunc(peer)

		assert.Equal(t, tt.expectedIdentifier, peer.Identifier(), tt.msg)
		assert.Equal(t, tt.expectedHostPort, peer.HostPort(), tt.msg)
		assert.Equal(t, tt.expectedAgent, peer.GetAgent(), tt.msg)
		assert.Equal(t, tt.expectedStatus, peer.GetStatus(), tt.msg)
	}
}
