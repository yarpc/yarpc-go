package peers

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
		pi          *HostPortPeerIdentifier
		agent       transport.PeerAgent
		appliedFunc func(transport.SubscribablePeer)
		assertFunc  func(*HostPortPeer)
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.pi = NewPeerIdentifier("localhost:12345")
			s.agent = transporttest.NewMockPeerAgent(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, s.pi.hostport, p.hostport)
				assert.Equal(t, s.pi.hostport, p.HostPort())
				assert.Equal(t, s.pi.Identifier(), p.Identifier())
				assert.Empty(t, p.references)
				assert.Equal(t, 0, p.Pending())
				assert.Equal(t, transport.PeerAvailable, p.GetStatus())
			}
			return
		}(),
		func() (s testStruct) {
			peerlist := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist)
			}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, 1, len(p.references))
				assert.Equal(t, true, p.references[peerlist])
			}
			return
		}(),
		func() (s testStruct) {
			peerlist := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist)
				p.OnRelease(peerlist)
			}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, 0, len(p.references))

				_, ok := p.references[peerlist]
				assert.Equal(t, false, ok)
			}
			return
		}(),
		func() (s testStruct) {
			peerlist1 := transporttest.NewMockPeerList(mockCtrl)
			peerlist2 := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist1)
				p.OnRetain(peerlist2)
				p.OnRelease(peerlist1)
			}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, 1, len(p.references))

				val, ok := p.references[peerlist1]
				assert.Equal(t, false, ok)
				assert.Equal(t, false, val)

				val, ok = p.references[peerlist2]
				assert.Equal(t, true, ok)
				assert.Equal(t, true, val)
			}
			return
		}(),
		func() (s testStruct) {
			peerlist1 := transporttest.NewMockPeerList(mockCtrl)
			peerlist2 := transporttest.NewMockPeerList(mockCtrl)
			peerlist3 := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist1)
				p.OnRetain(peerlist2)
				p.OnRetain(peerlist3)
			}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, 3, len(p.references))

				val, ok := p.references[peerlist1]
				assert.Equal(t, true, ok)
				assert.Equal(t, true, val)

				val, ok = p.references[peerlist2]
				assert.Equal(t, true, ok)
				assert.Equal(t, true, val)

				val, ok = p.references[peerlist3]
				assert.Equal(t, true, ok)
				assert.Equal(t, true, val)
			}
			return
		}(),
		func() (s testStruct) {
			peerlist1 := transporttest.NewMockPeerList(mockCtrl)
			peerlist2 := transporttest.NewMockPeerList(mockCtrl)
			peerlist3 := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist1)
				p.OnRetain(peerlist2)
			}
			s.assertFunc = func(p *HostPortPeer) {
				err := p.OnRelease(peerlist3)
				assert.NotNil(t, err)
			}
			return
		}(),
		func() (s testStruct) {
			peerlist1 := transporttest.NewMockPeerList(mockCtrl)
			s.appliedFunc = func(p transport.SubscribablePeer) {
				p.OnRetain(peerlist1)
				p.IncPending()
				p.IncPending()
				p.IncPending()
				p.IncPending()
				p.DecPending()
				p.DecPending()
			}
			s.assertFunc = func(p *HostPortPeer) {
				assert.Equal(t, 2, p.Pending())
			}
			return
		}(),
	}

	for _, tt := range tests {
		if tt.pi == nil {
			tt.pi = NewPeerIdentifier("localhost:12345")
		}
		if tt.agent == nil {
			tt.agent = transporttest.NewMockPeerAgent(mockCtrl)
		}

		peer := NewPeer(tt.pi, tt.agent)

		tt.appliedFunc(peer)

		tt.assertFunc(peer)
	}
}
