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
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args)
	}
	t.result.Status = Fatal
	t.result.Errors = append(t.result.Errors, msg)

	// this kills the current goroutine, unwinding deferred functions.
	runtime.Goexit()
}
