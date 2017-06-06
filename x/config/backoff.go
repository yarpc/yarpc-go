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

package config

import (
	"time"

	"go.uber.org/yarpc/internal/backoff"
)

// Backoff specifies a backoff strategy, particularly for retries.
// The only supported strategy at time of writing is "exponential" with full
// jitter. This structure may be extended in the future to support registering
// alternate backoff strategies.
//
//  exponential:
//    min: 100ms
//    base: 100ms
//    max: 30s
type Backoff struct {
	Exponential ExponentialBackoff `config:"exponential"`
}

// Strategy returns a backoff strategy constructor (in terms of the number of
// attempts already made) and the given configuration, or an error.
func (c Backoff) Strategy() (func() func(uint) time.Duration, error) {
	return c.Exponential.Strategy()
}

// ExponentialBackoff details the exponential with full jitter backoff
// strategy.
// For each attempt, the delay before the next attempt will be the minimum,
// plus a random amount of time up to the base duration doubled after each
// attempt, up to the maximum inclusive.
//
//   min: 100ms
//   base: 100ms
//   max: 30s
type ExponentialBackoff struct {
	Min  time.Duration `config:"min"`
	Max  time.Duration `config:"max"`
	Base time.Duration `config:"base"`
}

// Strategy returns an exponential backoff strategy (in terms of the number of
// attempts already made) and the given configuration.
func (c ExponentialBackoff) Strategy() (func() func(uint) time.Duration, error) {
	var opts []backoff.ExponentialOption

	if c.Min > 0 {
		opts = append(opts, backoff.MinBackoff(c.Min))
	}
	if c.Max > 0 {
		opts = append(opts, backoff.MaxBackoff(c.Max))
	}
	if c.Base > 0 {
		opts = append(opts, backoff.BaseJump(c.Base))
	}

	backoff, err := backoff.NewExponential(opts...)
	if err != nil {
		return nil, err
	}
	return backoff, nil
}
