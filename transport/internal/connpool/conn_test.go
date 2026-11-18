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
	"time"

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

// --- ported from the former transport/grpc client_conn_wrapper tests ---

func TestWrapper_NewWrapperInitialisesFields(t *testing.T) {
	conn := &fakeConn{}
	before := time.Now()
	w := newWrapper(context.Background(), conn)
	after := time.Now()

	assert.Same(t, conn, w.Conn)
	assert.NotNil(t, w.ctx)
	assert.NotNil(t, w.cancel)
	assert.NotNil(t, w.StoppedC)
	assert.False(t, w.createdAt.Before(before), "createdAt should be >= before")
	assert.False(t, w.createdAt.After(after), "createdAt should be <= after")
}

func TestState_ConstantsKeepIotaOrdering(t *testing.T) {
	assert.Equal(t, State(0), StateActive)
	assert.Equal(t, State(1), StateDraining)
	assert.Equal(t, State(2), StateIdle)
	assert.Equal(t, State(3), StateClosing)
}

func TestWrapper_GetSetStateRoundTrips(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	for _, s := range []State{StateActive, StateDraining, StateIdle, StateClosing} {
		w.setState(s)
		assert.Equal(t, s, w.GetState())
	}
}

func TestWrapper_IsActiveOnlyWhenActive(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})
	assert.True(t, w.IsActive(), "newly created wrapper should be active")

	for _, s := range []State{StateDraining, StateIdle, StateClosing} {
		w.setState(s)
		assert.False(t, w.IsActive(), "state %d", s)
	}
	w.setState(StateActive)
	assert.True(t, w.IsActive())
}

func TestWrapper_StreamCountConcurrentIncAndDecBalance(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for range goroutines {
		go func() {
			defer wg.Done()
			w.IncStreamCount()
		}()
		go func() {
			defer wg.Done()
			w.DecStreamCount()
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 0, w.StreamCount())
}

func TestWrapper_SetIdleNowRecordsCurrentTime(t *testing.T) {
	w := newWrapper(context.Background(), &fakeConn{})

	before := time.Now()
	w.setIdleNow()
	after := time.Now()

	idle := w.IdleSince()
	assert.False(t, idle.IsZero())
	assert.False(t, idle.Before(before.Truncate(time.Nanosecond)), "IdleSince should be >= before")
	assert.False(t, idle.After(after), "IdleSince should be <= after")
}

func TestWrapper_TransitionStateConcurrentSingleWinner(t *testing.T) {
	for range 100 {
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

		assert.EqualValues(t, 1, atomic.LoadInt32(&wins), "exactly one transition must win")
	}
}
