// Copyright (c) 2018 Uber Technologies, Inc.
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

// Forked from github.com/andres-erbsen/clock to isolate a missing nap.

import (
	"container/heap"
	"runtime"
	"sync"
	"time"
)

// FakeClock represents a fake clock that only moves forward programmically.
// It can be preferable to a real-time clock when testing time-based functionality.
type FakeClock struct {
	sync.Mutex

	addLock sync.Mutex
	now     time.Time
	timers  timers
}

var _ Clock = (*FakeClock)(nil)

// NewFake returns an instance of a fake clock.
// The current time of the fake clock on initialization is the Unix epoch.
func NewFake() *FakeClock {
	// Note: Unix(0, 0) is not the zero value for time, and we need the zero
	// value to distinguish rate limiters that have not started.
	return &FakeClock{now: time.Unix(0, 0)}
}

// Add moves the current time of the fake clock forward by the duration.
// This should only be called from a single goroutine at a time.
func (fc *FakeClock) Add(d time.Duration) {
	fc.Lock()
	// Calculate the final time.
	end := fc.now.Add(d)
	fc.flush(end)

	if fc.now.Before(end) {
		fc.now = end
	}
	fc.Unlock()
	nap()
}

// Set advances the current time of the fake clock to the given absolute time.
func (fc *FakeClock) Set(end time.Time) {
	fc.Lock()
	fc.flush(end)

	if fc.now.Before(end) {
		fc.now = end
	}
	fc.Unlock()
	nap()
}

// flush runs all timers before the given end time and is used to run newly
// added timers as well as expired timers in Add().
func (fc *FakeClock) flush(end time.Time) {
	for len(fc.timers) > 0 && !fc.timers[0].time.After(end) {
		t := fc.timers[0]
		heap.Pop(&fc.timers)
		if fc.now.Before(t.time) {
			fc.now = t.time
		}
		fc.Unlock()
		t.tick()
		fc.Lock()
	}
}

// FakeTimer produces a timer that will emit a time some duration after now,
// exposing the fake timer internals and type.
func (fc *FakeClock) FakeTimer(d time.Duration) *FakeTimer {
	fc.Lock()
	defer fc.Unlock()

	c := make(chan time.Time, 1)
	t := &FakeTimer{
		c:     c,
		clock: fc,
		time:  fc.now.Add(d),
	}
	fc.addTimer(t)
	return t
}

// Timer produces a timer that will emit a time some duration after now.
func (fc *FakeClock) Timer(d time.Duration) Timer {
	return fc.FakeTimer(d)
}

func (fc *FakeClock) addTimer(t *FakeTimer) {
	heap.Push(&fc.timers, t)
	fc.flush(fc.now)
}

// After produces a channel that will emit the time after a duration passes.
func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	return fc.Timer(d).C()
}

// FakeAfterFunc waits for the duration to elapse and then executes a function.
// A Timer is returned that can be stopped.
func (fc *FakeClock) FakeAfterFunc(d time.Duration, f func()) *FakeTimer {
	t := fc.FakeTimer(d)
	go func() {
		<-t.c
		f()
	}()
	nap()
	return t
}

// AfterFunc waits for the duration to elapse and then executes a function.
// A Timer is returned that can be stopped.
func (fc *FakeClock) AfterFunc(d time.Duration, f func()) Timer {
	return fc.FakeAfterFunc(d, f)
}

// Now returns the current time on the fake clock.
func (fc *FakeClock) Now() time.Time {
	fc.Lock()
	defer fc.Unlock()
	return fc.now
}

// Sleep pauses the goroutine for the given duration on the fake clock.
// The clock must be moved forward in a separate goroutine.
func (fc *FakeClock) Sleep(d time.Duration) {
	<-fc.After(d)
}

// FakeTimer represents a single event.
type FakeTimer struct {
	c     chan time.Time
	time  time.Time
	clock *FakeClock
	index int
}

// C returns a channel that will send the time when it fires.
func (t *FakeTimer) C() <-chan time.Time {
	return t.c
}

// tick advances the clock to this timer.
func (t *FakeTimer) tick() {
	select {
	case t.c <- t.time:
	default:
	}
	nap()
}

// Reset adjusts the timer's scheduled time forward from now, unless it has
// already fired.
func (t *FakeTimer) Reset(d time.Duration) bool {
	t.time = t.clock.now.Add(d)

	// Empty the channel if already filled.
	select {
	case <-t.c:
	default:
	}

	if t.index >= 0 {
		heap.Fix(&t.clock.timers, t.index)
		return true
	}
	heap.Push(&t.clock.timers, t)
	return false
}

// Stop removes a timer from the scheduled timers.
func (t *FakeTimer) Stop() bool {
	if t.index < 0 {
		return false
	}

	// Empty the channel if already filled.
	select {
	case <-t.c:
	default:
	}

	t.clock.timers.Swap(t.index, len(t.clock.timers)-1)
	t.clock.timers.Pop()
	heap.Fix(&t.clock.timers, t.index)
	return true
}

func nap() {
	// time.Sleep(time.Millisecond)
	runtime.Gosched()
}
