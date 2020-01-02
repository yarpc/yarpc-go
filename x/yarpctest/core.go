// Copyright (c) 2020 Uber Technologies, Inc.
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

package yarpctest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// Action is the interface for applying actions (Requests) in tests.
type Action api.Action

// Lifecycle is the interface for creating/starting/stopping lifecycles
// (Services) in tests.
type Lifecycle api.Lifecycle

// Lifecycles is a wrapper around a list of Lifecycle definitions.
func Lifecycles(l ...api.Lifecycle) api.Lifecycle {
	return lifecycles(l)
}

type lifecycles []api.Lifecycle

// Start the lifecycles. If there are any errors, stop any started lifecycles
// and fail the test.
func (ls lifecycles) Start(t testing.TB) error {
	startedLifecycles := make(lifecycles, 0, len(ls))
	for _, l := range ls {
		err := l.Start(t)
		if !assert.NoError(t, err) {
			// Cleanup started lifecycles (this could fail)
			return multierr.Append(err, startedLifecycles.Stop(t))
		}
		startedLifecycles = append(startedLifecycles, l)
	}
	return nil
}

// Stop the lifecycles. Record all errors. If any lifecycle failed to stop
// fail the test.
func (ls lifecycles) Stop(t testing.TB) error {
	var err error
	for _, l := range ls {
		err = multierr.Append(err, l.Stop(t))
	}
	assert.NoError(t, err)
	return err
}

// Actions will wrap a list of actions in a sequential executor.
func Actions(actions ...api.Action) api.Action {
	return multi(actions)
}

type multi []api.Action

func (m multi) Run(t testing.TB) {
	for i, req := range m {
		api.Run(fmt.Sprintf("Action #%d", i), t, req.Run)
	}
}
