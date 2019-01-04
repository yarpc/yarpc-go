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

package backoff

import "time"

// Strategy is a factory for backoff algorithms.
// Each backoff instance may capture some state, typically a random number
// generator.
// The strategy guarantees that these backoff instances are either
// referentially independent and lockless or thread safe.
//
// Backoff strategies are useful for configuring retry loops, balancing the
// need to recover quickly against denial of service as a failure mode.
type Strategy interface {
	Backoff() Backoff
}

// Backoff is an algorithm for determining how long to wait after a number of
// attempts to perform some action.
// Backoff strategies typically use a random number generator that uses some
// state for feedback.
// Instances of backoff are intended to be used in the stack of a single
// goroutine and must therefore either be referentially independent or lock
// safe.
type Backoff interface {
	Duration(attempts uint) time.Duration
}
