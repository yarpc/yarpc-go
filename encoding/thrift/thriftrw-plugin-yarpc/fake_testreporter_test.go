// Copyright (c) 2021 Uber Technologies, Inc.
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

package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/golang/mock/gomock"
)

// FakeTestStatus states how a fake TestingT finished.
type FakeTestStatus int

const (
	// Finished is the default. This indicates that we ran to completion but
	// may have recorded errors using Errorf.
	Finished FakeTestStatus = iota

	// Fatal indicates that we ended early with a Fatalf.
	Fatal

	// Panicked indicates that we aborted with a panic.
	Panicked
)

func (f FakeTestStatus) String() string {
	switch f {
	case Finished:
		return "Finished"
	case Fatal:
		return "Fatal"
	case Panicked:
		return "Panicked"
	default:
		return fmt.Sprintf("FakeTestStatus(%v)", int(f))
	}
}

// FakeTestResult contains the result of using a fake TestingT.
type FakeTestResult struct {
	Errors     []string
	Status     FakeTestStatus
	Panic      interface{} // non-nil if we panicked
	PanicTrace string      // non-empty if we panicked
}

// withFakeTestReporter yields a TestReporter that records its results and
// exposes them in FakeTestResult.
func withFakeTestReporter(f func(gomock.TestReporter)) FakeTestResult {
	var (
		r FakeTestResult
		t = fakeTestReporter{&r}
	)

	done := make(chan struct{})
	go func() {
		defer func() {
			if v := recover(); v != nil {
				r.Panic = v
				r.PanicTrace = string(debug.Stack())
				r.Status = Panicked
			}
			close(done)
		}()

		f(&t)
	}()
	<-done

	return r
}

type fakeTestReporter struct{ result *FakeTestResult }

func (t *fakeTestReporter) Errorf(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args)
	}
	t.result.Errors = append(t.result.Errors, msg)
}

func (t *fakeTestReporter) Fatalf(msg string, args ...interface{}) {
	t.Errorf(msg, args...)
	t.result.Status = Fatal

	// this kills the current goroutine, unwinding deferred functions.
	runtime.Goexit()
}
