package shardproxy

import (
	"context"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/x/shardproxy/internal/gen/go/github.com/uber/tchannel/meta"
	sharder2 "go.uber.org/yarpc/x/shardproxy/internal/gen/go/go.uber.org/yarpc/sharder"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

type sharder struct {
	name string

	logger *zap.Logger
	scope  *metrics.Scope

	serviceName          string
	peerServiceName      string
	peerBackoffStrategy  backoff.Strategy
	maxPeerCheckInterval time.Duration
	minPeerCheckInterval time.Duration

	healthReqTimeout time.Duration
	shardReqTimeout  time.Duration

	shouldRetainPeers atomic.Bool

	peerLock            sync.RWMutex
	peerConnections     atomic.Int32
	peerConnectionGroup sync.WaitGroup
	uninitializedPeers  map[string]peer.Identifier
	initializedPeers    map[string]*peerThunk

	chooserLock   sync.RWMutex
	shardChoosers map[string]*shardChooser

	peerBlockType      blockingStatus
	waitForPeerTimeout time.Duration

	// Health and shard requests are the same across all peers, so we can share
	// the same objects.
	healthRequest meta.HealthRequest
	shardRequest  sharder2.ShardInfoRequest

	transport peer.Transport

	edgeMiddleware middleware.UnaryOutbound
	once           *lifecycle.Once
}

// IsRunning returns whether the peer list is running.
func (pl *sharder) IsRunning() bool {
	return pl.once.IsRunning()
}

// Start notifies the List that requests will start coming
func (pl *sharder) Start() error {
	return pl.once.Start(pl.start)
}

func (pl *sharder) start() error {
	pl.peerLock.Lock()
	defer pl.peerLock.Unlock()

	var errs error
	for k, pid := range pl.uninitializedPeers {
		errs = multierr.Append(errs, pl.addPeerIdentifier(pid))
		delete(pl.uninitializedPeers, k)
	}

	pl.shouldRetainPeers.Store(true)

	return errs
}

// Stop notifies the List that requests will stop coming
func (pl *sharder) Stop() error {
	return pl.once.Stop(pl.stop)
}

// stop will release all the peers from the list
func (pl *sharder) stop() error {
	pl.peerLock.Lock()
	defer pl.peerLock.Unlock()

	var errs error

	for _, t := range pl.initializedPeers {
		multierr.Append(errs, pl.removeThunk(t))
		pl.uninitializedPeers[t.id.Identifier()] = t.id
	}

	pl.shouldRetainPeers.Store(false)

	// Wait for all the peer goroutines to finish.
	pl.peerConnectionGroup.Wait()

	return errs
}

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures.
func (pl *sharder) Update(updates peer.ListUpdates) error {
	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	pl.peerLock.Lock()
	defer pl.peerLock.Unlock()

	if !pl.shouldRetainPeers.Load() {
		return pl.updateUninitialized(updates)
	}
	return pl.updateInitialized(updates)
}

// updateUninitialized applies peer list updates when the peer list
// is **not** able to retain peers, putting the updates into a single
// uninitialized peer list.
//
// Must be run inside a mutex.Lock()
func (pl *sharder) updateUninitialized(updates peer.ListUpdates) error {
	var errs error
	for _, peerID := range updates.Removals {
		if _, ok := pl.uninitializedPeers[peerID.Identifier()]; !ok {
			errs = multierr.Append(errs, peer.ErrPeerRemoveNotInList(peerID.Identifier()))
		}
		delete(pl.uninitializedPeers, peerID.Identifier())
	}
	for _, peerID := range updates.Additions {
		pl.uninitializedPeers[peerID.Identifier()] = peerID
	}

	return errs
}

// updateInitialized applies peer list updates when the peer list
// is able to retain peers, putting the updates into the available
// or unavailable containers.
//
// Must be run inside a mutex.Lock()
func (pl *sharder) updateInitialized(updates peer.ListUpdates) error {
	var errs error
	for _, peerID := range updates.Removals {
		errs = multierr.Append(errs, pl.removePeerIdentifier(peerID))
	}

	for _, peerID := range updates.Additions {
		errs = multierr.Append(errs, pl.addPeerIdentifier(peerID))
	}
	return errs
}

// removePeerIdentifier will go remove references to the peer identifier and release
// it from the transport
// Must be run in the peerLock.Lock()
func (pl *sharder) removePeerIdentifier(pid peer.Identifier) error {
	t := pl.getThunk(pid)
	if t == nil {
		return peer.ErrPeerRemoveNotInList(pid.Identifier())
	}
	return pl.removeThunk(t)
}

// removeThunk removes a peerThunk from the sharder, and releases it from the
// transport.
// Must be run in the peerLock.Lock()
func (pl *sharder) removeThunk(t *peerThunk) error {
	shards := t.RemoveAndGetShards()
	for shard := range shards {
		sc, ok := pl.getShardChooser(shard)
		if !ok {
			// There was no chooser.
			// TODO add stat/log
			continue
		}
		sc.Remove(t)
	}
	delete(pl.initializedPeers, t.id.Identifier())

	return pl.transport.ReleasePeer(t.id, t)
}

// Must be run in the peerLock.Lock()
func (pl *sharder) addPeerIdentifier(pid peer.Identifier) error {
	if t := pl.getThunk(pid); t != nil {
		return peer.ErrPeerAddAlreadyInList(pid.Identifier())
	}

	t := newThunk(pl, pid)
	// We need to lock to guarantee the peer is set before the transport starts
	// sending health info.
	p, err := pl.transport.RetainPeer(pid, t)
	if err != nil {
		return err
	}
	if err := t.SetPeer(p); err != nil {
		return err
	}
	pl.initializedPeers[t.id.Identifier()] = t

	// Start Probing for shard information (this function controls all
	// adds/removes from shards.
	pl.peerConnections.Inc()
	pl.peerConnectionGroup.Add(1)
	go t.MaintainConnection()

	return nil
}

// getThunk returns a peerThunk if one already exists.
// Must be run in the peerLock.Lock()
func (pl *sharder) getThunk(identifier peer.Identifier) *peerThunk {
	t, _ := pl.initializedPeers[identifier.Identifier()]
	return t
}

type shardUpdates struct {
	Additions []string
	Removals  []string
}

func (pl *sharder) UpdateShards(t *peerThunk, updates shardUpdates) {
	for _, removeShard := range updates.Removals {
		if sc, ok := pl.getShardChooser(removeShard); ok {
			sc.Remove(t)
		}
	}
	for _, addShard := range updates.Additions {
		sc := pl.getOrCreateShardChooser(addShard)
		sc.Add(t)
	}
}

// Choose selects the next available peer in the peer list
func (pl *sharder) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, pl.newNotRunningError(err)
	}

	if chooser, ok := pl.getShardChooser(req.ShardKey); ok {
		return chooser.Choose(ctx, req)
	}
	if pl.peerBlockType == noBlockingForPeer {
		return nil, nil, yarpcerrors.UnavailableErrorf("%s peer list had no peers available for: %s", "sharder", req.ShardKey)
	}

	chooser := pl.getOrCreateShardChooser(req.ShardKey)
	return chooser.Choose(ctx, req)
}

func (pl *sharder) newNotRunningError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "%s peer list is not running: %s", pl.name, err.Error())
}

func (pl *sharder) getOrCreateShardChooser(shard string) *shardChooser {
	if sc, ok := pl.getShardChooser(shard); ok {
		return sc
	}

	// Get failed, get the full lock and create a new sharder.
	pl.chooserLock.Lock()
	if sc, ok := pl.shardChoosers[shard]; ok {
		pl.chooserLock.Unlock()
		return sc
	}
	chooser := newShardChooser(chooserOpts{
		shardName:          shard,
		logger:             pl.logger,
		scope:              pl.scope,
		peerBlockType:      pl.peerBlockType,
		waitForPeerTimeout: pl.waitForPeerTimeout,
	})
	pl.shardChoosers[shard] = chooser
	pl.chooserLock.Unlock()
	return chooser
}

func (pl *sharder) getShardChooser(shard string) (_ *shardChooser, ok bool) {
	pl.chooserLock.RLock()
	sc, ok := pl.shardChoosers[shard]
	pl.chooserLock.RUnlock()
	return sc, ok
}
