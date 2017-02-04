// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// LifecycleAction defines actions that can be applied to a Lifecycle
type LifecycleAction interface {
	// Apply runs a function on the PeerList and asserts the result
	Apply(*testing.T, LifecycleOnce)
}

// StartAction is an action for testing LifecycleOnce.Start
type StartAction struct {
	Wait          time.Duration
	Err           error
	ExpectedErr   error
	ExpectedState LifecycleState
}

// Apply runs "Start" on the LifecycleOnce and validates the error
func (a StartAction) Apply(t *testing.T, l LifecycleOnce) {
	err := l.Start(func() error {
		if a.Wait > 0 {
			time.Sleep(a.Wait)
		}
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
func (a StopAction) Apply(t *testing.T, l LifecycleOnce) {
	err := l.Stop(func() error {
		if a.Wait > 0 {
			time.Sleep(a.Wait)
		}
		return a.Err
	})

	assert.Equal(t, a.ExpectedErr, err)
	assert.Equal(t, a.ExpectedState, l.LifecycleState())
}

// GetStateAction is an action for checking the LifecycleOnce's state
type GetStateAction struct {
	ExpectedState LifecycleState
}

// Apply Checks the state on the LifecycleOnce
func (a GetStateAction) Apply(t *testing.T, l LifecycleOnce) {
	assert.Equal(t, a.ExpectedState, l.LifecycleState())
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
func (a ConcurrentAction) Apply(t *testing.T, l LifecycleOnce) {
	var wg sync.WaitGroup

	wg.Add(len(a.Actions))
	for _, action := range a.Actions {
		go func(ac LifecycleAction) {
			defer wg.Done()
			ac.Apply(t, l)
		}(action)

		if a.Wait > 0 {
			time.Sleep(a.Wait)
		}
	}

	wg.Wait()
}

// ApplyLifecycleActions runs all the LifecycleActions on the LifecycleOnce
func ApplyLifecycleActions(t *testing.T, l LifecycleOnce, actions []LifecycleAction) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, l)
		})
	}
}
