package shardproxy

import (
	"context"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/shardproxy/internal/gen/go/github.com/uber/tchannel/meta/metaclient"
	"go.uber.org/yarpc/x/shardproxy/internal/gen/go/go.uber.org/yarpc/sharder/shardclient"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

// CallablePeer is a peer that can be called.
type callablePeer interface {
	peer.Peer

	Call(ctx context.Context, request *transport.Request) (*transport.Response, error)
}

type outboundPeer struct {
	callablePeer
}

// Start implements UnaryOutbound
func (o outboundPeer) Start() error { return nil }

// Stop implements UnaryOutbound
func (o outboundPeer) Stop() error { return nil }

// IsRunning implements UnaryOutbound
func (o outboundPeer) IsRunning() bool { return false }

// Transports implements UnaryOutbound
func (o outboundPeer) Transports() []transport.Transport { return nil }

// PeerStatus maintains information about the Peer's shard/health state
type PeerStatus int

const (
	// OutOfRotation indicates the Peer is not currently in Peer Lists
	OutOfRotation PeerStatus = iota

	// InRotation indicates the Peer is in one or more shard managers.
	InRotation
)

var _ transport.UnaryOutbound = (*peerThunk)(nil)

// peerThunk captures a peer and its corresponding subscriber,
// and serves as a subscriber by proxy.
type peerThunk struct {
	lock    sync.RWMutex
	sharder *sharder

	id peer.Identifier

	peerLock sync.RWMutex
	peer     outboundPeer

	peerBackoff backoff.Backoff
	rand        *rand.Rand

	// State
	shards map[string]struct{}
	status PeerStatus

	lastConnectionStatus atomic.Int32
	// the changed channel will be filled when there is a notification to the peer.
	changed chan struct{}
	// The removed channel will be closed when the peer is removed by the provider.
	removed chan struct{}
	timer   *time.Timer

	// Used to call the peer for control purposes.
	config       transport.OutboundConfig
	healthClient metaclient.Interface
	shardClient  shardclient.Interface

	boundOnFinish func(error)
}

func newThunk(sharder *sharder, identifier peer.Identifier) *peerThunk {
	timer := time.NewTimer(0)
	if !timer.Stop() { // Reset and drain the Timer (just in case)
		<-timer.C
	}

	t := &peerThunk{
		sharder:     sharder,
		id:          identifier,
		peerBackoff: sharder.peerBackoffStrategy.Backoff(),
		shards:      make(map[string]struct{}, 0),
		removed:     make(chan struct{}, 0),
		changed:     make(chan struct{}, 1),
		timer:       timer,
		status:      OutOfRotation,
		rand:        newRand(),
	}
	t.lastConnectionStatus.Store(int32(peer.Unavailable))
	t.boundOnFinish = t.onFinish
	t.config = transport.OutboundConfig{
		CallerName: sharder.serviceName,
		Outbounds: transport.Outbounds{
			ServiceName: sharder.peerServiceName,
			Unary:       t,
		},
	}
	t.healthClient = metaclient.New(&t.config)
	t.shardClient = shardclient.New(&t.config)
	return t
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (t *peerThunk) SetPeer(p peer.Peer) error {
	t.peerLock.Lock()
	if cp, ok := p.(callablePeer); ok {
		t.peer = outboundPeer{callablePeer: cp}
		t.peerLock.Unlock()
		return nil
	}
	t.peerLock.Unlock()
	return yarpcerrors.InternalErrorf("cannot use shard peer list with non-callable yarpc peer.Peer.")
}

// There is a single case (NotifyStatusChanged) where we need to validate that
// the peer exists, otherwise we're good.
func (t *peerThunk) getPeer() callablePeer {
	t.peerLock.RLock()
	peer := t.peer.callablePeer
	t.peerLock.RUnlock()
	return peer
}

func (t *peerThunk) onStart() {
	t.getPeer().StartRequest()
}

func (t *peerThunk) onFinish(err error) {
	if yarpcerrors.IsUnavailable(err) {
		t.onStatusChanged()
	}
	t.getPeer().EndRequest()
}

func (t *peerThunk) Identifier() string {
	return t.peer.Identifier()
}

func (t *peerThunk) Status() peer.Status {
	return t.peer.Status()
}

func (t *peerThunk) StartRequest() {
	t.peer.StartRequest()
}

func (t *peerThunk) EndRequest() {
	t.peer.EndRequest()
}

func (t *peerThunk) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if t.sharder.edgeMiddleware != nil {
		return t.sharder.edgeMiddleware.Call(ctx, request, t.peer)
	}
	return t.peer.Call(ctx, request)
}

// Start implements transport.UnaryOutbound.
func (t *peerThunk) Start() error {
	return nil
}

// Stop implements transport.UnaryOutbound.
func (t *peerThunk) Stop() error {
	return nil
}

// Transports implements transport.UnaryOutbound.
func (t *peerThunk) Transports() []transport.Transport {
	return nil
}

// IsRunning implements transport.UnaryOutbound.
func (t *peerThunk) IsRunning() bool {
	return true
}

// NotifyStatusChanged forwards a status notification to the peer list and to
// the underlying identifier chooser list.
func (t *peerThunk) NotifyStatusChanged(pid peer.Identifier) {
	if p := t.getPeer(); p != nil {
		// Action should only be taken if there is change.
		newStatus := int32(p.Status().ConnectionStatus)
		if t.lastConnectionStatus.Swap(newStatus) != newStatus {
			t.onStatusChanged()
		}
	}
}

func (t *peerThunk) onStatusChanged() {
	select {
	case t.changed <- struct{}{}:
	default:
	}
}

func (t *peerThunk) isClosed() bool {
	select {
	case <-t.removed:
		return true
	default:
	}
	return false
}

func (t *peerThunk) RemoveAndGetShards() map[string]struct{} {
	t.lock.Lock()
	if t.isClosed() { // Guarantee we only close the channel once.
		t.lock.Unlock()
		return nil
	}
	close(t.removed)

	shards := t.shards
	t.lock.Unlock()
	return shards
}

// MaintainConnection maintains the connection loop to the peer.
func (t *peerThunk) MaintainConnection() {
	cancel := func() {}
	defer func() {
		t.sharder.peerConnections.Dec()
		t.sharder.peerConnectionGroup.Done()
		cancel()
		if r := recover(); r != nil {
			t.sharder.logger.DPanic("peer connection maintainer panicked!", zap.Any("recover", r), zap.Any("debug", debug.Stack()))
		}
	}()

	var attempt uint
	shouldExit := false
	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	for {
		// If the peer has been "killed" (i.e. removed from uns path) then exit.
		select {
		case <-t.removed:
			shouldExit = true
		default:
		}
		if shouldExit {
			break
		}

		// If peer is not connected, Backoff.
		if t.peer.Status().ConnectionStatus != peer.Available {
			t.lock.Lock()
			if t.isClosed() {
				t.lock.Unlock()
				continue
			}
			t.updateRotation([]string{})
			t.lock.Unlock()

			t.backoff(attempt)
			attempt++
			continue
		}

		// If peer is connected, attempt to call health endpoint
		healthCtx, cancel := context.WithTimeout(context.Background(), t.sharder.healthReqTimeout)
		healthRes, err := t.healthClient.Health(healthCtx, &t.sharder.healthRequest)
		if err != nil {
			cancel()
			t.lock.Lock()
			if t.isClosed() {
				t.lock.Unlock()
				continue
			}
			t.updateRotation([]string{})
			t.lock.Unlock()

			t.backoff(attempt)
			attempt++
			continue
		}
		cancel()
		if !healthRes.Ok {
			t.lock.Lock()
			if t.isClosed() {
				t.lock.Unlock()
				continue
			}
			t.updateRotation([]string{})
			t.lock.Unlock()

			t.backoff(attempt)
			attempt++
			continue
		}

		// If the peer is healthy, call the shard endpoint.
		shardCtx, cancel := context.WithTimeout(context.Background(), t.sharder.shardReqTimeout)
		shardRes, err := t.shardClient.ShardInfo(shardCtx, &t.sharder.shardRequest)
		if err != nil {
			cancel()
			t.lock.Lock()
			if t.isClosed() {
				t.lock.Unlock()
				continue
			}
			t.updateRotation([]string{})
			t.lock.Unlock()

			t.backoff(attempt)
			attempt++
			continue
		}
		cancel()

		// If all this is successful update the shard mappings.
		t.lock.Lock()
		if t.isClosed() {
			t.lock.Unlock()
			continue
		}
		t.updateRotation(shardRes.SupportedShards)
		t.lock.Unlock()

		attempt = 0
		t.sleepUntilNextPeerCheck()
	}

}

// updateRotation updates the shard lists the thunk is connected to to match the
// new shard list.
// Must be run in a mutex.Lock
func (t *peerThunk) updateRotation(shards []string) {
	newShardMap := make(map[string]struct{}, len(shards))
	var addedShards []string
	var removedShards []string
	for _, shard := range shards {
		newShardMap[shard] = struct{}{}
		if _, ok := t.shards[shard]; !ok {
			addedShards = append(addedShards, shard)
		}
	}
	for shard := range t.shards {
		if _, ok := newShardMap[shard]; !ok {
			removedShards = append(removedShards, shard)
		}
	}
	if len(removedShards) > 0 || len(addedShards) > 0 {
		t.sharder.UpdateShards(
			t,
			shardUpdates{
				Additions: addedShards,
				Removals:  removedShards,
			},
		)
	}
	t.shards = newShardMap
	if len(t.shards) > 0 {
		t.status = InRotation
	} else {
		t.status = OutOfRotation
	}
}

func (t *peerThunk) sleepUntilNextPeerCheck() (completed bool) {
	sleepTime := time.Duration(t.rand.Int63n(t.sharder.maxPeerCheckInterval.Nanoseconds() - t.sharder.minPeerCheckInterval.Nanoseconds()))
	sleepTime += t.sharder.minPeerCheckInterval
	return t.sleep(sleepTime)
}

func (t *peerThunk) backoff(attempt uint) (completed bool) {
	return t.sleep(t.peerBackoff.Duration(attempt))
}

// sleep waits for a duration, but exits early if the transport releases the
// peer or stops. sleep returns whether it successfully waited the entire
// duration.
func (t *peerThunk) sleep(delay time.Duration) (completed bool) {
	t.timer.Reset(delay)

	select {
	case <-t.timer.C:
		return true
	case <-t.removed:
	case <-t.changed:
	}

	if !t.timer.Stop() {
		<-t.timer.C
	}
	return false
}
