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

package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFakeClockAdd(t *testing.T) {
	clock := NewFake()
	start := clock.Now()
	clock.Add(time.Second)
	assert.Equal(t, time.Second, clock.Now().Sub(start))
}

func TestFakeClockSet(t *testing.T) {
	clock := NewFake()
	assert.Equal(t, time.Unix(0, 0), clock.Now())
	clock.Set(time.Unix(100, 0))
	assert.Equal(t, time.Unix(100, 0), clock.Now())
}

func TestFakeClockAfter(t *testing.T) {
	clock := NewFake()
	start := clock.Now()
	done := make(chan struct{})
	then := clock.After(time.Second)

	go func() {
		assert.Equal(t, start.Add(time.Second), <-then)
		close(done)
	}()

	clock.Add(time.Second)

	select {
	case <-done:
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestFakeClockSleep(t *testing.T) {
	t.Skip("TODO this test is flaky, we need to fix, https://github.com/yarpc/yarpc-go/issues/1171")
	clock := NewFake()

	go func() {
		clock.Add(2 * time.Second)
	}()

	clock.Sleep(time.Second)
}

func TestFakeAfterFunc(t *testing.T) {
	clock := NewFake()
	start := clock.Now()
	done := make(chan struct{})
	clock.AfterFunc(time.Second, func() {
		assert.False(t, clock.Now().Before(start.Add(time.Second)), "should be called after one second")
		close(done)
	})
	clock.Add(time.Second)

	select {
	case <-done:
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestFakeTimerStop(t *testing.T) {
	clock := NewFake()
	timer := clock.Timer(60 * time.Second)
	assert.True(t, timer.Stop())
	assert.False(t, timer.Stop())
}

func TestFakeTimerReset(t *testing.T) {
	clock := NewFake()
	timer := clock.Timer(60 * time.Second)
	assert.True(t, timer.Stop())
	assert.False(t, timer.Stop())
	assert.False(t, timer.Reset(time.Second))

	go func() {
		clock.Add(time.Second)
	}()

	select {
	case <-timer.C():
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestFakeTimerResetWithoutStop(t *testing.T) {
	clock := NewFake()
	timer := clock.Timer(60 * time.Second)
	assert.True(t, timer.Reset(time.Second))

	go func() {
		clock.Add(time.Second)
	}()

	select {
	case <-timer.C():
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}
