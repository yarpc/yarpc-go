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

import (
	"time"

	"go.uber.org/yarpc/api/backoff"
	ibackoff "go.uber.org/yarpc/internal/backoff"
)

// Policy defines how a retry will be applied.  It contains all the information
// needed to perform a retry.
type Policy struct {
	opts policyOptions
}

// NewPolicy creates a new retry Policy that can be used in retry middleware.
func NewPolicy(opts ...PolicyOption) *Policy {
	policyOpts := defaultPolicyOpts
	for _, opt := range opts {
		opt.apply(&policyOpts)
	}
	return &Policy{opts: policyOpts}
}

var defaultPolicyOpts = policyOptions{
	retries:           0,
	maxRequestTimeout: time.Second,
	backoffStrategy:   ibackoff.None,
}

type policyOptions struct {
	// retries is the number of times we will retry the request (after the
	// initial attempt).
	retries uint

	// maxRequestTimeout is the timeout we will enforce for each outgoing
	// request.  This will be clamped down to the context deadline.
	maxRequestTimeout time.Duration

	// backoffStrategy is a backoff strategy that will be called after every
	// retry.
	backoffStrategy backoff.Strategy
}

// PolicyOption customizes the behavior of a retry policy.
type PolicyOption interface {
	apply(*policyOptions)
}

type policyOptionFunc func(*policyOptions)

func (f policyOptionFunc) apply(opts *policyOptions) { f(opts) }

// Retries is the number of times we will retry the request (after the initial
// attempt).
//
// Defaults to 1.
func Retries(retries uint) PolicyOption {
	return policyOptionFunc(func(opts *policyOptions) {
		opts.retries = retries
	})
}

// MaxRequestTimeout is the Timeout we will enforce per request (if this is
// greater than the context deadline, we'll use that instead).
//
// Defaults to 1 second.
func MaxRequestTimeout(timeout time.Duration) PolicyOption {
	return policyOptionFunc(func(opts *policyOptions) {
		opts.maxRequestTimeout = timeout
	})
}

// BackoffStrategy sets the backoff strategy that will be used after each failed
// request.
//
// Defaults to no backoff.
func BackoffStrategy(strategy backoff.Strategy) PolicyOption {
	return policyOptionFunc(func(opts *policyOptions) {
		if strategy != nil {
			opts.backoffStrategy = strategy
		}
	})
}
