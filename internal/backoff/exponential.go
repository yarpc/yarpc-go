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

package backoff

import (
	"errors"
	"math/rand"
	"time"

	"go.uber.org/multierr"
)

// ExponentialOption defines options that can be applied to an
// exponential backoff stragety
type ExponentialOption func(*exponentialOptions)

// exponentialOptions are the configuration options for an exponential backoff
type exponentialOptions struct {
	base, min, max time.Duration
	newRand        func() *rand.Rand
}

func (e exponentialOptions) validate() (err error) {
	if e.base <= 0 {
		err = multierr.Append(err, errors.New("invalid base for exponential backoff, need greater than zero"))
	}
	if e.min < 0 {
		err = multierr.Append(err, errors.New("invalid min for exponential backoff, need greater than or equal to zero"))
	}
	if e.max < 0 {
		err = multierr.Append(err, errors.New("invalid max for exponential backoff, need greater than or equal to zero"))
	}
	if e.max < e.min {
		err = multierr.Append(err, errors.New("exponential max value must be greater than min value"))
	}
	return err
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

var defaultExponentialOpts = exponentialOptions{
	base:    100 * time.Millisecond,
	min:     100 * time.Millisecond,
	max:     time.Minute,
	newRand: newRand,
}

// DefaultExponential is an exponential backoff.Strategy configured
// with a 100ms minimum delay, a 100ms base for a jittered exponential backoff
// curve, and a maximum time to recovery of one minute.
//
// Exponential strategies are not thread safe.
// Use the Isolate() method to obtain an independent backoff generator with the
// same options and a referentially independent random number generator.
var DefaultExponential = (&exponentialStrategy{
	opts: defaultExponentialOpts,
}).NewBackoff

var shortOpts = exponentialOptions{
	base:    time.Duration(0),
	min:     time.Duration(0),
	max:     100 * time.Millisecond,
	newRand: newRand,
}

// ShortExponential is a shorted backoff strategy with 0 min, 0 base, and a
// 100ms max.
//
// Exponential strategies are not thread safe.
// Use the Isolate() method to obtain an independent backoff generator with the
// same options and a referentially independent random number generator.
var ShortExponential = (&exponentialStrategy{
	opts: shortOpts,
}).NewBackoff

// BaseJump sets the default "jump" the exponential backoff strategy will use.
func BaseJump(t time.Duration) ExponentialOption {
	return func(options *exponentialOptions) {
		options.base = t
	}
}

// MaxBackoff sets absolute max time that will ever be returned for a backoff.
func MaxBackoff(t time.Duration) ExponentialOption {
	return func(options *exponentialOptions) {
		options.max = t
	}
}

// MinBackoff sets absolute min time that will ever be returned for a backoff.
func MinBackoff(t time.Duration) ExponentialOption {
	return func(options *exponentialOptions) {
		options.min = t
	}
}

// randGenerator is an internal option for overriding the random number
// generator.
func randGenerator(newRand func() *rand.Rand) ExponentialOption {
	return func(options *exponentialOptions) {
		options.newRand = newRand
	}
}

type exponentialStrategy struct {
	opts exponentialOptions
}

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
func NewExponential(opts ...ExponentialOption) (func() func(uint) time.Duration, error) {
	options := defaultExponentialOpts
	for _, opt := range opts {
		opt(&options)
	}

	if err := options.validate(); err != nil {
		return nil, err
	}

	return (&exponentialStrategy{
		opts: options,
	}).NewBackoff, nil
}

func (e *exponentialStrategy) NewBackoff() func(uint) time.Duration {
	return (&exponentialBackoff{
		base:       e.opts.base,
		min:        e.opts.min,
		max:        e.opts.max,
		minMaxDiff: e.opts.max.Nanoseconds() - e.opts.min.Nanoseconds(),
		rand:       e.opts.newRand(),
	}).Duration
}

type exponentialBackoff struct {
	base, min, max time.Duration
	minMaxDiff     int64
	rand           *rand.Rand
}

// Duration takes an attempt number and returns the duration the caller should
// wait.
func (e *exponentialBackoff) Duration(attempts uint) time.Duration {
	minlessBackoff := (1 << attempts) * e.base.Nanoseconds()

	// either the bit shift went negative, or we went past the max
	// duration we're willing to backoff.
	// In both cases we should go to our max value
	if minlessBackoff > e.minMaxDiff || minlessBackoff <= 0 {
		minlessBackoff = e.minMaxDiff
	}

	return e.min + time.Duration(e.rand.Int63n(minlessBackoff+1))
}
