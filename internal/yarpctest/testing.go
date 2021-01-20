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

package yarpctest

import (
	"fmt"
	"runtime"

	"github.com/stretchr/testify/require"
)

// FakeTestStatus states how a fake TestingT finished.
type FakeTestStatus int

const (
	// Finished is the default. This indicates that it was run to completion
	// but may have recorded errors using Errorf.
	Finished FakeTestStatus = iota

	// Fatal indicates that the TestingT aborted early with a FailNow.
	Fatal

	// Panicked indicates that the TestingT aborted with a panic.
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
	Errors []string
	Status FakeTestStatus
	Panic  interface{} // non-nil if we panicked
}

// WithFakeTestingT yields a TestingT that records its results and exposes
// them in FakeTestResult.
func WithFakeTestingT(f func(require.TestingT)) *FakeTestResult {
	var (
		r FakeTestResult
		t = fakeTestingT{&r}
	)

	done := make(chan struct{})
	go func() {
		defer func() {
			if v := recover(); v != nil {
				t.result.Status = Panicked
			}
			close(done)
		}()

		f(&t)
	}()
	<-done

	return &r
}

type fakeTestingT struct{ result *FakeTestResult }

func (t *fakeTestingT) Errorf(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args)
	}
	t.result.Errors = append(t.result.Errors, msg)
}

func (t *fakeTestingT) FailNow() {
	t.result.Status = Fatal

	// this kills the current goroutine, unwinding deferred functions.
	runtime.Goexit()
}
