package shardproxy

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer/peertest"
)

func TestInvalidConfiguration(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tran := peertest.NewMockTransport(mockCtrl)

	cb := &chooserBuilder{
		ServiceName: "test",
	}
	cfg := sharderConfig{}

	_, err := cb.newChooserList(cfg, tran, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must provide `health-request-timeout` for self-assigned-sharder")

	cfg.HealthReqTimeout = time.Second

	_, err = cb.newChooserList(cfg, tran, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must provide `shard-request-timeout` for self-assigned-sharder")

	cfg.ShardReqTimeout = time.Second

	_, err = cb.newChooserList(cfg, tran, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must provide `peer-service` for self-assigned-sharder")

	cfg.PeerService = "peerservice"

	_, err = cb.newChooserList(cfg, tran, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must provide `max-peer-check-interval` for self-assigned-sharder")

	cfg.MaxPeerCheckInterval = time.Minute

	_, err = cb.newChooserList(cfg, tran, nil)
	require.NoError(t, err)
}
