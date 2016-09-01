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

import "github.com/yarpc/yarpc-go/internal/errors"

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

//go:generate mockgen -destination=transporttest/register.go -package=transporttest github.com/yarpc/yarpc-go/transport Handler

// Registry maintains and provides access to a collection of procedures and
// their handlers.
type Registry interface {
	// Registers a procedure with this registry under the given service name.
	//
	// service may be empty to indicate that the default service name should
	// be used.
	Register(service, procedure string, handler Handler)

	// Gets the handler for the given service, procedure tuple. An
	// UnrecognizedProcedureError will be returned if the handler does not
	// exist.
	//
	// service may be empty to indicate that the default service name should
	// be used.
	GetHandler(service, procedure string) (Handler, error)
}

// MapRegistry is a Registry that maintains a map of the registered
// procedures.
type MapRegistry struct {
	defaultService string
	entries        map[registryEntry]Handler
}

type registryEntry struct {
	Service, Procedure string
}

// NewMapRegistry builds a new MapRegistry that uses the given name as the
// default service name.
func NewMapRegistry(defaultService string) MapRegistry {
	return MapRegistry{
		defaultService: defaultService,
		entries:        make(map[registryEntry]Handler),
	}
}

// Register registers the procedure with the MapRegistry.
func (m MapRegistry) Register(service, procedure string, handler Handler) {
	if service == "" {
		service = m.defaultService
	}

	m.entries[registryEntry{service, procedure}] = handler
}

// GetHandler retrieves the Handler for the given Procedure or returns an
// error.
func (m MapRegistry) GetHandler(service, procedure string) (Handler, error) {
	if service == "" {
		service = m.defaultService
	}

	if h, ok := m.entries[registryEntry{service, procedure}]; ok {
		return h, nil
	}
	return nil, errors.UnrecognizedProcedureError{
		Service:   service,
		Procedure: procedure,
	}
}
