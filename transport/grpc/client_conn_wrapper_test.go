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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newTestConnWrapper creates a grpcClientConnWrapper backed by a real (but
// non-connected) *grpc.ClientConn suitable for unit tests.
func newTestConnWrapper(t *testing.T) *grpcClientConnWrapper {
	t.Helper()
	cc, err := grpc.NewClient("passthrough:///localhost:0", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cc.Close() })
	return newConnWrapper(context.Background(), cc)
}

// TestNewConnWrapper verifies that newConnWrapper initialises all fields
// correctly.
func TestNewConnWrapper(t *testing.T) {
	before := time.Now()
	w := newTestConnWrapper(t)
	after := time.Now()

	assert.NotNil(t, w.clientConn)
	assert.NotNil(t, w.ctx)
	assert.NotNil(t, w.cancel)
	assert.NotNil(t, w.stoppedC)
	assert.Equal(t, connStateActive, w.getState())
	assert.Equal(t, int32(0), w.getStreamCount())
	assert.False(t, w.createdAt.Before(before), "createdAt should be >= before")
	assert.False(t, w.createdAt.After(after), "createdAt should be <= after")
}

// TestConnStateConstants verifies the iota ordering that the rest of the pool
// logic depends on.
func TestConnStateConstants(t *testing.T) {
	assert.Equal(t, connState(0), connStateActive)
	assert.Equal(t, connState(1), connStateDraining)
	assert.Equal(t, connState(2), connStateIdle)
}

// TestGetSetState verifies that setState and getState round-trip correctly for
// all defined states.
func TestGetSetState(t *testing.T) {
	w := newTestConnWrapper(t)

	for _, s := range []connState{connStateActive, connStateDraining, connStateIdle} {
		w.setState(s)
		assert.Equal(t, s, w.getState())
	}
}

// TestIsActive verifies isActive returns true only when state is connStateActive.
func TestIsActive(t *testing.T) {
	w := newTestConnWrapper(t)

	assert.True(t, w.isActive(), "newly created wrapper should be active")

	w.setState(connStateDraining)
	assert.False(t, w.isActive())

	w.setState(connStateIdle)
	assert.False(t, w.isActive())

	w.setState(connStateActive)
	assert.True(t, w.isActive())
}

// TestStreamCountIncDec verifies incStreamCount, decStreamCount, and
// getStreamCount.
func TestStreamCountIncDec(t *testing.T) {
	w := newTestConnWrapper(t)

	assert.Equal(t, int32(0), w.getStreamCount())

	w.incStreamCount()
	assert.Equal(t, int32(1), w.getStreamCount())

	w.incStreamCount()
	w.incStreamCount()
	assert.Equal(t, int32(3), w.getStreamCount())

	w.decStreamCount()
	assert.Equal(t, int32(2), w.getStreamCount())

	w.decStreamCount()
	w.decStreamCount()
	assert.Equal(t, int32(0), w.getStreamCount())
}

// TestStreamCountConcurrent verifies that inc/dec are safe under concurrent
// access.
func TestStreamCountConcurrent(t *testing.T) {
	w := newTestConnWrapper(t)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			w.incStreamCount()
		}()
		go func() {
			defer wg.Done()
			w.decStreamCount()
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(0), w.getStreamCount())
}

// TestSetIdleNow verifies that setIdleNow records a timestamp and idleSince
// returns it.
func TestSetIdleNow(t *testing.T) {
	w := newTestConnWrapper(t)

	// Before setIdleNow, idleSince should return the zero time.
	assert.True(t, w.idleSince().IsZero(), "idleSince should be zero before setIdleNow")

	before := time.Now()
	w.setIdleNow()
	after := time.Now()

	idle := w.idleSince()
	assert.False(t, idle.IsZero())
	assert.False(t, idle.Before(before.Truncate(time.Nanosecond)), "idleSince should be >= before")
	assert.False(t, idle.After(after), "idleSince should be <= after")
}

// TestIdleSinceZeroBeforeSet verifies the zero-value branch in idleSince.
func TestIdleSinceZeroBeforeSet(t *testing.T) {
	w := newTestConnWrapper(t)
	assert.Equal(t, time.Time{}, w.idleSince())
}

// TestContextCancelledOnCancel verifies that the context derived inside
// newConnWrapper is cancelled when cancel() is called.
func TestContextCancelledOnCancel(t *testing.T) {
	w := newTestConnWrapper(t)

	require.NoError(t, w.ctx.Err(), "context should not be cancelled initially")

	w.cancel()

	assert.ErrorIs(t, w.ctx.Err(), context.Canceled)
}

// TestContextInheritsParentCancellation verifies that cancelling the parent
// context also cancels the wrapper's context.
func TestContextInheritsParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())

	cc, err := grpc.NewClient("passthrough:///localhost:0", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cc.Close() })

	w := newConnWrapper(parent, cc)

	require.NoError(t, w.ctx.Err())
	cancel()
	assert.ErrorIs(t, w.ctx.Err(), context.Canceled)
}
