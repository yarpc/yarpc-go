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

// Package priority is an API for determining the priority of requests.
package priority

import (
	"context"
	"strconv"

	"go.uber.org/yarpc/api/transport"
)

// Priority indicates a request's priority.
//
// Small numbers indicate priority over larger numbers.
type Priority int8

// Priority returns the string representation of a priority from "0%" to
// "100%".
func (p Priority) String() string {
	return strconv.Itoa(int(p)) + "%"
}

const (
	// Lowest priority (all other priorities will be served first)
	Lowest = Priority(0)
	// Highest priority (all other priorities will be served later)
	Highest = Priority(100)
)

// Prioritizer extracts or assigns priority for an inbound request context.
type Prioritizer interface {
	// Priority returns the priority of the request in context.
	Priority(context.Context, *transport.RequestMeta) (priority Priority, fortune Priority)
}

type nopPrioritizer struct{}

// Priority returns the lowest priority.
func (nopPrioritizer) Priority(context.Context, *transport.RequestMeta) (Priority, Priority) {
	return Lowest, Lowest
}

// NopPrioritizer assigns the lowest priority to all requests.
var NopPrioritizer Prioritizer = nopPrioritizer{}
