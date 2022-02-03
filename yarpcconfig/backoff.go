// Copyright (c) 2022 Uber Technologies, Inc.
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

package yarpcconfig

import (
	"time"

	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/internal/backoff"
)

// Backoff specifies a backoff strategy, particularly for retries.
// The only supported strategy at time of writing is "exponential" with full
// jitter. This structure may be extended in the future to support registering
// alternate backoff strategies.
//
//  exponential:
//    first: 100ms
//    max: 30s
type Backoff struct {
	Exponential ExponentialBackoff `config:"exponential"`
}

// Strategy returns a backoff strategy constructor (in terms of the number of
// attempts already made) and the given configuration, or an error.
func (c Backoff) Strategy() (backoffapi.Strategy, error) {
	return c.Exponential.Strategy()
}

// ExponentialBackoff details the exponential with full jitter backoff
// strategy.
// "first" defines the range of possible durations for the first attempt.
// Each subsequent attempt has twice the range of possible jittered delay
// duration.
// The range of possible values will not exceed "max", inclusive.
//
//   first: 100ms
//   max: 30s
type ExponentialBackoff struct {
	First time.Duration `config:"first"`
	Max   time.Duration `config:"max"`
}

// Strategy returns an exponential backoff strategy (in terms of the number of
// attempts already made) and the given configuration.
func (c ExponentialBackoff) Strategy() (backoffapi.Strategy, error) {
	var opts []backoff.ExponentialOption

	if c.First > 0 {
		opts = append(opts, backoff.FirstBackoff(c.First))
	}
	if c.Max > 0 {
		opts = append(opts, backoff.MaxBackoff(c.Max))
	}

	return backoff.NewExponential(opts...)
}
