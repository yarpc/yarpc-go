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

	"go.uber.org/atomic"
)

var errDeadlineRequired = errors.New("deadline required")

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
type LifecycleOnce interface {
	Start(func() error) error
	Stop(func() error) error
	LifecycleState() LifecycleState
	IsRunning() bool
	WhenRunning(context.Context) error
}

type lifecycleOnce struct {
	// startCh closes to allow goroutines to resume after the lifecycle is in
	// the Running state or beyond.
	startCh chan struct{}
	// stopCh closes to allow goroutines to resume after the lifecycle is in
	// the Stopped state or Errored.
	stopCh chan struct{}
	// err is the error, if any, that Start() or Stop() returned and all
	// subsequent Start() or Stop() calls will return. The right to set
	// err is conferred to whichever goroutine is starting or stopping, until
	// it has started or stopped, after which `err` becomes immutable.
	err error
	// starting indicates that the lifecycle is at least starting.
	// Losing the race to set "starting" means that your goroutine must block
	// on "startCh".
	starting atomic.Bool
	// started indicates that the lifecycle is at least running.
	// Winning the race to set "started" means that the goroutine must close
	// "startCh".
	started atomic.Bool
	// stopping indicates that the lifecycle is at least stopping.
	// Losing the race to set "stopping" means that the goroutine must block on
	// "stopCh".
	stopping atomic.Bool
	// stopped indicates that the lifecycle is at least stopped.
	// Winning the race to set "stopped" means that the goroutine must close
	// "stopCh".
	stopped atomic.Bool
	// errored indicates that the lifecycle produced an error either while
	// starting or stopping.
	errored atomic.Bool
}

// Once returns a lifecycle controller.
// 0. The observable lifecycle state must only go forward from birth to death.
// 1. Start() must block until the state is >= Running
// 2. Stop() must block until the state is >= Stopped
// 3. Stop() must pre-empt Start() if it occurs first
// 4. Start() and Stop() may be backed by a do actual work function, and that
//    function must be called at most once.
func Once() LifecycleOnce {
	return &lifecycleOnce{
		startCh: make(chan struct{}, 0),
		stopCh:  make(chan struct{}, 0),
	}
}

// Start will run the `f` function once and return the error.
// If Start is called multiple times it will return the error
// from the first time it was called.
func (l *lifecycleOnce) Start(f func() error) error {
	if l.starting.Swap(true) {
		<-l.startCh
		return l.err
	}
	if f != nil {
		l.err = f()
	}
	// skip forward to error state
	if l.err != nil {
		l.errored.Store(true)
		l.stopped.Store(true)
		l.stopping.Store(true)
		close(l.stopCh)
	}
	l.started.Store(true)
	close(l.startCh)

	return l.err
}

func (l *lifecycleOnce) WhenRunning(ctx context.Context) error {
	if !l.stopping.Load() && l.started.Load() {
		return nil
	}

	if _, ok := ctx.Deadline(); !ok {
		return errDeadlineRequired
	}

	select {
	case <-l.startCh:
		return nil
	case <-l.stopCh:
		return context.DeadlineExceeded
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop will run the `f` function once and return the error.
// If Stop is called multiple times it will return the error
// from the first time it was called.
func (l *lifecycleOnce) Stop(f func() error) error {
	if l.stopping.Swap(true) {
		<-l.stopCh
		return l.err
	}

	if !l.starting.Swap(true) {
		// Was not already starting:
		// Pre-empt start
		l.started.Store(true)
		close(l.startCh)
	} else if l.started.Swap(true) {
		// Starting, but not yet started:
		// Wait for concurrent start to finish
		<-l.startCh
	}

	if f != nil {
		l.err = f()
	}
	l.errored.Store(l.err != nil)
	l.stopped.Store(true)
	close(l.stopCh)

	return l.err
}

// LifecycleState returns the state of the object within its life cycle, from
// start to full stop.
// The function only guarantees that the lifecycle has at least passed through
// the returned state and may have progressed further in the intervening time.
func (l *lifecycleOnce) LifecycleState() LifecycleState {
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
func (l *lifecycleOnce) IsRunning() bool {
	return l.LifecycleState() == Running
}
