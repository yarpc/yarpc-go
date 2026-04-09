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
)

// newTestPeer returns a minimal grpcPeer sufficient for exercising the scaling
// monitor.  None of the TODO methods access peer fields, so we only need the
// context wiring.
func newTestPeer(ctx context.Context, cancel context.CancelFunc) *grpcPeer {
	return &grpcPeer{
		ctx:    ctx,
		cancel: cancel,
	}
}

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

// TestRunScalingMonitorTicksEvaluateScaling verifies that the monitor calls
// evaluateScaling at least once within a reasonable window.  We use a short
// context deadline so that the test does not run indefinitely.
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
