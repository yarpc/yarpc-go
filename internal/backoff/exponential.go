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
	rand           *rand.Rand
	minMaxDiff     int64
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

var defaultExponentialOpts = exponentialOptions{
	base: time.Millisecond,
	max:  time.Hour, // :shrug:
	rand: rand.New(rand.NewSource(time.Now().UnixNano())),
}

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
func randGenerator(rand *rand.Rand) ExponentialOption {
	return func(options *exponentialOptions) {
		options.rand = rand
	}
}

// Exponential is an exponential backoff strategy with jitter.  Under the
// aws backoff strategies this is a "Full Jitter" backoff implementation
// https://www.awsarchitectureblog.com/2015/03/backoff.html with the addition
// of a Min and Max Value.  The range of durations will be contained in
// a closed [Min, Max] interval.
// It is a stateless implementation and is safe to use concurrently.
type Exponential struct {
	opts exponentialOptions
}

// NewExponential returns a new Exponential Backoff Strategy.
func NewExponential(opts ...ExponentialOption) (*Exponential, error) {
	options := defaultExponentialOpts
	for _, opt := range opts {
		opt(&options)
	}

	if err := options.validate(); err != nil {
		return nil, err
	}
	options.minMaxDiff = options.max.Nanoseconds() - options.min.Nanoseconds()

	return &Exponential{
		opts: options,
	}, nil
}

// Duration takes an attempt number and returns the duration the caller should
// wait.
func (e *Exponential) Duration(attempts uint) time.Duration {
	minlessBackoff := (1 << attempts) * e.opts.base.Nanoseconds()

	// either the bit shift went negative, or we went past the max
	// duration we're willing to backoff.
	// In both cases we should go to our max value
	if minlessBackoff > e.opts.minMaxDiff || minlessBackoff <= 0 {
		minlessBackoff = e.opts.minMaxDiff
	}

	return e.opts.min + time.Duration(e.opts.rand.Int63n(minlessBackoff+1))
}
