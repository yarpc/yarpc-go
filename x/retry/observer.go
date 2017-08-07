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

var (
	_callsName         = "retry_calls"
	_successesName     = "retry_successes"
	_failuresName      = "retry_failures"
	_errTag            = "error"
	_unretryableErrTag = "unretryable"
	_yarpcErrTag       = "yarpc_internal"
	_noTimeErrTag      = "no_time"
	_maxAttemptErrTag  = "max_attempts"
)

type observer struct {
	calls           tally.Counter
	successes       tally.Counter
	unretryableErrs tally.Counter
	yarpcErrs       tally.Counter
	noTimeErrs      tally.Counter
	maxAttemptErrs  tally.Counter
}

func newObserver(scope tally.Scope) *observer {
	unretryableErrScope := scope.Tagged(map[string]string{_errTag: _unretryableErrTag})
	yarpcErrScope := scope.Tagged(map[string]string{_errTag: _yarpcErrTag})
	noTimeErrScope := scope.Tagged(map[string]string{_errTag: _noTimeErrTag})
	maxAttemptErrScope := scope.Tagged(map[string]string{_errTag: _maxAttemptErrTag})
	return &observer{
		calls:           scope.Counter(_callsName),
		successes:       scope.Counter(_successesName),
		unretryableErrs: unretryableErrScope.Counter(_failuresName),
		yarpcErrs:       yarpcErrScope.Counter(_failuresName),
		noTimeErrs:      noTimeErrScope.Counter(_failuresName),
		maxAttemptErrs:  maxAttemptErrScope.Counter(_failuresName),
	}
}

func (o *observer) call() {
	o.calls.Inc(1)
}

func (o *observer) success() {
	o.successes.Inc(1)
}

func (o *observer) unretryableError() {
	o.unretryableErrs.Inc(1)
}

func (o *observer) yarpcError() {
	o.yarpcErrs.Inc(1)
}

func (o *observer) noTimeError() {
	o.noTimeErrs.Inc(1)
}

func (o *observer) maxAttemptsError() {
	o.maxAttemptErrs.Inc(1)
}
