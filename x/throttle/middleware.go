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

package throttle

import (
	"context"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// MiddlewareOption customizes the behavior of a throttle middleware.
type MiddlewareOption interface {
	apply(*middlewareOptions)
}

type throttleOptionFunc func(*middlewareOptions)

func (f throttleOptionFunc) apply(opts *middlewareOptions) { f(opts) }

// middlewareOptions enumerates the options for throttle middleware.
type middlewareOptions struct {
	// scope is an interface for recording metrics to tally.
	scope tally.Scope

	// logger is a zap logger
	logger *zap.Logger

	// rate is the rate in requests per second
	rate int

	// burst is the maximum number of allowed instantaneous requests
	burst int
}

var defaultMiddlewareOptions = middlewareOptions{
	scope:  tally.NoopScope,
	logger: zap.NewNop(),
	rate:   -1,
	burst:  10,
}

// WithTally sets a Tally scope that will be used to record throttle metrics.
func WithTally(scope tally.Scope) MiddlewareOption {
	return throttleOptionFunc(func(opts *middlewareOptions) {
		opts.scope = scope
	})
}

// WithLogger sets a zap Logger that will be used to record throttle logs.
func WithLogger(logger *zap.Logger) MiddlewareOption {
	return throttleOptionFunc(func(opts *middlewareOptions) {
		opts.logger = logger
	})
}

// WithRate sets the rate in requests per second that the throttle will allow.
func WithRate(rate int) MiddlewareOption {
	return throttleOptionFunc(func(opts *middlewareOptions) {
		opts.rate = rate
	})
}

// WithBurstLimit sets the number of allowed instantaneous requests in a burst.
func WithBurstLimit(burst int) MiddlewareOption {
	return throttleOptionFunc(func(opts *middlewareOptions) {
		opts.burst = burst
	})
}

// NewUnaryMiddleware creates a new Throttle Middleware, or returns nil if
// there is no configure rate limit.
func NewUnaryMiddleware(opts ...MiddlewareOption) *OutboundMiddleware {
	options := defaultMiddlewareOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	metrics, stopPush := newMetrics(options.logger, options.scope)

	return &OutboundMiddleware{
		limiter: rate.NewLimiter(rate.Limit(options.rate), options.burst),

		logger:   options.logger,
		scope:    options.scope,
		metrics:  metrics,
		stopPush: stopPush,
	}
}

// OutboundMiddleware is a throttle middleware that wraps a UnaryOutbound with
// Middleware.
type OutboundMiddleware struct {
	limiter *rate.Limiter

	scope    tally.Scope
	logger   *zap.Logger
	metrics  metrics
	stopPush context.CancelFunc
}

// Stop tells the retry middleware to clean itself up before the process stops.
// We currently use this to stop sending metrics to Tally.
func (t *OutboundMiddleware) Stop() error {
	if t == nil {
		return nil
	}
	if t.stopPush != nil {
		t.stopPush()
	}
	return nil
}

// Call implements the middleware.UnaryOutbound interface.
func (t *OutboundMiddleware) Call(ctx context.Context, request *transport.Request, next transport.UnaryOutbound) (resp *transport.Response, err error) {
	before := time.Now()
	if err := t.limiter.Wait(ctx); err != nil {
		t.metrics.drops.Inc()
		return nil, err
	}
	after := time.Now()

	overhead := after.Sub(before)
	t.metrics.overhead.Observe(overhead)
	t.metrics.passes.Inc()

	return next.Call(ctx, request)
}
