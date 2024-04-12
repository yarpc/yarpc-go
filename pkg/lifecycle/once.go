// Copyright (c) 2024 Uber Technologies, Inc.
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

package lifecycle

import (
	"context"
	"errors"
	syncatomic "sync/atomic"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/yarpcerrors"
)

// State represents `states` that a lifecycle object can be in.
type State int

const (
	// Idle indicates the Lifecycle hasn't been operated on yet.
	Idle State = iota

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

var stateToName = map[State]string{
	Idle:     "idle",
	Starting: "starting",
	Running:  "running",
	Stopping: "stopping",
	Stopped:  "stopped",
	Errored:  "errored",
}

func getStateName(s State) string {
	if name, ok := stateToName[s]; ok {
		return name
	}
	return "unknown"
}

// Once is a helper for implementing objects that advance monotonically through
// lifecycle states using at-most-once start and stop implementations in a
// thread safe manner.
type Once struct {
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
	// state is an atomic State representing the object's current
	// state (Idle, Starting, Running, Stopping, Stopped, Errored).
	state atomic.Int32
}

// NewOnce returns a lifecycle controller.
//
//  0. The observable lifecycle state must only go forward from birth to death.
//  1. Start() must block until the state is >= Running
//  2. Stop() must block until the state is >= Stopped
//  3. Stop() must pre-empt Start() if it occurs first
//  4. Start() and Stop() may be backed by a do-actual-work function, and that
//     function must be called at-most-once.
func NewOnce() *Once {
	return &Once{
		startCh:    make(chan struct{}),
		stoppingCh: make(chan struct{}),
		stopCh:     make(chan struct{}),
	}
}

// Start will run the `f` function once and return the error.
// If Start is called multiple times it will return the error
// from the first time it was called.
func (o *Once) Start(f func() error) error {
	if o.state.CAS(int32(Idle), int32(Starting)) {
		var err error
		if f != nil {
			err = f()
		}

		// skip forward to error state
		if err != nil {
			o.setError(err)
			o.state.Store(int32(Errored))
			close(o.stoppingCh)
			close(o.stopCh)
		} else {
			o.state.Store(int32(Running))
		}
		close(o.startCh)

		return err
	}

	<-o.startCh
	return o.loadError()
}

// WaitUntilRunning blocks until the instance enters the running state, or the
// context times out.
func (o *Once) WaitUntilRunning(ctx context.Context) error {
	state := State(o.state.Load())
	if state == Running {
		return nil
	}
	if state > Running {
		return yarpcerrors.FailedPreconditionErrorf("could not wait for instance to start running: current state is %q", getStateName(state))
	}

	if _, ok := ctx.Deadline(); !ok {
		return yarpcerrors.InvalidArgumentErrorf("could not wait for instance to start running: deadline required on request context")
	}

	select {
	case <-o.startCh:
		state := State(o.state.Load())
		if state == Running {
			return nil
		}
		return yarpcerrors.FailedPreconditionErrorf("instance did not enter running state, current state is %q", getStateName(state))
	case <-ctx.Done():
		return yarpcerrors.FailedPreconditionErrorf("context finished while waiting for instance to start: %s", ctx.Err().Error())
	}
}

// Stop will run the `f` function once and return the error.
// If Stop is called multiple times it will return the error
// from the first time it was called.
func (o *Once) Stop(f func() error) error {
	if o.state.CAS(int32(Idle), int32(Stopped)) {
		close(o.startCh)
		close(o.stoppingCh)
		close(o.stopCh)
		return nil
	}

	<-o.startCh

	if o.state.CAS(int32(Running), int32(Stopping)) {
		close(o.stoppingCh)

		var err error
		if f != nil {
			err = f()
		}

		if err != nil {
			o.setError(err)
			o.state.Store(int32(Errored))
		} else {
			o.state.Store(int32(Stopped))
		}
		close(o.stopCh)
		return err
	}

	<-o.stopCh
	return o.loadError()
}

// Started returns a channel that will close when the lifecycle starts.
func (o *Once) Started() <-chan struct{} {
	return o.startCh
}

// Stopping returns a channel that will close when the lifecycle is stopping.
func (o *Once) Stopping() <-chan struct{} {
	return o.stoppingCh
}

// Stopped returns a channel that will close when the lifecycle stops.
func (o *Once) Stopped() <-chan struct{} {
	return o.stopCh
}

func (o *Once) setError(err error) {
	o.err.Store(err)
}

func (o *Once) loadError() error {
	errVal := o.err.Load()
	if errVal == nil {
		return nil
	}

	if err, ok := errVal.(error); ok {
		return err
	}

	// TODO replace with DPanic log
	return errors.New("lifecycle err was not `error` type")
}

// State returns the state of the object within its life cycle, from
// start to full stop.
// The function only guarantees that the lifecycle has at least passed through
// the returned state and may have progressed further in the intervening time.
func (o *Once) State() State {
	return State(o.state.Load())
}

// IsRunning will return true if current state of the Lifecycle is running
func (o *Once) IsRunning() bool {
	return o.State() == Running
}
