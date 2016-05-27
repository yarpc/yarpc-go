// Copyright (c) 2016 Uber Technologies, Inc.
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

package crossdock

import (
	"fmt"
	"runtime"
)

// T records the result of calling different behaviors.
type T interface {
	Put(interface{})
	FailNow()

	Behavior() string
	Param(key string) string
}

// Params represents args to a test
type Params map[string]string

// Skipf records a skipped test.
//
// This may be called multiple times if multiple tests inside a behavior were
// skipped.
func Skipf(t T, format string, args ...interface{}) {
	t.Put(Entry{
		Status: Skipped,
		Output: fmt.Sprintf(format, args...),
	})
}

// Errorf records a failed test.
//
// This may be called multiple times if multiple tests inside a behavior
// failed.
func Errorf(t T, format string, args ...interface{}) {
	t.Put(Entry{
		Status: Failed,
		Output: fmt.Sprintf(format, args...),
	})
}

// Fatalf records a failed test and stops executing the current behavior.
//
// This may be used to stop executing in case of irrecoverable errors.
func Fatalf(t T, format string, args ...interface{}) {
	Errorf(t, format, args...)
	t.FailNow()
}

// Successf records a successful test.
//
// This may be called multiple times for multiple successful tests inside a
// behavior.
func Successf(t T, format string, args ...interface{}) {
	t.Put(Entry{
		Status: Passed,
		Output: fmt.Sprintf(format, args...),
	})
}

//////////////////////////////////////////////////////////////////////////////

// entryT is a sink that keeps track of entries in-order
type entryT struct {
	behavior string
	params   Params

	entries []interface{}
}

func (*entryT) FailNow() {
	// Exit this goroutine and call any deferred functions
	runtime.Goexit()
}

// Put an entry into the EntrySink.
func (t *entryT) Put(v interface{}) {
	t.entries = append(t.entries, v)
}

// Param gets a key out of the params map
func (t entryT) Param(key string) string {
	return t.params[key]
}

// Behavior returns the test to dispatch on
func (t entryT) Behavior() string {
	return t.behavior
}
