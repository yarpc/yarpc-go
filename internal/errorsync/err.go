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

package errorsync

import "sync"

// ErrorWaiter is similar to a WaitGroup except it allows collecting failures
// from subtasks.
type ErrorWaiter struct {
	wait   sync.WaitGroup
	lock   sync.Mutex
	errors []error
}

// Submit submits a task for execution on the ErrorWaiter.
//
// The function returns immediately.
func (ew *ErrorWaiter) Submit(f func() error) {
	ew.wait.Add(1)
	go func() {
		defer ew.wait.Done()
		if err := f(); err != nil {
			ew.lock.Lock()
			ew.errors = append(ew.errors, err)
			ew.lock.Unlock()
		}
	}()
}

// Wait waits until all submitted tasks have finished and returns a list of
// all errors that occurred during task execution in no particular order.
func (ew *ErrorWaiter) Wait() []error {
	ew.wait.Wait()
	return ew.errors
}
