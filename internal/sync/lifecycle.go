// Copyright (c) 2017 Uber Technologies, Inc.
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

package sync

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/atomic"
)

// LifecycleState represents `states` that a lifecycle object can be in.
type LifecycleState int

const (
	// Idle indicates the Lifecycle hasn't been operated on yet.
	Idle LifecycleState = iota

	// Starting indicates that the Lifecycle has begun it's "start" command
	// but hasn't finished yet.
	Starting

	// Running indicates that the Lifecycle has finished starting and is
	// available.
	Running

	// Stopping indicates that the Lifecycle 'stop' method has been called
	// but hasn't finished yet.
	Stopping

	// Stopped indicates that the Lifecycle has been stopped.
	Stopped

	// Errored indicates that the Lifecycle experienced an error and we can't
	// reasonably determine what state the lifecycle is in.
	Errored
)

// ErrAlreadyStopped is an error that indicates that WaitForStart returned
// early because the component has already stopped.
var ErrAlreadyStopped = errors.New("component has stopped")

// LifecycleOnce is a helper for implementing transport.Lifecycles
// with similar behavior.
type LifecycleOnce struct {
	// We use the sync.Mutex in order to guarantee that multiple calls
	// to Start/Stop will wait until the first call finishes and return
	// the same error (stored in startErr/stopErr).  This is not possible
	// with atomics because atomics will not block and we want the same error
	// to be returned for multiple calls to Start/Stop.
	//
	// State is stored in an atomic so that we can read it from the `IsRunning`
	// function without having to worry about race conditions with Start/Stop.
	// A RWMutex is not used because we don't want IsRunning to wait until
	// Start/Stop are finished.
	lock     sync.Mutex
	state    atomic.Int32
	startErr error
	stopErr  error
	started  chan struct{}
}

// WaitForStart blocks until the lifecycle has started, or the context expires.
// Returns context.DeadlineExceeded if the context expires first.
// Returns sync.AlreadyStopped if the lifecycle has already stopped.
func (l *LifecycleOnce) WaitForStart(ctx context.Context) error {
	state := LifecycleState(l.state.Load())
	// Fast success in the common case
	if state == Running {
		return nil
	}
	// Fast fail if we ran already
	if state > Running {
		return ErrAlreadyStopped
	}

	// Create a started channel if none exists. Do so in a lock so we're not
	// racing Start to set up for blocking.
	l.lock.Lock()
	if l.started == nil {
		l.started = make(chan struct{}, 0)
	}
	l.lock.Unlock()

	select {
	case <-l.started:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start will run the `f` function once and return the error.
// If Start is called multiple times it will return the error
// from the first time it was called.
func (l *LifecycleOnce) Start(f func() error) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	// If we've already moved on from the idle state we've either
	// called the start function already, or called the stop function
	// in which case we should exit now and return the result of the
	// last start command (or nil).
	if state := LifecycleState(l.state.Load()); state != Idle {
		return l.startErr
	}

	l.state.Store(int32(Starting))
	if f != nil {
		l.startErr = f()
	}
	if l.startErr == nil {
		l.state.Store(int32(Running))
	} else {
		l.state.Store(int32(Errored))
	}

	// Unblock WaitForStart callers, if there are any.
	if l.started != nil {
		close(l.started)
	}

	return l.startErr
}

// Stop will run the `f` function once and return the error.
// If Stop is called multiple times it will return the error
// from the first time it was called.
func (l *LifecycleOnce) Stop(f func() error) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	// If the lifecycle is already stopped or errored, don't execute the stop func
	if LifecycleState(l.state.Load()) == Stopped || LifecycleState(l.state.Load()) == Errored {
		return l.stopErr
	}

	// Unblock WaitForStart callers (ccounting Stop() without Start())
	if l.started != nil {
		close(l.started)
	}

	l.state.Store(int32(Stopping))
	if f != nil {
		l.stopErr = f()
	}
	if l.stopErr == nil {
		l.state.Store(int32(Stopped))
	} else {
		l.state.Store(int32(Errored))
	}

	return l.stopErr
}

// IsRunning will return true if current state of the Lifecycle is running
func (l *LifecycleOnce) IsRunning() bool {
	state := LifecycleState(l.state.Load())
	return state == Starting ||
		state == Running ||
		state == Stopping
}
