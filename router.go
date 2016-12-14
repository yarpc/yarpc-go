// Copyright (c) 2016 Uber Technologies, Inc.
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

package yarpc

import (
	"context"
	"sort"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
)

type serviceProcedure struct {
	service   string
	procedure string
}

// MapRouter is a Router that maintains a map of the registered
// procedures.
type MapRouter struct {
	defaultService string
	entries        map[serviceProcedure]transport.Procedure
}

// NewMapRouter builds a new MapRouter that uses the given name as the
// default service name.
func NewMapRouter(defaultService string) MapRouter {
	return MapRouter{
		defaultService: defaultService,
		entries:        make(map[serviceProcedure]transport.Procedure),
	}
}

// Register registers the procedure with the MapRouter.
func (m MapRouter) Register(rs []transport.Procedure) {
	for _, r := range rs {
		if r.Service == "" {
			r.Service = m.defaultService
		}

		if r.Name == "" {
			panic("Expected procedure name not to be empty string in registration")
		}

		sp := serviceProcedure{
			service:   r.Service,
			procedure: r.Name,
		}
		m.entries[sp] = r
	}
}

// Procedures returns a list procedures that
// have been registered so far.
func (m MapRouter) Procedures() []transport.Procedure {
	procs := make([]transport.Procedure, 0, len(m.entries))
	for _, v := range m.entries {
		procs = append(procs, v)
	}
	sort.Sort(transport.ProceduresByServiceProcedure(procs))
	return procs
}

// ChooseProcedure retrieves the HandlerSpec for the given Procedure or returns an
// error.
func (m MapRouter) ChooseProcedure(service, procedure string) (transport.HandlerSpec, error) {
	if service == "" {
		service = m.defaultService
	}

	sp := serviceProcedure{
		service:   service,
		procedure: procedure,
	}
	if procedure, ok := m.entries[sp]; ok {
		return procedure.HandlerSpec, nil
	}

	return transport.HandlerSpec{}, errors.UnrecognizedProcedureError{
		Service:   service,
		Procedure: procedure,
	}
}

// Choose retrives the HandlerSpec for the service and procedure noted on the
// transport request, or returns an error.
func (m MapRouter) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return m.ChooseProcedure(req.Service, req.Procedure)
}
