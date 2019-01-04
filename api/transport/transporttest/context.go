// Copyright (c) 2019 Uber Technologies, Inc.
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
	"context"
	"fmt"
	"testing"
	"time"
)

// ContextMatcher is a Matcher for verifying that a context's deadline is
// within expected bounds: the current time, plus a TTL, plus or minus some
// tolerance.
type ContextMatcher struct {
	t   *testing.T
	ttl time.Duration

	TTLDelta time.Duration
}

// ContextMatcherOption customizes the behavior of a ContextMatcher.
type ContextMatcherOption interface {
	run(*ContextMatcher)
}

// ContextTTL requires that a Context have the given TTL on it, with a
// tolerance of TTLDelta.
type ContextTTL time.Duration

func (ttl ContextTTL) run(c *ContextMatcher) {
	c.ttl = time.Duration(ttl)
}

// NewContextMatcher creates a ContextMatcher to verify properties about a
// Context.
func NewContextMatcher(t *testing.T, options ...ContextMatcherOption) *ContextMatcher {
	matcher := &ContextMatcher{t: t, TTLDelta: DefaultTTLDelta}
	for _, opt := range options {
		opt.run(matcher)
	}
	return matcher
}

// Matches a context against an expected context, returning true only if the
// given object is a context with a deadline that is now, plus the expected
// TTL, plus or minus some tolerance.
func (c *ContextMatcher) Matches(got interface{}) bool {
	ctx, ok := got.(context.Context)
	if !ok {
		c.t.Logf("expected a Context but got a %T: %v", got, got)
		return false
	}

	if c.ttl != 0 {
		d, ok := ctx.Deadline()
		if !ok {
			c.t.Logf(
				"expected Context to have a TTL of %v but it has no deadline", c.ttl)
			return false
		}

		ttl := d.Sub(time.Now())
		maxTTL := c.ttl + c.TTLDelta
		minTTL := c.ttl - c.TTLDelta
		if ttl > maxTTL || ttl < minTTL {
			c.t.Logf("TTL out of expected bounds: %v < %v < %v", minTTL, ttl, maxTTL)
			return false
		}
	}

	return true
}

func (c *ContextMatcher) String() string {
	return fmt.Sprintf("ContextMatcher(TTL:%v±%v)", c.ttl, c.TTLDelta)
}
