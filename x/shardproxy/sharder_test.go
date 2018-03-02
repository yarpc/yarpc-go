package shardproxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/yarpcconfig"
)

func TestSharderIsRunning(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tran := peertest.NewMockTransport(mockCtrl)
	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	require.False(t, s.IsRunning())
	require.NoError(t, s.Start())
	require.True(t, s.IsRunning())
	require.NoError(t, s.Stop())
	require.False(t, s.IsRunning())
}

func TestSharderWithUninitializedPeers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()
	mPeer := peertest.NewMockPeer(mockCtrl)
	mPeer.EXPECT().Status().Return(peer.Status{ConnectionStatus: peer.Unavailable}).AnyTimes()
	p1 := &fakeCallablePeer{mPeer}

	pid2 := peertest.NewMockIdentifier(mockCtrl)
	pid2.EXPECT().Identifier().Return("pid2").AnyTimes()

	tran := peertest.NewMockTransport(mockCtrl)
	tran.EXPECT().RetainPeer(pid1, gomock.Any()).Return(p1, nil)
	tran.EXPECT().ReleasePeer(pid1, gomock.Any()).Return(nil)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	err = s.Update(peer.ListUpdates{
		Additions: []peer.Identifier{pid1, pid2},
	})
	require.NoError(t, err)
	require.Len(t, s.uninitializedPeers, 2)

	err = s.Update(peer.ListUpdates{
		Removals: []peer.Identifier{pid2},
	})
	require.NoError(t, err)
	require.Len(t, s.uninitializedPeers, 1)

	require.NoError(t, s.Start())
	require.Len(t, s.initializedPeers, 1)
	require.Len(t, s.uninitializedPeers, 0)

	require.NoError(t, s.Stop())
	require.Len(t, s.initializedPeers, 0)
	require.Len(t, s.uninitializedPeers, 1)
}

func TestSharderUninitializedInvalidRemove(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()

	tran := peertest.NewMockTransport(mockCtrl)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	err = s.Update(peer.ListUpdates{
		Removals: []peer.Identifier{pid1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't remove peer")
	require.Len(t, s.uninitializedPeers, 0)
}

func TestSharderInitializedInvalidRemove(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()

	tran := peertest.NewMockTransport(mockCtrl)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	require.NoError(t, s.Start())

	err = s.Update(peer.ListUpdates{
		Removals: []peer.Identifier{pid1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't remove peer")
	require.Len(t, s.initializedPeers, 0)

	require.NoError(t, s.Stop())
}

func TestSharderInitializedDuplicateAdd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()
	mPeer := peertest.NewMockPeer(mockCtrl)
	mPeer.EXPECT().Status().Return(peer.Status{ConnectionStatus: peer.Unavailable}).AnyTimes()
	p1 := &fakeCallablePeer{mPeer}

	tran := peertest.NewMockTransport(mockCtrl)
	tran.EXPECT().RetainPeer(pid1, gomock.Any()).Return(p1, nil)
	tran.EXPECT().ReleasePeer(pid1, gomock.Any()).Return(nil)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	require.NoError(t, s.Start())

	err = s.Update(peer.ListUpdates{
		Additions: []peer.Identifier{pid1},
	})
	require.NoError(t, err)

	err = s.Update(peer.ListUpdates{
		Additions: []peer.Identifier{pid1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't add peer")
	require.Len(t, s.initializedPeers, 1)

	require.NoError(t, s.Stop())
}

func TestSharderInitializedRetainError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()

	retainErr := errors.New("bad times")

	tran := peertest.NewMockTransport(mockCtrl)
	tran.EXPECT().RetainPeer(pid1, gomock.Any()).Return(nil, retainErr)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	require.NoError(t, s.Start())

	err = s.Update(peer.ListUpdates{
		Additions: []peer.Identifier{pid1},
	})
	require.Error(t, err)
	require.EqualError(t, err, retainErr.Error())
	require.Len(t, s.initializedPeers, 0)

	require.NoError(t, s.Stop())
}

func TestSharderInvalidPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pid1 := peertest.NewMockIdentifier(mockCtrl)
	pid1.EXPECT().Identifier().Return("pid1").AnyTimes()
	p1 := peertest.NewMockPeer(mockCtrl)

	tran := peertest.NewMockTransport(mockCtrl)
	tran.EXPECT().RetainPeer(pid1, gomock.Any()).Return(p1, nil)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	require.NoError(t, s.Start())

	err = s.Update(peer.ListUpdates{
		Additions: []peer.Identifier{pid1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot use shard peer list with non-callable yarpc peer.Peer")
	require.Len(t, s.initializedPeers, 0)

	require.NoError(t, s.Stop())
}

func TestSharderChooseWhenNotRunning(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tran := peertest.NewMockTransport(mockCtrl)

	s, err := newFakeChooserList(tran)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	_, _, err = s.Choose(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer list is not running")
	require.Len(t, s.initializedPeers, 0)
}

func newFakeChooserList(tran peer.Transport) (*sharder, error) {
	cb := &chooserBuilder{
		ServiceName: "myservice",
	}
	cl, err := cb.newChooserList(
		sharderConfig{
			PeerService: "service",
			PeerBackoff: yarpcconfig.Backoff{
				Exponential: yarpcconfig.ExponentialBackoff{
					First: time.Millisecond * 10,
					Max:   time.Second,
				},
			},
			MaxPeerCheckInterval: time.Minute,
			HealthReqTimeout:     time.Second,
			ShardReqTimeout:      time.Second,
		},
		tran,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return cl.(*sharder), nil
}
