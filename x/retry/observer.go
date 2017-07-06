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

package retry

import "github.com/uber-go/tally"

type observer struct {
	unretryableErrorCounter tally.Counter
	yarpcErrorCounter       tally.Counter
	noTimeErrorCounter      tally.Counter
	maxAttemptsErrorCounter tally.Counter
	successCounter          tally.Counter
	callCounter             tally.Counter
}

func newObserver(scope tally.Scope) *observer {
	unretryableErrScope := scope.Tagged(map[string]string{"error": "unretryable"})
	yarpcErrScope := scope.Tagged(map[string]string{"error": "yarpc_internal"})
	noTimeErrScope := scope.Tagged(map[string]string{"error": "notime"})
	maxAttemptsErrScope := scope.Tagged(map[string]string{"error": "max_attempts"})
	return &observer{
		unretryableErrorCounter: unretryableErrScope.Counter("retry_failures"),
		yarpcErrorCounter:       yarpcErrScope.Counter("retry_failures"),
		noTimeErrorCounter:      noTimeErrScope.Counter("retry_failures"),
		maxAttemptsErrorCounter: maxAttemptsErrScope.Counter("retry_failures"),
		successCounter:          scope.Counter("retry_successes"),
		callCounter:             scope.Counter("retry_calls"),
	}
}

func (o *observer) unretryableError() {
	o.unretryableErrorCounter.Inc(1)
}

func (o *observer) yarpcError() {
	o.yarpcErrorCounter.Inc(1)
}

func (o *observer) noTimeError() {
	o.noTimeErrorCounter.Inc(1)
}

func (o *observer) maxAttemptsError() {
	o.maxAttemptsErrorCounter.Inc(1)
}

func (o *observer) success() {
	o.successCounter.Inc(1)
}

func (o *observer) call() {
	o.callCounter.Inc(1)
}
