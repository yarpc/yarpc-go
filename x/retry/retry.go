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

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/ioutil"
	"go.uber.org/yarpc/yarpcerrors"
)

// MiddlewareOption customizes the behavior of a retry middleware.
type MiddlewareOption interface {
	apply(*middlewareOptions)
}

type retryOptionFunc func(*middlewareOptions)

func (f retryOptionFunc) apply(opts *middlewareOptions) { f(opts) }

// middlewareOptions enumerates the options for retry middleware.
type middlewareOptions struct {
	// policyProvider is a function that will provide a Retry policy for a
	// context and request.
	policyProvider PolicyProvider

	// tallyScope is an interface for recording metrics.
	scope tally.Scope
}

var defaultMiddlewareOptions = middlewareOptions{
	policyProvider: nil,
	scope:          tally.NoopScope,
}

// WithPolicyProvider allows a custom retry policy to be used in the retry
// middleware.
func WithPolicyProvider(provider PolicyProvider) MiddlewareOption {
	return retryOptionFunc(func(opts *middlewareOptions) {
		opts.policyProvider = provider
	})
}

// TallyScope sets a Tally scope that will be used to record retry metrics.
func TallyScope(scope tally.Scope) MiddlewareOption {
	return retryOptionFunc(func(opts *middlewareOptions) {
		opts.scope = scope
	})
}

// NewUnaryMiddleware creates a new Retry Middleware
func NewUnaryMiddleware(opts ...MiddlewareOption) *OutboundMiddleware {
	options := defaultMiddlewareOptions
	for _, opt := range opts {
		opt.apply(&options)
	}
	return &OutboundMiddleware{
		provider: options.policyProvider,
		observer: newObserver(options.scope),
	}
}

// OutboundMiddleware is a retry middleware that wraps a UnaryOutbound with
// Middleware.
type OutboundMiddleware struct {
	provider PolicyProvider
	observer *observer
}

// Call implements the middleware.UnaryOutbound interface.
func (r *OutboundMiddleware) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (resp *transport.Response, err error) {
	if r == nil {
		return out.Call(ctx, request)
	}
	policy := r.getPolicy(ctx, request)
	if policy == nil {
		return out.Call(ctx, request)
	}
	rereader, finish := ioutil.NewRereader(request.Body)
	defer finish()
	request.Body = rereader
	boff := policy.opts.backoffStrategy.Backoff()

	for i := uint(0); i < policy.opts.retries+1; i++ {
		r.observer.call()
		timeout, _ := getTimeLeft(ctx, policy.opts.maxRequestTimeout)
		subCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, err = out.Call(subCtx, request)
		cancel() // Clear the new ctx immdediately after the call

		if err == nil {
			r.observer.success()
			return resp, err
		}

		if !isRetryable(err) {
			r.observer.unretryableError()
			return resp, err
		}

		// Reset the rereader so we can do another request.
		if resetErr := rereader.Reset(); resetErr != nil {
			r.observer.yarpcError()
			// TODO(#1080) Append the reset error to the err.
			err = resetErr
			return resp, err
		}

		boffDur := boff.Duration(i)
		if _, ctxWillTimeout := getTimeLeft(ctx, boffDur); ctxWillTimeout {
			r.observer.noTimeError()
			return resp, err
		}
		time.Sleep(boffDur)
	}
	r.observer.maxAttemptsError()
	return resp, err
}

func (r *OutboundMiddleware) getPolicy(ctx context.Context, request *transport.Request) *Policy {
	if r.provider == nil {
		return nil
	}
	return r.provider.Policy(ctx, request)
}

// getTimeLeft will return the amount of time left in the context or the "max"
// duration passed in.  It will also return a boolean indicating whether the
// context will timeout.
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
	switch yarpcerrors.ErrorCode(err) {
	case yarpcerrors.CodeInternal, yarpcerrors.CodeDeadlineExceeded:
		return true
	default:
		return false
	}
}
