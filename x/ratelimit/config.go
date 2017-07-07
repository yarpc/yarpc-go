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

import "fmt"

// UnaryInboundMiddlewareConfig describes how to configure and construct a
// unary inbound rate limiter.
type UnaryInboundMiddlewareConfig struct {
	// RPS is the maximum requests per second, after which the inbound will
	// throttle inbound requests with a ResourceExhaustedError of "rate limit
	// exceeded".
	RPS int `config:"rps"`
	// BurstLimit determines how much slack the rate limiter will tolerate for
	// a burst of requests from an idle state before throttling.
	// The default is 10. A burstLimit of 0 implies the default.
	// Use "noSlack" to configure a rate limiter without slack.
	BurstLimit int `config:"burstLimit"`
	// NoSlack configures the rate limiter without any slack, even after idling
	// indefinitely.
	NoSlack bool `config:"noSlack"`
}

// Build creates a unary inbound rate limit middleware, or returns an error if
// the configuration is invalid.
func (c UnaryInboundMiddlewareConfig) Build() (*UnaryInboundMiddleware, error) {
	var opts []Option
	if c.NoSlack && c.BurstLimit > 0 {
		return nil, fmt.Errorf("unary inbound rate limit middleware configured with contradictory noSlack and non-zero BurstLimit (%d)", c.BurstLimit)
	}
	if c.NoSlack {
		opts = append(opts, WithoutSlack)
	}
	if c.BurstLimit > 0 {
		opts = append(opts, WithBurstLimit(c.BurstLimit))
	}
	return NewUnaryInboundMiddleware(c.RPS, opts...)
}
