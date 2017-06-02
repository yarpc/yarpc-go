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

// middlewareOptions enumerates the options for retry middleware.
type middlewareOptions struct {
	// Retries is the number of attempts we will retry (after the
	// initial attempt.
	retries int

	// Timeout is the Timeout we will enforce per request (if this
	// is less than the context deadline, we'll use that instead).
	timeout time.Duration
}

var defaultMiddlewareOptions = middlewareOptions{
	retries: 1,
	timeout: time.Second,
}

// MiddlewareOption customizes the behavior of a retry middleware.
type MiddlewareOption func(*middlewareOptions)

// Retries is the number of attempts we will retry (after the
// initial attempt.
//
// Defaults to 1.
func Retries(retries int) MiddlewareOption {
	return func(options *middlewareOptions) {
		options.retries = retries
	}
}

// PerRequestTimeout is the Timeout we will enforce per request (if this
// is less than the context deadline, we'll use that instead).
//
// Defaults to 1 second.
func PerRequestTimeout(timeout time.Duration) MiddlewareOption {
	return func(options *middlewareOptions) {
		options.timeout = timeout
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
	rereader, finish := ioutil.NewRereader(request.Body)
	defer finish()
	request.Body = rereader

	for i := 0; i < r.opts.retries+1; i++ {
		subCtx, cancel := context.WithTimeout(ctx, r.getTimeout(ctx))
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

		// TODO add backoff semantics
	}
	return resp, err
}

func (r *OutboundMiddleware) getTimeout(ctx context.Context) time.Duration {
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		return r.opts.timeout
	}
	now := time.Now()
	if ctxDeadline.After(now.Add(r.opts.timeout)) {
		return r.opts.timeout
	}
	return ctxDeadline.Sub(now)
}

func isRetryable(err error) bool {
	// TODO(#1080) Update Error assertions to be more granular.
	return transport.IsUnexpectedError(err) || transport.IsTimeoutError(err)
}
