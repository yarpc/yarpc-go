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

package transporttest

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"
)

// ContextMatcher is a Matcher for verifying that a context's deadline is
// within expected bounds: the current time, plus a TTL, plus or minus some
// tolerance.
type ContextMatcher struct {
	t        *testing.T
	ttl      time.Duration
	ttlDelta time.Duration
}

// NewContextMatcher creates a ContextMatcher for a testing context and an
// expected TTL.
func NewContextMatcher(t *testing.T, ttl time.Duration) *ContextMatcher {
	return &ContextMatcher{
		t:        t,
		ttl:      ttl,
		ttlDelta: DefaultTTLDelta,
	}
}

// Matches a context against an expected context, returning true only if the
// given object is a context with a deadline that is now, plus the expected
// TTL, plus or minus some tolerance.
func (c *ContextMatcher) Matches(got interface{}) bool {
	ctx, ok := got.(context.Context)
	if !ok {
		c.t.Logf("no context")
		return false
	}

	d, ok := ctx.Deadline()
	if !ok {
		c.t.Logf("no context deadline")
		return false
	}

	ttl := d.Sub(time.Now())
	maxTTL := c.ttl + c.ttlDelta
	minTTL := c.ttl - c.ttlDelta
	if ttl > maxTTL || ttl < minTTL {
		c.t.Logf("TTL out of expected bounds: %v < %v < %v", minTTL, ttl, maxTTL)
		return false
	}

	return true
}

func (c *ContextMatcher) String() string {
	return fmt.Sprintf("ContextMatcher(TTL:%v±%v)", c.ttl, c.ttlDelta)
}
