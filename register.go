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

// MapRegistry is a Registry that maintains a map of the registered
// procedures.
type MapRegistry struct {
	defaultService string
	entries        map[transport.ServiceProcedure]transport.HandlerSpec
}

// NewMapRegistry builds a new MapRegistry that uses the given name as the
// default service name.
func NewMapRegistry(defaultService string) MapRegistry {
	return MapRegistry{
		defaultService: defaultService,
		entries:        make(map[transport.ServiceProcedure]transport.HandlerSpec),
	}
}

// Register registers the procedure with the MapRegistry.
func (m MapRegistry) Register(rs []transport.Registrant) {
	for _, r := range rs {
		if r.Service == "" {
			r.Service = m.defaultService
		}

		if r.Procedure == "" {
			panic("Expected procedure name not to be empty string in registration")
		}

		sp := transport.ServiceProcedure{
			Service:   r.Service,
			Procedure: r.Procedure,
		}
		m.entries[sp] = r.HandlerSpec
	}
}

// ServiceProcedures returns a list of services and their procedures that
// have been registered so far.
func (m MapRegistry) ServiceProcedures() []transport.ServiceProcedure {
	procs := make([]transport.ServiceProcedure, 0, len(m.entries))
	for k := range m.entries {
		procs = append(procs, k)
	}
	sort.Sort(byServiceProcedure(procs))
	return procs
}

// ChooseProcedure retrieves the HandlerSpec for the given Procedure or returns an
// error.
func (m MapRegistry) ChooseProcedure(service, procedure string) (transport.HandlerSpec, error) {
	if service == "" {
		service = m.defaultService
	}

	sp := transport.ServiceProcedure{
		Service:   service,
		Procedure: procedure,
	}
	if spec, ok := m.entries[sp]; ok {
		return spec, nil
	}

	return transport.HandlerSpec{}, errors.UnrecognizedProcedureError{
		Service:   service,
		Procedure: procedure,
	}
}

// Choose retrives the HandlerSpec for the service and procedure noted on the
// transport request, or returns an error.
func (m MapRegistry) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return m.ChooseProcedure(req.Service, req.Procedure)
}

type byServiceProcedure []transport.ServiceProcedure

func (sp byServiceProcedure) Len() int {
	return len(sp)
}

func (sp byServiceProcedure) Less(i int, j int) bool {
	if sp[i].Service == sp[j].Service {
		return sp[i].Procedure < sp[j].Procedure
	}
	return sp[i].Service < sp[j].Service
}

func (sp byServiceProcedure) Swap(i int, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}
