// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package interceptor defines the interceptor interfaces that are used within each transport.
// The package is currently put under internal because we don't allow customized interceptors at this moment.
// Interceptor interfaces are the alias types of middleware interfaces to share the common utility functions.
package interceptor

import (
	"go.uber.org/yarpc/api/middleware"
)

type (
	// UnaryInbound defines a transport interceptor for `UnaryHandler`s.
	//
	// UnaryInbound interceptor MAY do zero or more of the following: change the
	// context, change the request, call the ResponseWriter, modify the response
	// body by wrapping the ResponseWriter, handle the returned error, call the
	// given handler zero or more times.
	//
	// UnaryInbound interceptor MUST be thread-safe.
	//
	// UnaryInbound interceptor is re-used across requests and MAY be called multiple times
	// for the same request.
	UnaryInbound = middleware.UnaryInbound

	// OnewayInbound defines a transport interceptor for `OnewayHandler`s.
	//
	// OnewayInbound interceptor MAY do zero or more of the following: change the
	// context, change the request, handle the returned error, call the given
	// handler zero or more times.
	//
	// OnewayInbound interceptor MUST be thread-safe.
	//
	// OnewayInbound interceptor is re-used across requests and MAY be called
	// multiple times for the same request.
	OnewayInbound = middleware.OnewayInbound

	// StreamInbound defines a transport interceptor for `StreamHandler`s.
	//
	// StreamInbound interceptor MAY do zero or more of the following: change the
	// stream, handle the returned error, call the given handler zero or more times.
	//
	// StreamInbound interceptor MUST be thread-safe.
	//
	// StreamInbound interceptor is re-used across requests and MAY be called
	// multiple times for the same request.
	StreamInbound = middleware.StreamInbound
)
