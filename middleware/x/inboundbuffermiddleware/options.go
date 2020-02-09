// Copyright (c) 2020 Uber Technologies, Inc.
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

package inboundbuffermiddleware

import (
	"time"

	"go.uber.org/yarpc/api/priority"
	"go.uber.org/zap"
)

var defaultOptions = options{
	capacity:    256,
	concurrency: 8,
}

type options struct {
	capacity    int
	concurrency int
	logger      *zap.Logger
	prioritizer priority.Prioritizer
	now         func() time.Time
}

// Option is a constructor option for the QOS Buffer (New).
type Option interface {
	apply(*options)
}

// Capacity overrides the size of the bounded buffer.
//
// The default is 256.
func Capacity(capacity int) Option {
	return capacityOption{capacity: capacity}
}

type capacityOption struct {
	capacity int
}

func (o capacityOption) apply(opts *options) {
	opts.capacity = o.capacity
}

// Concurrency overrides the number of concurrent workers will process requests
// off the bounded buffer.
//
// The default is 8.
func Concurrency(concurrency int) Option {
	return concurrencyOption{concurrency: concurrency}
}

type concurrencyOption struct {
	concurrency int
}

func (o concurrencyOption) apply(opts *options) {
	opts.concurrency = o.concurrency
}

// Prioritizer specifies a prioritizer for requests.
//
// The default assigns an equally low priority to all requests.
func Prioritizer(prioritizer priority.Prioritizer) Option {
	return prioritizerOption{prioritizer: prioritizer}
}

type prioritizerOption struct {
	prioritizer priority.Prioritizer
}

func (o prioritizerOption) apply(opts *options) {
	opts.prioritizer = o.prioritizer
}

// Logger specifies a logger for the bounded buffer to send messages for worker
// startup and shutdown.
func Logger(logger *zap.Logger) Option {
	return loggerOption{logger: logger}
}

type loggerOption struct {
	logger *zap.Logger
}

func (o loggerOption) apply(opts *options) {
	opts.logger = o.logger
}

// Time overrides the system clock for determining the current time.
//
// The buffer will decline any request that expires before a worker will
// process it, based on the time.
func Time(now func() time.Time) Option {
	return timeOption{now: now}
}

type timeOption struct {
	now func() time.Time
}

func (o timeOption) apply(opts *options) {
	opts.now = o.now
}
