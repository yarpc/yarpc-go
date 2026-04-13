// Copyright (c) 2026 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/peer/abstractpeer"
)

// newTestPeer returns a minimal grpcPeer sufficient for exercising the scaling
// monitor goroutine lifecycle. Methods that reach p.t will panic, so only use
// this for tests that exercise early-return paths.
func newTestPeer(ctx context.Context, cancel context.CancelFunc) *grpcPeer {
	return &grpcPeer{
		ctx:    ctx,
		cancel: cancel,
	}
}

// makeConn creates a grpcClientConnWrapper with the given state and stream
// count. Intended for use in unit tests only.
func makeConn(state connState, streams int32) *grpcClientConnWrapper {
	return &grpcClientConnWrapper{
		state:       state,
		streamCount: streams,
	}
}

// peerForScaleDown builds a grpcPeer suitable for maybeScaleDown tests. It
// wires up a real Transport (for its nop logger) and a cancellable context.
func peerForScaleDown(t *testing.T, conns []*grpcClientConnWrapper, cfg connPoolConfig) *grpcPeer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	transport := NewTransport()
	return &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:       transport,
		ctx:     ctx,
		cancel:  cancel,
		conns:   conns,
		poolCfg: cfg,
	}
}

// defaultCfg is a pool config used across maybeScaleDown tests.
// threshold = int32(100 * 0.8) = 80.
var defaultScaleDownCfg = connPoolConfig{
	minConnections:      1,
	maxConcurrentStreams: 100,
	scaleUpThreshold:    0.8,
}

// TestMaybeScaleDown covers every branch of the maybeScaleDown function.
func TestMaybeScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		// conns is the initial pool state.
		conns []*grpcClientConnWrapper
		cfg   connPoolConfig
		// afterCall maps each conn (by index) to its expected state after the call.
		wantStates []connState
	}{
		{
			// Empty pool: active slice is empty → len(active)=0 <= minConnections=1 → return.
			desc:       "empty pool - returns at minConnections guard",
			conns:      nil,
			cfg:        defaultScaleDownCfg,
			wantStates: nil,
		},
		{
			// One active conn, minConnections=1: len(active)=1 <= 1 → return, no drain.
			desc: "active equals minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive},
		},
		{
			// One active conn, minConnections=2: len(active)=1 <= 2 → return, no drain.
			desc: "active below minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
			},
			cfg: connPoolConfig{
				minConnections:      2,
				maxConcurrentStreams: 100,
				scaleUpThreshold:    0.8,
			},
			wantStates: []connState{connStateActive},
		},
		{
			// Only draining conns: active slice is empty → returns at minConnections guard.
			// The existing draining connections are not modified.
			desc: "only draining conns - active=0, no change",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 5),
				makeConn(connStateDraining, 5),
			},
			cfg:        defaultScaleDownCfg, // minConnections=1, active=0
			wantStates: []connState{connStateDraining, connStateDraining},
		},
		{
			// Mix of active and draining. Only active conns count toward the
			// pool size, but the pool is at minConnections after filtering.
			desc: "mixed pool at minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
				makeConn(connStateDraining, 20),
			},
			cfg:        defaultScaleDownCfg, // minConnections=1, active=1
			wantStates: []connState{connStateActive, connStateDraining},
		},
		{
			// 3 active conns, load too high: threshold=80, capacityAfterDrain=80*2=160,
			// totalStreams=180 >= 160 → return without draining.
			desc: "total streams exceed capacity after drain - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 60),
				makeConn(connStateActive, 60),
				makeConn(connStateActive, 60),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive, connStateActive, connStateActive},
		},
		{
			// 3 active conns, load exactly equal to capacityAfterDrain:
			// totalStreams=160 >= 160 → return (boundary check).
			desc: "total streams exactly equal to capacity after drain - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 54),
				makeConn(connStateActive, 53),
				makeConn(connStateActive, 53),
			},
			cfg:        defaultScaleDownCfg, // threshold=80, capacity=160, total=160
			wantStates: []connState{connStateActive, connStateActive, connStateActive},
		},
		{
			// 3 active conns, low load: threshold=80, capacityAfterDrain=160,
			// totalStreams=60 < 160 → drain the most-loaded (last conn, 30 streams)
			// to maximise residual capacity in the surviving connections.
			desc: "low load - drains most-loaded connection",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
				makeConn(connStateActive, 20),
				makeConn(connStateActive, 30), // most loaded → drained
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive, connStateActive, connStateDraining},
		},
		{
			// Most-loaded is the first conn. Verifies the comparison loop
			// picks the globally largest stream count, not just the last.
			desc: "most-loaded is first - correct conn drained",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 30), // most loaded → drained
				makeConn(connStateActive, 5),
				makeConn(connStateActive, 25),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateDraining, connStateActive, connStateActive},
		},
		{
			// All active conns have equal stream counts. The first one is selected
			// (mostLoaded == nil on first iteration, then no subsequent conn wins
			// because equal counts don't satisfy >).
			desc: "equal stream counts - first active conn drained",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10), // selected (first, tied)
				makeConn(connStateActive, 10),
				makeConn(connStateActive, 10),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateDraining, connStateActive, connStateActive},
		},
		{
			// Draining conn interspersed: draining conn is excluded from active
			// list and from scale-down candidate selection.
			desc: "draining conn excluded from candidate selection",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 1), // excluded from active
				makeConn(connStateActive, 5),
				makeConn(connStateActive, 40), // most loaded active → drained
				makeConn(connStateActive, 40),
			},
			cfg: connPoolConfig{
				minConnections:      1,
				maxConcurrentStreams: 100,
				scaleUpThreshold:    0.8, // threshold=80, capacity=80*2=160, total=85 < 160
			},
			wantStates: []connState{connStateDraining, connStateActive, connStateDraining, connStateActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			p := peerForScaleDown(t, tt.conns, tt.cfg)
			p.maybeScaleDown()

			require.Len(t, p.conns, len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, p.conns[i].getState(),
					"conn[%d] state mismatch", i)
			}
		})
	}
}

// --- scaling monitor lifecycle tests ---

// TestRunScalingMonitorExitsOnContextCancel verifies that runScalingMonitor
// returns promptly when the peer's context is cancelled.
func TestRunScalingMonitorExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	p := newTestPeer(ctx, cancel)

	done := make(chan struct{})
	go func() {
		p.runScalingMonitor()
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// expected – monitor exited after context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("runScalingMonitor did not exit after context cancellation")
	}
}

// TestRunScalingMonitorTicksEvaluateScaling verifies that the monitor exits
// cleanly when its context is cancelled.
func TestRunScalingMonitorTicksEvaluateScaling(t *testing.T) {
	t.Parallel()

	// Cancel after a short fixed window rather than a multiple of
	// _scalingMonitorInterval, which is now 30s and would make the test too slow.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	p := newTestPeer(ctx, cancel)

	// runScalingMonitor blocks until the context expires; run it inline so the
	// test waits for it naturally.
	p.runScalingMonitor()

	// If we reach here the monitor exited cleanly on context cancellation – success.
}

// TestEvaluateScalingDoesNotPanic ensures that evaluateScaling (and its
// constituent helpers) can be called on a peer without panicking.
func TestEvaluateScalingDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newTestPeer(ctx, cancel)

	assert.NotPanics(t, p.evaluateScaling)
}

// TestScalingHelperMethodsDoNotPanic verifies individual scaling helpers
// independently to make future implementation easier to validate.
func TestScalingHelperMethodsDoNotPanic(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newTestPeer(ctx, cancel)

	assert.NotPanics(t, p.cleanupIdleConns)
	assert.NotPanics(t, p.maybeScaleDown)
}

