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

package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// These real clock tests at best exercise code coverage but, wanting not to
// rely on precise timing or even spend wall-clock time testing these
// interfaces.

func TestRealClockNow(t *testing.T) {
	clock := NewReal()
	clock.Now()
}

func TestRealClockAfter(t *testing.T) {
	clock := NewReal()
	done := make(chan struct{})
	then := clock.After(time.Millisecond)

	go func() {
		<-then
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestRealClockSleep(t *testing.T) {
	clock := NewReal()
	clock.Sleep(time.Millisecond)
}

func TestRealAfterFunc(t *testing.T) {
	clock := NewReal()
	done := make(chan struct{})
	clock.AfterFunc(time.Millisecond, func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestTimerStop(t *testing.T) {
	clock := NewReal()
	timer := clock.Timer(60 * time.Second)
	assert.True(t, timer.Stop())
	assert.False(t, timer.Stop())
}

func TestRealTimerReset(t *testing.T) {
	clock := NewReal()
	timer := clock.Timer(60 * time.Millisecond)
	assert.True(t, timer.Stop())
	assert.False(t, timer.Stop())
	assert.False(t, timer.Reset(time.Millisecond))

	select {
	case <-timer.C():
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}

func TestTimerResetWithoutStop(t *testing.T) {
	clock := NewReal()
	timer := clock.Timer(60 * time.Second)
	assert.True(t, timer.Reset(time.Millisecond))

	select {
	case <-timer.C():
	case <-time.After(time.Second):
		assert.Fail(t, "test timed out")
	}
}
