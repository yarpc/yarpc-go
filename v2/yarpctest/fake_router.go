// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpctest

import (
	"context"
	"fmt"

	yarpc "go.uber.org/yarpc/v2"
)

var _ yarpc.Router = (*FakeRouter)(nil)

// FakeRouter is a fake router with procedures.
type FakeRouter struct {
	procedures []yarpc.Procedure
}

// NewFakeRouter creates a fake yarpc.Router.
func NewFakeRouter(procedures []yarpc.Procedure) *FakeRouter {
	return &FakeRouter{procedures}
}

// Procedures returns the procedures given in the constructor.
func (r *FakeRouter) Procedures() []yarpc.Procedure {
	return r.procedures
}

// Choose chooses a yarpc.HandlerSpec based on request.Procedure.
func (r *FakeRouter) Choose(_ context.Context, req *yarpc.Request) (yarpc.TransportHandlerSpec, error) {
	for _, procedure := range r.procedures {
		if procedure.Name == req.Procedure {
			return procedure.HandlerSpec, nil
		}
	}
	return yarpc.TransportHandlerSpec{}, fmt.Errorf("no procedure for name %s", req.Procedure)
}
