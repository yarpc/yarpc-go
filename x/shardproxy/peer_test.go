package shardproxy

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcconfig"
)

func TestInvalidPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))

	err := p.SetPeer(peertest.NewMockPeer(mockCtrl))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code:internal")
	assert.Contains(t, err.Error(), "non-callable yarpc peer.Peer")
}

func TestPeerStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))

	mockPeer := peertest.NewMockPeer(mockCtrl)
	mockPeer.EXPECT().Status().Return(peer.Status{PendingRequestCount: 1})

	err := p.SetPeer(&fakeCallablePeer{mockPeer})
	require.NoError(t, err)

	assert.Equal(t, 1, p.Status().PendingRequestCount)
}

func TestPeerStartAndEndRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))

	mockPeer := peertest.NewMockPeer(mockCtrl)
	mockPeer.EXPECT().StartRequest().Times(1)
	mockPeer.EXPECT().EndRequest().Times(1)

	err := p.SetPeer(&fakeCallablePeer{mockPeer})
	require.NoError(t, err)

	p.StartRequest()
	p.EndRequest()
}

func TestPeerNoopFuncs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	mockPeer := peertest.NewMockPeer(mockCtrl)
	require.NoError(t, p.SetPeer(&fakeCallablePeer{mockPeer}))
	require.NoError(t, p.Start())
	require.True(t, p.IsRunning())
	require.NoError(t, p.Stop())
	require.Empty(t, p.Transports())
}

func TestDuplicatePeerStatusChangedIsNoop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))

	p.onStatusChanged()
	p.onStatusChanged()
	p.onStatusChanged()

	select {
	case <-p.changed:
	default:
		require.Fail(t, "changed flag was not set")
	}

	select {
	case <-p.changed:
		require.Fail(t, "changed flag should only be set once")
	default:
	}
}

func TestDuplicateRemoveAndGetPeers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))

	assert.Empty(t, p.RemoveAndGetShards())
	assert.Empty(t, p.RemoveAndGetShards())
	assert.Empty(t, p.RemoveAndGetShards())
}

type fakeCallablePeer struct {
	*peertest.MockPeer
}

func (*fakeCallablePeer) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return nil, errors.New("unimplemented")
}

func fakeSharder(t *testing.T) *sharder {
	strategy, err := yarpcconfig.Backoff{Exponential: yarpcconfig.ExponentialBackoff{}}.Strategy()
	require.NoError(t, err)

	return &sharder{
		peerBackoffStrategy: strategy,
	}
}
