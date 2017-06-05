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
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/ioutil"
)

// MiddlewareOption customizes the behavior of a retry middleware.
type MiddlewareOption func(*middlewareOptions)

// middlewareOptions enumerates the options for retry middleware.
type middlewareOptions struct {
	// policyProvider is a function that will provide a Retry policy
	// for a context and request.
	policyProvider PolicyProvider
}

var defaultMiddlewareOptions = middlewareOptions{
	policyProvider: func(context.Context, *transport.Request) *Policy {
		return &defaultPolicy
	},
}

// WithPolicyProvider allows a custom retry policy to be used in the retry
// middleware.
func WithPolicyProvider(provider PolicyProvider) MiddlewareOption {
	return func(opts *middlewareOptions) {
		opts.policyProvider = provider
	}
}

// NewUnaryMiddleware creates a new Retry Middleware
func NewUnaryMiddleware(opts ...MiddlewareOption) *OutboundMiddleware {
	options := defaultMiddlewareOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &OutboundMiddleware{options}
}

// OutboundMiddleware is a retry middleware that wraps a UnaryOutbound with
// Middleware.
type OutboundMiddleware struct {
	opts middlewareOptions
}

// Call implements the middleware.UnaryOutbound interface.
func (r *OutboundMiddleware) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (resp *transport.Response, err error) {
	policy := r.getPolicy(ctx, request)
	rereader, finish := ioutil.NewRereader(request.Body)
	defer finish()
	request.Body = rereader
	boff := policy.backoffStrategy.Backoff()

	for i := uint(0); i < policy.retries+1; i++ {
		timeout, _ := getTimeLeft(ctx, policy.timeout)
		subCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, err = out.Call(subCtx, request)
		cancel() // Clear the new ctx immdediately after the call

		if err == nil || !isRetryable(err) {
			return resp, err
		}

		// Reset the rereader so we can do another request.
		if resetErr := rereader.Reset(); resetErr != nil {
			// TODO(#1080) Append the reset error to the err.
			err = resetErr
			return resp, err
		}

		boffDur := boff.Duration(i)
		if _, ctxWillTimeout := getTimeLeft(ctx, boffDur); ctxWillTimeout {
			return resp, err
		}
		time.Sleep(boffDur)
	}
	return resp, err
}

func (r *OutboundMiddleware) getPolicy(ctx context.Context, request *transport.Request) *Policy {
	if r.opts.policyProvider == nil {
		return &defaultPolicy
	}
	if pol := r.opts.policyProvider(ctx, request); pol != nil {
		return pol
	}
	return &defaultPolicy
}

// getTimeLeft will return the amount of time left in the context or the
// "max" duration passed in.  It will also return a boolean indicating
// whether the context will timeout.
func getTimeLeft(ctx context.Context, max time.Duration) (timeleft time.Duration, ctxWillTimeout bool) {
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		return max, false
	}
	now := time.Now()
	if ctxDeadline.After(now.Add(max)) {
		return max, false
	}
	return ctxDeadline.Sub(now), true
}

func isRetryable(err error) bool {
	// TODO(#1080) Update Error assertions to be more granular.
	return transport.IsUnexpectedError(err) || transport.IsTimeoutError(err)
}
