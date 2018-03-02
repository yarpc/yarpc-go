package shardproxy

import (
	"errors"
	"time"

	"go.uber.org/fx"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/x/shardproxy/internal/gen/go/github.com/uber/tchannel/meta"
	sharder2 "go.uber.org/yarpc/x/shardproxy/internal/gen/go/go.uber.org/yarpc/sharder"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/zap"
)

// Module provides a sharder PeerListSpec into the fx graph.
var Module = fx.Provide(New)

// Params are the input params for the Sharder.
type Params struct {
	fx.In

	Service        string `name:"service"`
	Logger         *zap.Logger
	Scope          *metrics.Scope
	EdgeMiddleware middleware.UnaryOutbound `name:"edge" optional:"true"`
}

// Results returns a PeerListSpec that will be inserted into the yarpcfx module.
type Results struct {
	fx.Out

	Spec yarpcconfig.PeerListSpec `group:"yarpcfx"`
}

// New creates a new PeerListSpec for sharders.
func New(p Params) (Results, error) {
	cb := &chooserBuilder{
		ServiceName:    p.Service,
		Logger:         p.Logger,
		Scope:          p.Scope,
		EdgeMiddleware: p.EdgeMiddleware,
	}
	return Results{
		Spec: yarpcconfig.PeerListSpec{
			Name:          "self-assigned-sharder",
			BuildPeerList: cb.newChooserList,
		},
	}, nil
}

type sharderConfig struct {
	// PeerService is the "service" name that will be used to make health and
	// shard requests to the peer.
	PeerService string `config:"peer-service"`

	// PeerBackoff is the backoff strategy that will be used when the connection
	// of a peer has errored, or the peer is unavailable.
	PeerBackoff yarpcconfig.Backoff `config:"peer-backoff"`

	// MaxPeerCheckInterval is the max time that will be waited between
	// successful calls to the peer's shard and health endpoints. The smaller
	// this is the more requests will be sent to the peer to validate the shard
	// and health state (this will have a cost). If the peer returns an
	// "unavailable" error this will short-circuit the peer check timeout.
	// This will have full jitter applied (will bottom out at the
	// MinPeerCheckInterval).
	MaxPeerCheckInterval time.Duration `config:"max-peer-check-interval"`

	// MinPeerCheckInterval is the minimum time that will be waited between
	// successful calls to the peer's shard and health endpoints.  This
	// guarantees a lower bound on all timeouts between peer checks.
	MinPeerCheckInterval time.Duration `config:"min-peer-check-interval"`

	// HealthReqTimeout is the timeout that will be set for requests to the peer
	// meta::health endpoint.
	HealthReqTimeout time.Duration `config:"health-request-timeout"`

	// ShardReqTimeout is the timeout that will be set for requests to the peer
	// shard::shardInfo endpoint.
	ShardReqTimeout time.Duration `config:"shard-request-timeout"`

	// WaitForPeerTimeout is a configuration option that will change how long a
	// request with no peers will "wait" for peers to become available. Waiting
	// for peers will have implications if the cardinality of the shards is
	// quite large (it will create a new peer chooser if one doesn't already
	// exist in order to block against it.
	WaitForPeerTimeout *time.Duration `config:"wait-for-peer-timeout"`
}

type chooserBuilder struct {
	ServiceName    string
	Logger         *zap.Logger
	Scope          *metrics.Scope
	EdgeMiddleware middleware.UnaryOutbound
}

func (cb *chooserBuilder) newChooserList(c sharderConfig, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error) {
	if cb.Logger == nil {
		cb.Logger = zap.NewNop()
	}
	if c.HealthReqTimeout == time.Duration(0) {
		return nil, errors.New("must provide `health-request-timeout` for self-assigned-sharder")
	}
	if c.ShardReqTimeout == time.Duration(0) {
		return nil, errors.New("must provide `shard-request-timeout` for self-assigned-sharder")
	}
	if c.PeerService == "" {
		return nil, errors.New("must provide `peer-service` for self-assigned-sharder")
	}
	if c.MaxPeerCheckInterval == time.Duration(0) {
		return nil, errors.New("must provide `max-peer-check-interval` for self-assigned-sharder")
	}
	if c.MaxPeerCheckInterval < c.MinPeerCheckInterval {
		return nil, errors.New("`max-peer-check-interval` must be larger than `min-peer-check-interval` for self-assigned-sharder")
	}
	strategy, err := c.PeerBackoff.Strategy()
	if err != nil {
		return nil, err
	}
	peerBlockType := blockForPeer
	waitForPeerTimeout := time.Duration(0)
	if c.WaitForPeerTimeout != nil {
		waitForPeerTimeout = *c.WaitForPeerTimeout
		peerBlockType = blockForPeerWithCap
		if waitForPeerTimeout == time.Duration(0) {
			peerBlockType = noBlockingForPeer
		}
	}

	reqType := meta.HealthRequestTypeTraffic
	healthRequest := meta.HealthRequest{
		Type: &reqType,
	}
	shardRequest := sharder2.ShardInfoRequest{}
	scope := cb.Scope.Tagged(metrics.Tags{
		"component": "self_assigned_sharder",
	})
	return &sharder{
		once:                 lifecycle.NewOnce(),
		name:                 "self-assigned-sharder",
		logger:               cb.Logger,
		scope:                scope,
		serviceName:          cb.ServiceName,
		peerServiceName:      c.PeerService,
		peerBackoffStrategy:  strategy,
		maxPeerCheckInterval: c.MaxPeerCheckInterval,
		minPeerCheckInterval: c.MinPeerCheckInterval,
		healthReqTimeout:     c.HealthReqTimeout,
		shardReqTimeout:      c.ShardReqTimeout,
		shardChoosers:        make(map[string]*shardChooser),
		uninitializedPeers:   make(map[string]peer.Identifier),
		initializedPeers:     make(map[string]*peerThunk),
		healthRequest:        healthRequest,
		shardRequest:         shardRequest,
		peerBlockType:        peerBlockType,
		waitForPeerTimeout:   waitForPeerTimeout,
		transport:            t,
		edgeMiddleware:       cb.EdgeMiddleware,
	}, nil
}
