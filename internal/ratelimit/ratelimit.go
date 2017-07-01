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

package ratelimit

import (
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/internal/clock"
)

// Note: This file is inspired by:
// https://github.com/prashantv/go-bench/blob/master/ratelimit

// Clock is the minimum necessary interface to instantiate a throttle with a
// clock or fake clock, compatible with clocks created using
// github.com/andres-erbsen/clock or go.uber.org/yarpc/internal/clock.
type Clock interface {
	Now() time.Time
}

// Throttle is a rate limiter.
type Throttle struct {
	// minAllowableTime is the time relative to the unix epoch in nanoseconds
	// representing the absolute minimum time we will accept requests at.
	// During normal operation this should be less than the current time.
	minAllowableTime *atomic.Int64
	// requestInterval is the number of nanoseconds between requests and
	// consequently the duration to advance the time each time the throttle
	// allows another request.  The request interval is the inverse of requests
	// per nanosecond but conveniently expressible as an integer.
	requestInterval int64
	// maxSlack is maximum the number of nanoseconds the time can fall behind
	// the current time.  The max slack is the burst limit over the rate limit
	// (the burst limit times the inverse of the rate limit.)
	maxSlack int64
	clock    Clock
}

type throttleOptions struct {
	burstLimit int64
	clock      Clock
}

// Option is an option for a rate limiter constructor.
type Option func(*throttleOptions)

// NewThrottle returns a Throttle that will limit to the given RPS.
func NewThrottle(rps int, opts ...Option) *Throttle {
	options := throttleOptions{burstLimit: 10}
	for _, opt := range opts {
		opt(&options)
	}
	if options.clock == nil {
		options.clock = clock.NewReal()
	}

	throttle := &Throttle{
		clock:           options.clock,
		requestInterval: time.Second.Nanoseconds() / int64(rps),
		maxSlack:        options.burstLimit * time.Second.Nanoseconds() / int64(rps),
	}
	throttle.minAllowableTime = atomic.NewInt64(throttle.clock.Now().UnixNano() - throttle.maxSlack)
	return throttle
}

// WithClock returns an option for ratelimit.New that provides an alternate
// Clock implementation, typically a mock Clock for testing.
func WithClock(clock Clock) func(*throttleOptions) {
	return func(options *throttleOptions) {
		options.clock = clock
	}
}

// WithBurstLimit returns an option for ratelimit.New that provides an
// alternate limit for a burst of requests with an idle throttle.
func WithBurstLimit(burstLimit int) func(*throttleOptions) {
	return func(options *throttleOptions) {
		options.burstLimit = int64(burstLimit)
	}
}

// WithoutSlack is an option for ratelimit.New that initializes the limiter
// without any initial tolerance for bursts of traffic.
func WithoutSlack(options *throttleOptions) {
	options.burstLimit = 0
}

// Throttle returns whether a call should be dropped to ensure that accepted
// requests remain time.Second/rate on average.
// All other calls count toward the rate limit.
func (t *Throttle) Throttle() bool {
	now := t.clock.Now().UnixNano()
	// Race to advance the time and permit a request.
	for {
		// Disallow a request if the next allowable time has advanced beyond
		// the current time.
		minAllowableTime := t.minAllowableTime.Load()
		if now <= minAllowableTime {
			return true
		}

		// Clamp the next allowable time so we do not accumulate unlimited
		// slack when we are idle.
		nextMinAllowableTime := minAllowableTime
		clampedMinAllowableTime := now - t.maxSlack
		if nextMinAllowableTime < clampedMinAllowableTime {
			nextMinAllowableTime = clampedMinAllowableTime
		}

		// Advance the time to the next allowable request.
		nextMinAllowableTime = nextMinAllowableTime + t.requestInterval

		// Attempt to commit the decision to accept a request and postpone the
		// next allowable request.
		if t.minAllowableTime.CAS(minAllowableTime, nextMinAllowableTime) {
			return false
		}

		// Failing to advance the time atomically means that throttle was
		// successfully called in parallel.
		// Next time through the loop, the time will be farther into the future
		// and our chance to allow a request may have expired.
		// We do not advance `now` to reduce the probability of spinning in
		// this loop, prefering to throttle if we repeatedly lose the race.

		// Coverage over the next line reveals that tests exercise contention.
		_ = struct{}{}
	}
}

// OpenThrottle is a singleton open throttle. An open throttle provides no rate
// limit.
var OpenThrottle = &openThrottle{}

// openThrottle is an unlimited throttle rate limiter.
type openThrottle struct{}

// Throttle always returns false, since you should never throttle with an
// unlimited rate limiter.
func (*openThrottle) Throttle() bool {
	return false
}
