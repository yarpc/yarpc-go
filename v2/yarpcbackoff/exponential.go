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

package yarpcbackoff

import (
	"errors"
	"math/rand"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
)

var (
	errInvalidFirst = errors.New("invalid first duration for exponential backoff, need greater than zero")
	errInvalidMax   = errors.New("invalid max for exponential backoff, need greater than or equal to zero")
)

// ExponentialOption defines options that can be applied to an
// exponential backoff strategy
type ExponentialOption func(*exponentialOptions)

// exponentialOptions are the configuration options for an exponential backoff
type exponentialOptions struct {
	first, max time.Duration
	newRand    func() *rand.Rand
}

func (e exponentialOptions) validate() (err error) {
	if e.first <= 0 {
		err = multierr.Append(err, errInvalidFirst)
	}
	if e.max < 0 {
		err = multierr.Append(err, errInvalidMax)
	}
	return err
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

var defaultExponentialOpts = exponentialOptions{
	first:   10 * time.Millisecond,
	max:     time.Minute,
	newRand: newRand,
}

// DefaultExponential is an exponential yarpc.BackoffStrategy with full jitter.
// The first attempt has a range of 0 to 10ms and each successive attempt
// doubles the range of the possible delay.
//
// Exponential strategies are not thread safe. The Backoff() method returns a
// referentially independent backoff generator and random number generator.
var DefaultExponential = &ExponentialStrategy{
	opts: defaultExponentialOpts,
}

// FirstBackoff sets the initial range of durations that the first backoff
// duration will provide.
// The range of durations will double for each successive attempt.
func FirstBackoff(t time.Duration) ExponentialOption {
	return func(options *exponentialOptions) {
		options.first = t
	}
}

// MaxBackoff sets absolute max time that will ever be returned for a backoff.
func MaxBackoff(t time.Duration) ExponentialOption {
	return func(options *exponentialOptions) {
		options.max = t
	}
}

// randGenerator is an internal option for overriding the random number
// generator.
func randGenerator(newRand func() *rand.Rand) ExponentialOption {
	return func(options *exponentialOptions) {
		options.newRand = newRand
	}
}

// ExponentialStrategy can create instances of the exponential backoff strategy
// with full jitter.
// Each instance has referentially independent random number generators.
type ExponentialStrategy struct {
	opts exponentialOptions
}

var _ yarpc.BackoffStrategy = (*ExponentialStrategy)(nil)

// NewExponential returns a new exponential backoff strategy, which in turn
// returns backoff functions.
//
// Exponential is an exponential backoff strategy with jitter.  Under the
// AWS backoff strategies this is a "Full Jitter" backoff implementation
// https://www.awsarchitectureblog.com/2015/03/backoff.html with the addition
// of a Min and Max Value.  The range of durations will be contained in
// a closed [Min, Max] interval.
//
// Backoff functions are lockless and referentially independent, but not
// thread-safe.
func NewExponential(opts ...ExponentialOption) (*ExponentialStrategy, error) {
	options := defaultExponentialOpts
	for _, opt := range opts {
		opt(&options)
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	return &ExponentialStrategy{
		opts: options,
	}, nil
}

// Backoff returns an instance of the exponential backoff strategy with its own
// random number generator.
func (e *ExponentialStrategy) Backoff() yarpc.Backoff {
	return &exponentialBackoff{
		first: e.opts.first,
		max:   e.opts.max.Nanoseconds(),
		rand:  e.opts.newRand(),
	}
}

// IsEqual returns whether this strategy is equivalent to another strategy.
func (e *ExponentialStrategy) IsEqual(o *ExponentialStrategy) bool {
	if e.opts.first != o.opts.first {
		return false
	}
	if e.opts.max != o.opts.max {
		return false
	}
	return true
}

// ExponentialBackoff is an instance of the exponential backoff strategy with
// full jitter.
type exponentialBackoff struct {
	first time.Duration
	max   int64
	rand  *rand.Rand
}

// Duration takes an attempt number and returns the duration the caller should
// wait.
func (e *exponentialBackoff) Duration(attempts uint) time.Duration {
	spread := (1 << attempts) * e.first.Nanoseconds()
	if spread <= 0 || spread > e.max {
		spread = e.max
	}
	// Adding 1 to the spread ensures that the upper bound of the range of
	// possible durations includes the maximum.
	return time.Duration(e.rand.Int63n(spread + 1))
}
