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

package ratelimit

import (
	"context"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

var errRateLimitExceeded = yarpcerrors.ResourceExhaustedErrorf("rate limit exceeded")

// NewUnaryInboundMiddleware creates a unary inbound middleware that
// introduces a throttle, shedding inbound requests if they arrive more often
// than the configured rate limit.
func NewUnaryInboundMiddleware(rps int, opts ...Option) (*UnaryInboundMiddleware, error) {
	throttle, err := NewThrottle(rps, opts...)
	if err != nil {
		return nil, err
	}
	return &UnaryInboundMiddleware{
		throttle: throttle,
	}, nil
}

// UnaryInboundMiddleware is a unary inbound middleware that sheds inbound
// requests above a rate limit, with some slack for bursts.
type UnaryInboundMiddleware struct {
	throttle *Throttle
}

var _ middleware.UnaryInbound = (*UnaryInboundMiddleware)(nil)

// Handle drops inbound requests with a ResourceExhaustedError if the arrive
// more frequently than the configured rate limit.
func (m *UnaryInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, next transport.UnaryHandler) error {
	if m.throttle.Throttle() {
		return errRateLimitExceeded
	}
	return next.Handle(ctx, req, resw)
}
