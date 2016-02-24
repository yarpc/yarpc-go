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
	"fmt"

	"golang.org/x/net/context"
)

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

//go:generate mockgen -destination=testing/register.go -package=test_transport github.com/yarpc/yarpc-go/transport Handler

// Handler handles a single transport-level request.
type Handler interface {
	Handle(ctx context.Context, req *Request, resw ResponseWriter) error
}

// Registry maintains and provides access to a collection of procedures and
// their handlers.
type Registry interface {
	// Registers a procedure with this registry.
	Register(procedure string, handler Handler)

	// Gets the handler for the given procedure. An UnknownProcedureErr may be
	// returned if the handler does not exist.
	GetHandler(procedure string) (Handler, error)
}

// UnknownProcedureErr is returned if a procedure that is not known was
// requested.
type UnknownProcedureErr struct {
	Name string
}

func (e UnknownProcedureErr) Error() string {
	return fmt.Sprintf("unknown procedure %q", e.Name)
}

// MapRegistry is a Registry that maintains a map of the registered
// procedures.
type MapRegistry map[string]Handler

// Register registers the procedure with the MapRegistry.
func (m MapRegistry) Register(procedure string, handler Handler) {
	m[procedure] = handler
}

// GetHandler retrieves the Handler for the given Procedure or returns an
// UnknownProcedureErr.
func (m MapRegistry) GetHandler(procedure string) (Handler, error) {
	if h, ok := m[procedure]; ok {
		return h, nil
	}
	return nil, UnknownProcedureErr{procedure}
}
