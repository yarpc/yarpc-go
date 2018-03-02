package shardproxy

import (
	"context"
	"sync"
	"time"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	yarpcerrors "go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

var (
	_noContextDeadlineError = "can't wait for peer without a context deadline for a %s peer list"
)

// blockingStatus represents how the unavailability of peers will be handled
// by the sharder and chooser.
type blockingStatus int

const (
	// noBlocking means that peers will immediately error if there is no peer
	// for a shard.
	noBlockingForPeer blockingStatus = 0

	// blockForPeer means that peers will use the given context to block on
	// peers indefinitely.
	blockForPeer blockingStatus = 1

	// blockForPeerWithCap means that peers will wait until a peer is available
	// for a shard, or if the ctx times out, OR if the provided time cap passes.
	blockForPeerWithCap blockingStatus = 2
)

type chooserOpts struct {
	shardName          string
	logger             *zap.Logger
	scope              *metrics.Scope
	peerBlockType      blockingStatus
	waitForPeerTimeout time.Duration
}

func newShardChooser(opts chooserOpts) *shardChooser {
	if opts.logger == nil {
		opts.logger = zap.NewNop()
	}
	numPeersGauge, err := opts.scope.Gauge(metrics.Spec{
		Name: "yarpc_peer_sharder_num_peers_per_shard",
		Help: "The number of peer that are currently available for sharding per shard.",
		ConstTags: metrics.Tags{
			"shard": opts.shardName,
		},
	})
	if err != nil {
		opts.logger.DPanic(
			"failed to create counter guage for shard chooser",
			zap.String("shardKey", opts.shardName),
			zap.Error(err),
		)
	}

	return &shardChooser{
		list:               newPeerRing(),
		shardName:          opts.shardName,
		numPeersGauge:      numPeersGauge,
		peerAvailableEvent: make(chan struct{}, 1),
		nodes:              make(map[string]*node),
		peerBlockType:      opts.peerBlockType,
		waitForPeerTimeout: opts.waitForPeerTimeout,
	}
}

type shardChooser struct {
	sync.RWMutex

	shardName     string
	numPeersGauge *metrics.Gauge

	list  *peerRing
	nodes map[string]*node

	peerAvailableEvent chan struct{}

	peerBlockType      blockingStatus
	waitForPeerTimeout time.Duration
}

func (s *shardChooser) Add(thunk *peerThunk) {
	s.Lock()
	if _, isDuplicate := s.nodes[thunk.Identifier()]; isDuplicate {
		s.Unlock()
		return
	}
	s.numPeersGauge.Inc()
	node := s.list.Add(thunk)
	s.nodes[thunk.Identifier()] = node
	s.Unlock()
	s.notifyPeerAvailable()
}

func (s *shardChooser) Remove(thunk *peerThunk) {
	s.Lock()
	n, ok := s.nodes[thunk.Identifier()]
	if !ok {
		// Invalid remove.
		s.Unlock()
		return
	}
	s.numPeersGauge.Dec()
	s.list.Remove(n)
	delete(s.nodes, thunk.Identifier())
	s.Unlock()
}

// Choose selects the next available peer in the peer list.  It implements the
// transport#Chooser api.
func (s *shardChooser) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	for {
		s.Lock()
		p := s.list.Choose(ctx, req)
		s.Unlock()

		if p != nil {
			t := p.(*peerThunk)
			s.notifyPeerAvailable()
			t.onStart()
			return t.peer.callablePeer, t.boundOnFinish, nil
		}
		if err := s.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (s *shardChooser) notifyPeerAvailable() {
	select {
	case s.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// waitForPeerAddedEvent waits until a peer is added to the peer list or the
// given context finishes.
// Must NOT be run in a mutex.Lock()
func (s *shardChooser) waitForPeerAddedEvent(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return s.newNoContextDeadlineError()
	}

	if s.peerBlockType == noBlockingForPeer {
		return yarpcerrors.UnavailableErrorf("%s peer list had no peers available for: %s", "sharder", s.shardName)
	}

	if s.peerBlockType == blockForPeer {
		return s.waitWithoutCap(ctx)
	}

	return s.waitWithCap(ctx, s.waitForPeerTimeout)
}

func (s *shardChooser) waitWithoutCap(ctx context.Context) error {
	select {
	case <-s.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return s.newUnavailableError(ctx.Err())
	}
}

func (s *shardChooser) waitWithCap(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-s.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return s.newUnavailableError(ctx.Err())
	case <-timeoutCtx.Done():
		return s.newUnavailableError(timeoutCtx.Err())
	}
}

func (s *shardChooser) newNoContextDeadlineError() error {
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, _noContextDeadlineError, "sharder")
}

func (s *shardChooser) newUnavailableError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%s peer list timed out waiting for peer: %s", "sharder", err.Error())
}
