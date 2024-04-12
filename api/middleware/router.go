// Copyright (c) 2024 Uber Technologies, Inc.
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

package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Router is a middleware for defining a customized routing experience for procedures
type Router interface {
	// Procedures returns the list of procedures that can be called on this router.
	// Procedures SHOULD call into router that is passed in.
	Procedures(transport.Router) []transport.Procedure

	// Choose returns a HandlerSpec for the given request and transport.
	// If the Router cannot determine what to call it should call into the router that was
	// passed in.
	Choose(context.Context, *transport.Request, transport.Router) (transport.HandlerSpec, error)
}

// ApplyRouteTable applies the given Router middleware to the given Router.
func ApplyRouteTable(r transport.RouteTable, m Router) transport.RouteTable {
	if m == nil {
		return r
	}
	return routeTableWithMiddleware{r: r, m: m}
}

type routeTableWithMiddleware struct {
	r transport.RouteTable
	m Router
}

func (r routeTableWithMiddleware) Procedures() []transport.Procedure {
	return r.m.Procedures(r.r)
}

func (r routeTableWithMiddleware) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return r.m.Choose(ctx, req, r.r)
}

func (r routeTableWithMiddleware) Register(procedures []transport.Procedure) {
	r.r.Register(procedures)
}
