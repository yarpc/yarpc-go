// Copyright (c) 2021 Uber Technologies, Inc.
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

package api

import (
	"testing"

	"go.uber.org/yarpc/api/transport"
)

// ServerStreamAction is an action applied to a ServerStream.
// If the action returns an error, that error will be used to end the ServerStream.
type ServerStreamAction interface {
	Lifecycle

	ApplyServerStream(*transport.ServerStream) error
}

// ServerStreamActionFunc converts a function into a StreamAction.
type ServerStreamActionFunc func(*transport.ServerStream) error

// ApplyServerStream implements ServerStreamAction.
func (f ServerStreamActionFunc) ApplyServerStream(c *transport.ServerStream) error { return f(c) }

// Start is a noop for wrapped functions
func (f ServerStreamActionFunc) Start(testing.TB) error { return nil }

// Stop is a noop for wrapped functions
func (f ServerStreamActionFunc) Stop(testing.TB) error { return nil }
