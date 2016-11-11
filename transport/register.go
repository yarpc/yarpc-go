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

package transport

import (
	"sort"

	"go.uber.org/yarpc/internal/errors"
)

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

//go:generate mockgen -destination=transporttest/register.go -package=transporttest go.uber.org/yarpc/transport UnaryHandler,Registry

// ServiceProcedure represents a service and procedure registered against a
// Registry.
type ServiceProcedure struct {
	Service   string
	Procedure string
}

// Registrant specifies a single handler registered against the registry.
type Registrant struct {
	// Service name or empty to use the default service name.
	Service string

	// Name of the procedure.
	Procedure string

	// Handler implementing the given procedure.
	Handler UnaryHandler
}

// Registry maintains and provides access to a collection of procedures and
// their handlers.
type Registry interface {
	// ServiceProcedures returns a list of services and their procedures that
	// have been registered so far.
	ServiceProcedures() []ServiceProcedure

	// Gets the handler for the given service, procedure tuple. An
	// UnrecognizedProcedureError will be returned if the handler does not
	// exist.
	//
	// service may be empty to indicate that the default service name should
	// be used.
	GetHandler(service, procedure string) (UnaryHandler, error)
}

// Registrar provides access to a collection of procedures and their handlers.
type Registrar interface {
	Registry

	// Registers zero or more registrants with the registry.
	Register([]Registrant)
}

// MapRegistry is a Registry that maintains a map of the registered
// procedures.
type MapRegistry struct {
	defaultService string
	unaryEntries   map[ServiceProcedure]UnaryHandler
}

// NewMapRegistry builds a new MapRegistry that uses the given name as the
// default service name.
func NewMapRegistry(defaultService string) MapRegistry {
	return MapRegistry{
		defaultService: defaultService,
		unaryEntries:   make(map[ServiceProcedure]UnaryHandler),
	}
}

// Register registers the procedure with the MapRegistry.
func (m MapRegistry) Register(rs []Registrant) {
	for _, r := range rs {
		if r.Service == "" {
			r.Service = m.defaultService
		}

		m.unaryEntries[ServiceProcedure{r.Service, r.Procedure}] = r.Handler
	}
}

// ServiceProcedures returns a list of services and their procedures that
// have been registered so far.
func (m MapRegistry) ServiceProcedures() []ServiceProcedure {
	procs := make([]ServiceProcedure, 0, len(m.unaryEntries))
	for k := range m.unaryEntries {
		procs = append(procs, k)
	}
	sort.Sort(byServiceProcedure(procs))
	return procs
}

// GetHandler retrieves the Handler for the given Procedure or returns an
// error.
func (m MapRegistry) GetHandler(service, procedure string) (UnaryHandler, error) {
	if service == "" {
		service = m.defaultService
	}

	if h, ok := m.unaryEntries[ServiceProcedure{service, procedure}]; ok {
		return h, nil
	}

	return nil, errors.UnrecognizedProcedureError{
		Service:   service,
		Procedure: procedure,
	}
}

type byServiceProcedure []ServiceProcedure

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
