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
	syncatomic "sync/atomic"

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
	Started() <-chan struct{}
	Stopping() <-chan struct{}
	Stopped() <-chan struct{}
}

type lifecycleOnce struct {
	// startCh closes to allow goroutines to resume after the lifecycle is in
	// the Running state or beyond.
	startCh chan struct{}
	// stoppingCh closes to allow goroutines to resume after the lifecycle is
	// in the Stopping state or beyond.
	stoppingCh chan struct{}
	// stopCh closes to allow goroutines to resume after the lifecycle is in
	// the Stopped state or Errored.
	stopCh chan struct{}
	// err is the error, if any, that Start() or Stop() returned and all
	// subsequent Start() or Stop() calls will return. The right to set
	// err is conferred to whichever goroutine is starting or stopping, until
	// it has started or stopped, after which `err` becomes immutable.
	err syncatomic.Value
	// state is an atomic LifecycleState representing the object's current
	// state (Idle, Starting, Running, Stopping, Stopped, Errored).
	state atomic.Int32
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
		startCh:    make(chan struct{}, 0),
		stoppingCh: make(chan struct{}, 0),
		stopCh:     make(chan struct{}, 0),
	}
}

// Start will run the `f` function once and return the error.
// If Start is called multiple times it will return the error
// from the first time it was called.
func (l *lifecycleOnce) Start(f func() error) error {
	if l.state.CAS(int32(Idle), int32(Starting)) {
		var err error
		if f != nil {
			err = f()
		}

		// skip forward to error state
		if err != nil {
			l.setError(err)
			l.state.Store(int32(Errored))
			close(l.stoppingCh)
			close(l.stopCh)
		} else {
			l.state.Store(int32(Running))
		}
		close(l.startCh)

		return err
	}

	<-l.startCh
	return l.loadError()
}

func (l *lifecycleOnce) WhenRunning(ctx context.Context) error {
	state := LifecycleState(l.state.Load())
	if state == Running {
		return nil
	}
	if state > Running {
		return context.DeadlineExceeded
	}

	if _, ok := ctx.Deadline(); !ok {
		return errDeadlineRequired
	}

	select {
	case <-l.startCh:
		state := LifecycleState(l.state.Load())
		if state == Running {
			return nil
		}
		return context.DeadlineExceeded
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop will run the `f` function once and return the error.
// If Stop is called multiple times it will return the error
// from the first time it was called.
func (l *lifecycleOnce) Stop(f func() error) error {
	if l.state.CAS(int32(Idle), int32(Stopped)) {
		close(l.startCh)
		close(l.stopCh)
		return nil
	}

	<-l.startCh

	if l.state.CAS(int32(Running), int32(Stopping)) {
		close(l.stoppingCh)

		var err error
		if f != nil {
			err = f()
		}

		if err != nil {
			l.setError(err)
			l.state.Store(int32(Errored))
		} else {
			l.state.Store(int32(Stopped))
		}
		close(l.stopCh)
		return err
	}

	<-l.stopCh
	return l.loadError()
}

// Started returns a channel that will close when the lifecycle starts.
func (l *lifecycleOnce) Started() <-chan struct{} {
	return l.startCh
}

// Stopping returns a channel that will close when the lifecycle is stopping.
func (l *lifecycleOnce) Stopping() <-chan struct{} {
	return l.stoppingCh
}

// Stopped returns a channel that will close when the lifecycle stops.
func (l *lifecycleOnce) Stopped() <-chan struct{} {
	return l.stopCh
}

func (l *lifecycleOnce) setError(err error) {
	l.err.Store(err)
}

func (l *lifecycleOnce) loadError() error {
	errVal := l.err.Load()
	if errVal == nil {
		return nil
	}

	if err, ok := errVal.(error); ok {
		return err
	}

	// TODO replace with DPanic log
	return errors.New("lifecycle err was not `error` type")
}

// LifecycleState returns the state of the object within its life cycle, from
// start to full stop.
// The function only guarantees that the lifecycle has at least passed through
// the returned state and may have progressed further in the intervening time.
func (l *lifecycleOnce) LifecycleState() LifecycleState {
	return LifecycleState(l.state.Load())
}

// IsRunning will return true if current state of the Lifecycle is running
func (l *lifecycleOnce) IsRunning() bool {
	return l.LifecycleState() == Running
}
