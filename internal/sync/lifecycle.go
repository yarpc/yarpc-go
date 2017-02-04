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

import "go.uber.org/atomic"

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

// LifecycleOnce is a helper for implementing transport.Lifecycles
// with similar behavior.
type LifecycleOnce struct {
	starting atomic.Bool
	started  atomic.Bool
	startCh  chan struct{}
	stopping atomic.Bool
	stopped  atomic.Bool
	stopCh   chan struct{}
	errored  atomic.Bool
	startErr error
	stopErr  error
}

// Once returns an initialized lifecycle.
func Once() LifecycleOnce {
	return LifecycleOnce{
		startCh: make(chan struct{}, 0),
		stopCh:  make(chan struct{}, 0),
	}
}

// Start will run the `f` function once and return the error.
// If Start is called multiple times it will return the error
// from the first time it was called.
func (l *LifecycleOnce) Start(f func() error) error {
	if l.starting.Swap(true) {
		<-l.startCh
		return l.startErr
	}
	if f != nil {
		l.startErr = f()
	}
	// skip forward to error state
	if l.startErr != nil {
		l.errored.Store(true)
		l.stopped.Store(true)
		l.stopping.Store(true)
		close(l.stopCh)
	}
	l.started.Store(true)
	close(l.startCh)

	return l.startErr
}

// Stop will run the `f` function once and return the error.
// If Stop is called multiple times it will return the error
// from the first time it was called.
func (l *LifecycleOnce) Stop(f func() error) error {
	if l.stopping.Swap(true) {
		<-l.stopCh
		return l.stopErr
	}

	// Pre-empt start
	if !l.starting.Swap(true) {
		l.started.Store(true)
		close(l.startCh)
	}

	if l.started.Swap(true) {
		// Wait for concurrent start to finish
		<-l.startCh
	}

	if f != nil {
		l.stopErr = f()
	}
	l.errored.Store(l.stopErr != nil)
	l.stopped.Store(true)
	close(l.stopCh)

	return l.stopErr
}

// LifecycleState returns the state of the object within its life cycle, from
// start to full stop.
// The function only guarantees that the lifecycle has at least passed through
// the returned state and may have progressed further in the intervening time.
func (l *LifecycleOnce) LifecycleState() LifecycleState {
	switch {
	case l.errored.Load():
		return Errored
	case l.stopped.Load():
		return Stopped
	case l.stopping.Load():
		return Stopping
	case l.started.Load():
		return Running
	case l.starting.Load():
		return Starting
	default:
		return Idle
	}
}

// IsRunning will return true if current state of the Lifecycle is running
func (l *LifecycleOnce) IsRunning() bool {
	return l.LifecycleState() == Running
}
