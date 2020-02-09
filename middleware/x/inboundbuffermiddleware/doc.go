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

// Package inboundbuffermiddleware provides a quality-of-service-aware load-shedding buffer for
// inbound unary requests.
//
// This takes the form of a middleware with a life cycle.
// The buffer maintains workers so it must be started and stopped.
//
// The QOS-aware load shedder is a bounded buffer and worker pool.
// The size of the worker pool (Concurrency) limits the  number
// of concurrent requests that the service will handle.
// The size of the bounded buffer dictates the size of the sample
// of requests that the load shedder can select requests from.
//
// The load shedder will evict any expired request.
// If the buffer is full, the load shedder will replace the
// lowest priority request with any higher priority request it receives.
// Workers accept the highest priority requests first.
package inboundbuffermiddleware
