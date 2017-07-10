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

package lifecycle

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/atomic"
	"go.uber.org/yarpc/internal/testtime"
)

type wrappedLifecycleOnce struct {
	LifecycleOnce
	running *atomic.Bool
}

// LifecycleAction defines actions that can be applied to a Lifecycle
type LifecycleAction interface {
	// Apply runs a function on the PeerList and asserts the result
	Apply(*testing.T, wrappedLifecycleOnce)
}

// StartAction is an action for testing LifecycleOnce.Start
type StartAction struct {
	Wait          time.Duration
	Err           error
	ExpectedErr   error
	ExpectedState LifecycleState
}

// Apply runs "Start" on the LifecycleOnce and validates the error
func (a StartAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	err := l.Start(func() error {
		assert.False(t, l.running.Swap(true), "expected no other running action")
		if a.Wait > 0 {
			testtime.Sleep(a.Wait)
		}
		assert.True(t, l.running.Swap(false), "expected no other running action")
		return a.Err
	})
	assert.Equal(t, a.ExpectedErr, err)
	state := l.LifecycleState()
	assert.True(t, a.ExpectedState <= state, "expected %v (or more advanced), got %v after start", a.ExpectedState, state)
}

// StopAction is an action for testing LifecycleOnce.Stop
type StopAction struct {
	Wait          time.Duration
	Err           error
	ExpectedErr   error
	ExpectedState LifecycleState
}

// Apply runs "Stop" on the LifecycleOnce and validates the error
func (a StopAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	err := l.Stop(func() error {
		assert.False(t, l.running.Swap(true), "expected no other running action")
		if a.Wait > 0 {
			testtime.Sleep(a.Wait)
		}
		assert.True(t, l.running.Swap(false), "expected no other running action")
		return a.Err
	})

	assert.Equal(t, a.ExpectedErr, err)
	assert.Equal(t, a.ExpectedState, l.LifecycleState())
}

// WaitForStartAction is a singleton that will block until the lifecycle
// reports that it has started.
var WaitForStartAction waitForStartAction

type waitForStartAction struct{}

// Apply blocks until the lifecycle starts.
func (a waitForStartAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	<-l.Started()
	assert.True(t, l.LifecycleState() >= Running, "expected lifecycle to be started")
}

// WaitForStoppingAction is a singleton that will block until the lifecycle
// reports that it has begun stopping.
var WaitForStoppingAction waitForStoppingAction

type waitForStoppingAction struct{}

// Apply blocks until the lifecycle stops or errs out.
func (a waitForStoppingAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	<-l.Stopping()
	assert.True(t, l.LifecycleState() >= Stopping, "expected lifecycle to be stopping")
}

// WaitForStopAction is a singleton that will block until the lifecycle
// reports that it has started.
var WaitForStopAction waitForStopAction

type waitForStopAction struct{}

// Apply blocks until the lifecycle stops or errs out.
func (a waitForStopAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	<-l.Stopped()
	assert.True(t, l.LifecycleState() >= Stopped, "expected lifecycle to be started")
}

// GetStateAction is an action for checking the LifecycleOnce's state.
// Since a goroutine may be delayed, the action only ensures that the lifecycle
// has at least reached the given state.
type GetStateAction struct {
	ExpectedState LifecycleState
}

// Apply Checks the state on the LifecycleOnce
func (a GetStateAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	assert.True(t, a.ExpectedState <= l.LifecycleState())
}

// ExactStateAction is an action for checking the LifecycleOnce's exact state.
type ExactStateAction struct {
	ExpectedState LifecycleState
}

// Apply Checks the state on the LifecycleOnce
func (a ExactStateAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	assert.True(t, a.ExpectedState == l.LifecycleState())
}

// Actions executes a plan of actions in order sequentially.
type Actions []LifecycleAction

// Apply runs all of the ConcurrentAction's actions sequentially.
func (a Actions) Apply(t *testing.T, l wrappedLifecycleOnce) {
	for _, action := range a {
		action.Apply(t, l)
	}
}

// ConcurrentAction executes a plan of actions, with a given interval between
// applying actions, but allowing every action to run concurrently in a
// goroutine until its independent completion time.
// The ConcurrentAction allows us to observe overlapping actions.
type ConcurrentAction struct {
	Actions []LifecycleAction
	Wait    time.Duration
}

// Apply runs all the ConcurrentAction's actions in goroutines with a delay of `Wait`
// between each action. Returns when all actions have finished executing
func (a ConcurrentAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	var wg sync.WaitGroup

	wg.Add(len(a.Actions))
	for _, action := range a.Actions {
		go func(ac LifecycleAction) {
			defer wg.Done()
			ac.Apply(t, l)
		}(action)

		if a.Wait > 0 {
			testtime.Sleep(a.Wait)
		}
	}

	wg.Wait()
}

// WaitAction is a plan to sleep for a duration.
type WaitAction time.Duration

// Apply waits the specified duration.
func (a WaitAction) Apply(t *testing.T, l wrappedLifecycleOnce) {
	testtime.Sleep(time.Duration(a))
}

// ApplyLifecycleActions runs all the LifecycleActions on the LifecycleOnce
func ApplyLifecycleActions(t *testing.T, l LifecycleOnce, actions []LifecycleAction) {
	wrapLife := wrappedLifecycleOnce{
		LifecycleOnce: l,
		running:       atomic.NewBool(false),
	}

	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, wrapLife)
		})
	}
}
