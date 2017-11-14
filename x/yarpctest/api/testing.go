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

package api

import (
	"sync"
	"testing"
)

// TestingT is an interface wrapper around *testing.T and *testing.B
type TestingT interface {
	testing.TB
}

// Run will cast the TestingT to it's sub and call the appropriate Run func.
func Run(name string, t TestingT, f func(TestingT)) {
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

// SafeTestingT is a struct that wraps a TestingT in a mutex for safe concurrent
// usage.
type SafeTestingT struct {
	sync.Mutex
	t TestingT
}

// SetTestingT safely sets the TestingT.
func (s *SafeTestingT) SetTestingT(t TestingT) {
	s.Lock()
	s.t = t
	s.Unlock()
}

// GetTestingT safely gets the TestingT for the testable.
func (s *SafeTestingT) GetTestingT() TestingT {
	s.Lock()
	t := s.t
	s.Unlock()
	return t
}

// SafeTestingTOnStart is an embeddable struct that automatically grabs TestingT
// objects on "Start" for lifecycles.
type SafeTestingTOnStart struct {
	SafeTestingT
}

// Start safely sets the TestingT for the testable.
func (s *SafeTestingTOnStart) Start(t TestingT) error {
	s.SetTestingT(t)
	return nil
}

// NoopLifecycle is a convenience struct that can be embedded to make a struct
// implement the Start and Stop methods of Lifecycle.
type NoopLifecycle struct{}

// Start is a Noop.
func (b *NoopLifecycle) Start(t TestingT) error {
	return nil
}

// Stop is a Noop.
func (b *NoopLifecycle) Stop(t TestingT) error {
	return nil
}

// NoopStop is a convenience struct that can be embedded to make a struct
// implement the Stop method of Lifecycle.
type NoopStop struct{}

// Stop is a Noop.
func (b *NoopStop) Stop(t TestingT) error {
	return nil
}
