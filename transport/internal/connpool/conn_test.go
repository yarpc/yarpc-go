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

package connpool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapper_NewWrapperDefaultsToActive(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	assert.Equal(t, StateActive, w.GetState())
	assert.True(t, w.IsActive())
	assert.Zero(t, w.StreamCount())
	assert.True(t, w.IdleSince().IsZero())
}

func TestWrapper_StreamCountIncDec(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	w.IncStreamCount()
	w.IncStreamCount()
	assert.EqualValues(t, 2, w.StreamCount())
	w.DecStreamCount()
	assert.EqualValues(t, 1, w.StreamCount())
}

func TestWrapper_StreamCountConcurrent(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.IncStreamCount()
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 50, w.StreamCount())
}

func TestWrapper_TransitionState(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	require.True(t, w.TransitionState(StateActive, StateDraining))
	assert.Equal(t, StateDraining, w.GetState())
	assert.False(t, w.IsActive())

	// Wrong `from` fails and leaves state unchanged.
	assert.False(t, w.TransitionState(StateActive, StateIdle))
	assert.Equal(t, StateDraining, w.GetState())
}

// TestWrapper_ConcurrentTransitionOnlyOneWins ports transport/grpc's
// concurrent-CAS subtest of TestTransitionState: when two goroutines race to
// transition the same wrapper out of the same starting state to two
// different destination states, exactly one of them must win.
func TestWrapper_ConcurrentTransitionOnlyOneWins(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	w.setState(StateIdle)

	var wins int32
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if w.TransitionState(StateIdle, StateActive) {
			atomic.AddInt32(&wins, 1)
		}
	}()
	go func() {
		defer wg.Done()
		if w.TransitionState(StateIdle, StateClosing) {
			atomic.AddInt32(&wins, 1)
		}
	}()
	wg.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&wins), "exactly one transition must win")
}

func TestWrapper_SetIdleNowAndIdleSince(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	assert.True(t, w.IdleSince().IsZero())
	w.setState(StateIdle)
	w.setIdleNow()
	assert.False(t, w.IdleSince().IsZero())
}

func TestWrapper_ContextCancelledOnCancel(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	select {
	case <-w.Context().Done():
		t.Fatal("context should not be done yet")
	default:
	}
	w.Cancel()
	select {
	case <-w.Context().Done():
	default:
		t.Fatal("context should be done after Cancel")
	}
}

func TestWrapper_ContextInheritsParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	w := newWrapper(parent, &fakeConn{})
	cancel()
	select {
	case <-w.Context().Done():
	default:
		t.Fatal("wrapper context should be cancelled when parent is cancelled")
	}
}
