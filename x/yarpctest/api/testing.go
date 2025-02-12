// Copyright (c) 2025 Uber Technologies, Inc.
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

package api

import (
	"sync"
	"testing"
)

// Run will cast the testing.TB to it's sub and call the appropriate Run func.
func Run(name string, t testing.TB, f func(testing.TB)) {
	if tt, ok := t.(*testing.T); ok {
		tt.Run(name, func(ttt *testing.T) { f(ttt) })
		return
	}
	if tb, ok := t.(*testing.B); ok {
		tb.Run(name, func(ttb *testing.B) { f(ttb) })
		return
	}
	t.Error("invalid test harness")
	t.FailNow()
}

// SafeTestingTB is a struct that wraps a testing.TB in a mutex for safe concurrent
// usage.
type SafeTestingTB struct {
	sync.Mutex
	t testing.TB
}

// SetTestingTB safely sets the testing.TB.
func (s *SafeTestingTB) SetTestingTB(t testing.TB) {
	s.Lock()
	s.t = t
	s.Unlock()
}

// GetTestingTB safely gets the testing.TB for the testable.
func (s *SafeTestingTB) GetTestingTB() testing.TB {
	s.Lock()
	t := s.t
	s.Unlock()
	return t
}

// SafeTestingTBOnStart is an embeddable struct that automatically grabs testing.TB
// objects on "Start" for lifecycles.
type SafeTestingTBOnStart struct {
	SafeTestingTB
}

// Start safely sets the testing.TB for the testable.
func (s *SafeTestingTBOnStart) Start(t testing.TB) error {
	s.SetTestingTB(t)
	return nil
}

// NoopLifecycle is a convenience struct that can be embedded to make a struct
// implement the Start and Stop methods of Lifecycle.
type NoopLifecycle struct{}

// Start is a Noop.
func (b *NoopLifecycle) Start(t testing.TB) error {
	return nil
}

// Stop is a Noop.
func (b *NoopLifecycle) Stop(t testing.TB) error {
	return nil
}

// NoopStop is a convenience struct that can be embedded to make a struct
// implement the Stop method of Lifecycle.
type NoopStop struct{}

// Stop is a Noop.
func (b *NoopStop) Stop(t testing.TB) error {
	return nil
}
