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
	"go.uber.org/zap"
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

	// scope is an interface for recording metrics to tally.
	scope tally.Scope

	// logger is a zap logger
	logger *zap.Logger
}

var defaultMiddlewareOptions = middlewareOptions{
	policyProvider: nil,
	scope:          tally.NoopScope,
	logger:         zap.NewNop(),
}

// WithPolicyProvider allows a custom retry policy to be used in the retry
// middleware.
func WithPolicyProvider(provider PolicyProvider) MiddlewareOption {
	return retryOptionFunc(func(opts *middlewareOptions) {
		opts.policyProvider = provider
	})
}

// WithTally sets a Tally scope that will be used to record retry metrics.
func WithTally(scope tally.Scope) MiddlewareOption {
	return retryOptionFunc(func(opts *middlewareOptions) {
		opts.scope = scope
	})
}

// WithLogger sets a zap Logger that will be used to record retry logs.
func WithLogger(logger *zap.Logger) MiddlewareOption {
	return retryOptionFunc(func(opts *middlewareOptions) {
		opts.logger = logger
	})
}

// NewUnaryMiddleware creates a new Retry Middleware
func NewUnaryMiddleware(opts ...MiddlewareOption) (*OutboundMiddleware, context.CancelFunc) {
	options := defaultMiddlewareOptions
	for _, opt := range opts {
		opt.apply(&options)
	}
	observer, stopPush := newObserverGraph(options.logger, options.scope)
	return &OutboundMiddleware{
		provider:      options.policyProvider,
		observerGraph: observer,
	}, stopPush
}

// OutboundMiddleware is a retry middleware that wraps a UnaryOutbound with
// Middleware.
type OutboundMiddleware struct {
	provider      PolicyProvider
	observerGraph *observerGraph
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
	call := r.observerGraph.begin(request)

	for i := uint(0); i < policy.opts.retries+1; i++ {
		call.call()
		if i > 0 { // Only log retries if this isn't the first attempt
			call.retryOnError(err)
		}
		timeout, _ := getTimeLeft(ctx, policy.opts.maxRequestTimeout)
		subCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, err = out.Call(subCtx, request)
		cancel() // Clear the new ctx immdediately after the call

		if err == nil {
			call.success()
			return resp, err
		}

		if !isIdempotentProcedureRetryable(err) {
			call.unretryableError(err)
			return resp, err
		}

		// Reset the rereader so we can do another request.
		if resetErr := rereader.Reset(); resetErr != nil {
			call.yarpcInternalError(err)
			// TODO(#1080) Append the reset error to the err.
			err = resetErr
			return resp, err
		}

		boffDur := boff.Duration(i)
		if _, ctxWillTimeout := getTimeLeft(ctx, boffDur); ctxWillTimeout {
			call.noTimeError(err)
			return resp, err
		}
		time.Sleep(boffDur)
	}
	call.maxAttemptsError(err)
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

func isIdempotentProcedureRetryable(err error) bool {
	switch yarpcerrors.ErrorCode(err) {
	case yarpcerrors.CodeInternal, yarpcerrors.CodeDeadlineExceeded, yarpcerrors.CodeUnavailable, yarpcerrors.CodeUnknown:
		return true
	default:
		return false
	}
}
