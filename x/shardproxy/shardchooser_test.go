package shardproxy

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/zap"
)

func TestDuplicatePeerAdd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPeer := peertest.NewMockPeer(mockCtrl)
	mockPeer.EXPECT().Identifier().Return("123.123.123.123").AnyTimes()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	require.NoError(t, p.SetPeer(&fakeCallablePeer{mockPeer}))

	sc := newShardChooser(chooserOpts{
		shardName: "test",
		logger:    zap.NewNop(),
		scope:     metrics.New().Scope(),
	})

	sc.Add(p)
	sc.Add(p) // Should be noop

	assert.Len(t, sc.nodes, 1)
	assert.Equal(t, int64(1), sc.numPeersGauge.Load())
}

func TestAddRemove(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPeer := peertest.NewMockPeer(mockCtrl)
	mockPeer.EXPECT().Identifier().Return("123.123.123.123").AnyTimes()
	mockPeer2 := peertest.NewMockPeer(mockCtrl)
	mockPeer2.EXPECT().Identifier().Return("123.123.123.124").AnyTimes()

	p := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	require.NoError(t, p.SetPeer(&fakeCallablePeer{mockPeer}))
	p2 := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	require.NoError(t, p2.SetPeer(&fakeCallablePeer{mockPeer2}))

	sc := newShardChooser(chooserOpts{
		shardName: "test",
		logger:    zap.NewNop(),
		scope:     metrics.New().Scope(),
	})

	sc.Add(p)
	assert.Equal(t, int64(1), sc.numPeersGauge.Load())

	sc.Remove(p)
	assert.Equal(t, int64(0), sc.numPeersGauge.Load())

	sc.Add(p)
	sc.Add(p2)
	assert.Equal(t, int64(2), sc.numPeersGauge.Load())

	sc.Remove(p2)
	assert.Equal(t, int64(1), sc.numPeersGauge.Load())

	sc.Remove(p)
	assert.Equal(t, int64(0), sc.numPeersGauge.Load())
}

func TestInvalidRemovePeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mp1 := peertest.NewMockPeer(mockCtrl)
	mp1.EXPECT().Identifier().Return("123.123.123.123").AnyTimes()
	p1 := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	require.NoError(t, p1.SetPeer(&fakeCallablePeer{mp1}))

	mp2 := peertest.NewMockPeer(mockCtrl)
	mp2.EXPECT().Identifier().Return("1.1.1.1").AnyTimes()
	p2 := newThunk(fakeSharder(t), peertest.NewMockIdentifier(mockCtrl))
	require.NoError(t, p2.SetPeer(&fakeCallablePeer{mp2}))

	sc := newShardChooser(chooserOpts{
		shardName: "test",
		logger:    zap.NewNop(),
		scope:     metrics.New().Scope(),
	})

	sc.Add(p1)
	sc.Remove(p2) // Should be noop

	assert.Len(t, sc.nodes, 1)
	assert.Equal(t, int64(1), sc.numPeersGauge.Load())
}

func TestRequestWithNoDeadline(t *testing.T) {
	sc := newShardChooser(chooserOpts{shardName: "test", logger: zap.NewNop()})

	p, _, err := sc.Choose(context.Background(), nil)

	require.Nil(t, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't wait for peer without a context deadline")
}
