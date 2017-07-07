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

package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/internal/clock"
	"go.uber.org/yarpc/x/ratelimit"
)

func TestThrottle(t *testing.T) {
	clock := clock.NewFake()
	rl, err := ratelimit.NewThrottle(1, ratelimit.WithClock(clock))
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		assert.False(t, rl.Throttle(), "slack should allow first %d", i)
	}
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Throttle(), "throttle after slack absorbed %d", i)
	}

	// Advance time a second to allow another call
	clock.Add(time.Second)
	assert.False(t, rl.Throttle(), "should allow 1 call after 1s")
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Throttle(), "should throttle after absorbing slack")
	}

	// Advance time twenty seconds, to build up as much slack as possible (10s at 10rps)
	clock.Add(20 * time.Second)
	for i := 0; i < 10; i++ {
		assert.False(t, rl.Throttle(), "slack should allow another %d", i)
	}
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Throttle(), "should throttle after absorbing slack")
	}
}

func TestThrottleWithoutSlack(t *testing.T) {
	clock := clock.NewFake()
	rl, err := ratelimit.NewThrottle(1, ratelimit.WithClock(clock), ratelimit.WithoutSlack)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		assert.True(t, rl.Throttle(), "throttle without slack")
	}

	// Advance time twenty seconds.
	clock.Add(20 * time.Second)
	assert.False(t, rl.Throttle(), "slack should allow a request")
	assert.True(t, rl.Throttle(), "but that is all you get for this second")
}

// Coverage testing for atomic races.
func TestCompetingThrottles(t *testing.T) {
	var wg sync.WaitGroup
	throttle, err := ratelimit.NewThrottle(1000, ratelimit.WithBurstLimit(20))
	require.NoError(t, err)
	count := atomic.NewInt32(0)
	wg.Add(3)
	go run(count, throttle, &wg, 100)
	go run(count, throttle, &wg, 100)
	go run(count, throttle, &wg, 100)
	wg.Wait()
}

func run(count *atomic.Int32, throttle *ratelimit.Throttle, wg *sync.WaitGroup, quantity int32) {
	defer wg.Done()
	for {
		n := count.Load()
		if n > quantity {
			return
		}
		if throttle.Throttle() {
			continue
		}
		count.Inc()
	}
}

func TestOpenThrottle(t *testing.T) {
	rl := ratelimit.OpenThrottle
	for i := 0; i < 10; i++ {
		assert.False(t, rl.Throttle(), "never throttle")
	}
}

func TestThrottleInvalidOptions(t *testing.T) {
	var err error

	_, err = ratelimit.NewThrottle(0)
	assert.Error(t, err, "misconfigured rps")

	_, err = ratelimit.NewThrottle(10, ratelimit.WithBurstLimit(-1))
	assert.Error(t, err, "misconfigured burst limit")
}
