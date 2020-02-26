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

import "time"

// RealClock implements a real-time clock by simply wrapping the time package functions.
type RealClock struct{}

// NewReal returns an instance of a real clock (changing in the very real fourth dimension).
func NewReal() RealClock {
	return RealClock{}
}

var _ Clock = (*RealClock)(nil)

// After produces a channel that will emit the time after a duration passes.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// AfterFunc waits for the duration to elapse and then executes a function.
// A Timer is returned that can be stopped.
func (RealClock) AfterFunc(d time.Duration, f func()) Timer {
	return &realTimer{time.AfterFunc(d, f)}
}

// Now returns the current time on the real clock.
func (RealClock) Now() time.Time { return time.Now() }

// Sleep pauses the goroutine for the given duration on the fake clock.
// The clock must be moved forward in a separate goroutine.
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }

// Timer produces a timer that will emit a time some duration after now.
func (RealClock) Timer(d time.Duration) Timer {
	return &realTimer{t: time.NewTimer(d)}
}

type realTimer struct {
	t *time.Timer
}

func (t *realTimer) Stop() bool {
	return t.t.Stop()
}

func (t *realTimer) Reset(d time.Duration) bool {
	return t.t.Reset(d)
}

func (t *realTimer) C() <-chan time.Time {
	return t.t.C
}
