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

package routertest

import "go.uber.org/yarpc/api/transport"

// Criterion is an argument that adds criteria to a transport request and
// context matcher.
type Criterion func(*Matcher)

// NewMatcher returns a pair of matchers corresponding to the arguments
// of GetHandler(Context, Request), for verifying that GetHandler is called with
// parameters that satisfy given constraints. Passing options like
// WithService("foo") adds constraints to the matcher.
func NewMatcher(criteria ...Criterion) *Matcher {
	m := Matcher{
		constraints: make([]choiceConstraint, 0, 5),
	}
	for _, criterion := range criteria {
		criterion(&m)
	}
	return &m
}

type choiceConstraint func(*transport.Request) bool

// Matcher is a gomock Matcher that validates transport requests with
// given criteria.
type Matcher struct {
	constraints []choiceConstraint
}

// WithCaller adds a constraint that the request caller must match.
func (m *Matcher) WithCaller(caller string) *Matcher {
	m.constraints = append(m.constraints, func(r *transport.Request) bool {
		return caller == r.Caller
	})
	return m
}

// WithService adds a constraint that the request callee must match.
func (m *Matcher) WithService(service string) *Matcher {
	m.constraints = append(m.constraints, func(r *transport.Request) bool {
		return service == r.Service
	})
	return m
}

// WithProcedure adds a constraint that the request procedure must match.
func (m *Matcher) WithProcedure(procedure string) *Matcher {
	m.constraints = append(m.constraints, func(r *transport.Request) bool {
		return procedure == r.Procedure
	})
	return m
}

// Matches returns whether a transport request matches the configured criteria
// for the matcher.
func (m *Matcher) Matches(got interface{}) bool {
	req := got.(*transport.Request)
	for _, check := range m.constraints {
		if !check(req) {
			return false
		}
	}
	return true
}

func (m *Matcher) String() string {
	return "choice matcher"
}
