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

package outboundtest

import (
	"bytes"
	"context"
	"io/ioutil"
	"time"

	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

// OutboundEventCallable is an object that can be used in conjunction with a
// yarpctest.FakeOutbound to create Fake functionality for an outbound through
// OutboundEvents.
// Every call to `OutboundEventCallable.Call` will be sent to an event in the
// OutboundEventCallable's event list.  We will atomically increment the event
// index every time a request is sent, and we will fail the test if too many or
// too few requests were called.
type OutboundEventCallable struct {
	t      require.TestingT
	events []*OutboundEvent
	index  *atomic.Int32
}

// NewOutboundEventCallable sets up an OutboundEventCallable.
func NewOutboundEventCallable(t require.TestingT, events []*OutboundEvent) *OutboundEventCallable {
	return &OutboundEventCallable{
		t:      t,
		events: events,
		index:  atomic.NewInt32(0),
	}
}

// Call implements the yarpctest.OutboundCallable function signature.  It will
// send all requests to the appropriate event registered in the event slice.
func (c *OutboundEventCallable) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	eventIndex := c.index.Inc() - 1
	require.True(c.t, int(eventIndex) < len(c.events), "attempted to execute event #%d on the outbound, there are only %d events", eventIndex+1, len(c.events))
	return c.events[eventIndex].Call(ctx, c.t, req)
}

// Cleanup validates that all events in the OutboundEventCallable were
// called before we finished the test.
func (c *OutboundEventCallable) Cleanup() {
	assert.Equal(
		c.t,
		int(c.index.Load()),
		len(c.events),
		"did not execute the proper number of outbound calls",
	)
}

// OutboundEvent is a struct to validate an individual call to an outbound.
// It has explicit checks for all request and timeout attributes, and it
// can return all the information needed from the request.
type OutboundEvent struct {
	// context.Deadline validation
	WantTimeout       time.Duration
	WantTimeoutBounds time.Duration

	// transport.Request validation
	WantCaller          string
	WantService         string
	WantEncoding        transport.Encoding
	WantProcedure       string
	WantShardKey        string
	WantRoutingKey      string
	WantRoutingDelegate string
	WantHeaders         transport.Headers

	// WantBody validates the request's body
	// It is special because if it is not set, we will not exhaust the
	// request body's io.Reader, which could cause an error up the stack,
	// this is by design for edge test cases.
	WantBody string

	// Indicates that we should block until the context is `Done`
	WaitForTimeout bool

	// Attributes put into the transport.Response object.
	GiveError            error
	GiveApplicationError bool
	GiveRespHeaders      transport.Headers
	GiveRespBody         string
}

// Call will validate a single call to the outbound event based on
// the OutboundEvent's parameters.
func (e *OutboundEvent) Call(ctx context.Context, t require.TestingT, req *transport.Request) (*transport.Response, error) {
	if e.WantTimeout != 0 {
		timeoutBounds := e.WantTimeoutBounds
		if timeoutBounds == 0 {
			timeoutBounds = time.Millisecond * 10
		}
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "wanted context deadline, but there was no deadline")
		deadlineDuration := deadline.Sub(time.Now())
		assert.True(t, deadlineDuration > (e.WantTimeout-timeoutBounds), "deadline was less than expected, want %q (within %s), got %q", e.WantTimeout, timeoutBounds, deadlineDuration)
		assert.True(t, deadlineDuration < (e.WantTimeout+timeoutBounds), "deadline was greater than expected, want %q (within %s), got %q", e.WantTimeout, timeoutBounds, deadlineDuration)
	}

	assertEqualIfSet(t, e.WantCaller, req.Caller, "invalid Caller")
	assertEqualIfSet(t, e.WantService, req.Service, "invalid Service")
	assertEqualIfSet(t, string(e.WantEncoding), string(req.Encoding), "invalid Encoding")
	assertEqualIfSet(t, e.WantProcedure, req.Procedure, "invalid Procedure")
	assertEqualIfSet(t, e.WantShardKey, req.ShardKey, "invalid ShardKey")
	assertEqualIfSet(t, e.WantRoutingKey, req.RoutingKey, "invalid RoutingKey")
	assertEqualIfSet(t, e.WantRoutingDelegate, req.RoutingDelegate, "invalid RoutingDelegate")

	if e.WantHeaders.Len() != 0 {
		assert.Equal(t, e.WantHeaders.Len(), req.Headers.Len(), "unexpected number of headers")
		for key, wantVal := range e.WantHeaders.Items() {
			gotVal, ok := req.Headers.Get(key)
			assert.True(t, ok, "header key %q was not in request headers", key)
			assert.Equal(t, wantVal, gotVal, "invalid request header value for %q", key)
		}
	}

	if e.WantBody != "" {
		body, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err, "got error reading request body")
		assert.Equal(t, e.WantBody, string(body), "request body did not match")
	}

	if e.WaitForTimeout {
		_, ok := ctx.Deadline()
		require.True(t, ok, "attempted to wait on context that has no deadline")
		<-ctx.Done()
	}

	return &transport.Response{
		Body:             ioutil.NopCloser(bytes.NewBuffer([]byte(e.GiveRespBody))),
		Headers:          e.GiveRespHeaders,
		ApplicationError: e.GiveApplicationError,
	}, e.GiveError
}

func assertEqualIfSet(t require.TestingT, want, got string, msgAndArgs ...interface{}) {
	if want != "" {
		assert.Equal(t, want, got, msgAndArgs...)
	}
}
