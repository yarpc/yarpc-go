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

package retry

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	. "go.uber.org/yarpc/internal/yarpctest/outboundtest"
	"go.uber.org/yarpc/yarpctest"
)

// MiddlewareAction is an action applied to a middleware.
type MiddlewareAction interface {
	Apply(*testing.T, middleware.UnaryOutbound)
}

// RequestAction is an Action for sending a request to a
// unary outbound middleware and asserting on the result.
type RequestAction struct {
	msg string

	request    *transport.Request
	reqTimeout time.Duration

	events []*OutboundEvent

	wantTimeLimit        time.Duration
	wantError            string
	wantApplicationError bool
	wantBody             string
}

func (r RequestAction) Apply(t *testing.T, mw middleware.UnaryOutbound) {
	callable := NewOutboundEventCallable(t, r.events)
	defer callable.Cleanup()

	trans := yarpctest.NewFakeTransport()
	out := trans.NewOutbound(yarpctest.NewFakePeerList(), yarpctest.OutboundCallOverride(callable.Call))
	out.Start()

	ctx := context.Background()
	if r.reqTimeout != 0 {
		if r.wantTimeLimit == 0 {
			r.wantTimeLimit = r.reqTimeout + testtime.Millisecond*10
		}

		newCtx, cancel := context.WithTimeout(ctx, testtime.Scale(r.reqTimeout))
		defer cancel()
		ctx = newCtx
	}

	start := time.Now()
	resp, err := mw.Call(ctx, r.request, out)
	elapsed := time.Now().Sub(start)

	if r.wantTimeLimit > 0 {
		assert.True(t, r.wantTimeLimit > elapsed, "execution took to long, wanted %s, took %s", r.wantTimeLimit, elapsed)
	}

	if r.wantError != "" {
		assert.EqualError(t, err, r.wantError)
		require.NotNil(t, resp)
		assert.Equal(t, r.wantApplicationError, resp.ApplicationError)
	} else {
		require.NotNil(t, resp)
		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, r.wantBody, string(body))
	}
}

// ConcurrentAction executes a plan of actions, with a given interval between
// applying actions, but allowing every action to run concurrently in a
// goroutine until its independent completion time.
// The ConcurrentAction allows us to observe overlapping actions.
type ConcurrentAction struct {
	Actions []MiddlewareAction
	Wait    time.Duration
}

// Apply runs all the ConcurrentAction's actions in goroutines with a delay of `Wait`
// between each action. Returns when all actions have finished executing
func (a ConcurrentAction) Apply(t *testing.T, mw middleware.UnaryOutbound) {
	var wg sync.WaitGroup

	wg.Add(len(a.Actions))
	for _, action := range a.Actions {
		go func(ac MiddlewareAction) {
			defer wg.Done()
			ac.Apply(t, mw)
		}(action)

		if a.Wait > 0 {
			testtime.Sleep(a.Wait)
		}
	}

	wg.Wait()
}

// ApplyMiddlewareActions runs all the MiddlewareActions on the Unary outbound Middleware
func ApplyMiddlewareActions(t *testing.T, mw middleware.UnaryOutbound, actions []MiddlewareAction) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, mw)
		})
	}
}
