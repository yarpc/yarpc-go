// Copyright (c) 2017 Uber Technologies, Inc.
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
	"fmt"
	"sort"

	"go.uber.org/yarpc/api/transport"
)

var (
	_ transport.Router = (*MapRouter)(nil)
)

type serviceProcedure struct {
	service   string
	procedure string
}

type serviceProcedureEncoding struct {
	service   string
	procedure string
	encoding  transport.Encoding
}

// MapRouter is a Router that maintains a map of the registered
// procedures.
type MapRouter struct {
	defaultService            string
	serviceProcedures         map[serviceProcedure]transport.Procedure
	serviceProcedureEncodings map[serviceProcedureEncoding]transport.Procedure
}

// NewMapRouter builds a new MapRouter that uses the given name as the
// default service name.
func NewMapRouter(defaultService string) MapRouter {
	return MapRouter{
		defaultService:            defaultService,
		serviceProcedures:         make(map[serviceProcedure]transport.Procedure),
		serviceProcedureEncodings: make(map[serviceProcedureEncoding]transport.Procedure),
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

		if r.Encoding != "" {
			// Route to individual handlers for unique combinations of service,
			// procedure, and encoding. This shall henceforth be the
			// recommended way for models to register procedures.
			spe := serviceProcedureEncoding{
				service:   r.Service,
				procedure: r.Name,
				encoding:  r.Encoding,
			}
			m.serviceProcedureEncodings[spe] = r
		}

		// This supports wild card encodings (for backward compatibility,
		// since type models like Thrift were not previously required to
		// specify the encoding of every procedure)
		sp := serviceProcedure{
			service:   r.Service,
			procedure: r.Name,
		}
		// Protect against registering ambiguous wildcards
		if _, ok := m.serviceProcedures[sp]; r.Encoding == "" && ok {
			panic(fmt.Sprintf("Cannot register multiple procedures with the same name and service without specifying alternate encodings, got ambiguous procedures for (%q, %q)", sp.service, sp.procedure))
		}
		m.serviceProcedures[sp] = r

	}
}

// Procedures returns a list procedures that
// have been registered so far.
func (m MapRouter) Procedures() []transport.Procedure {
	procs := make([]transport.Procedure, 0, len(m.serviceProcedures)+len(m.serviceProcedureEncodings))
	for _, v := range m.serviceProcedureEncodings {
		procs = append(procs, v)
	}
	for _, v := range m.serviceProcedures {
		procs = append(procs, v)
	}
	sort.Sort(proceduresByServiceProcedure(procs))
	return procs
}

// Choose retrives the HandlerSpec for the service, procedure, and encoding
// noted on the transport request, or returns an unrecognized procedure error
// (testable with transport.IsUnrecognizedProcedureError(err)).
func (m MapRouter) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	service, procedure, encoding := req.Service, req.Procedure, req.Encoding
	if service == "" {
		service = m.defaultService
	}

	// Fully specified combinations of service, procedure, and encoding shadow
	// and precede less specific combinations with an encoding wild card.
	spe := serviceProcedureEncoding{
		service:   service,
		procedure: procedure,
		encoding:  encoding,
	}
	if procedure, ok := m.serviceProcedureEncodings[spe]; ok {
		return procedure.HandlerSpec, nil
	}

	// Fall back to the original behavior for backward compatibility: route all
	// encodings to the same procedure, if a model specifies a handler
	// generically.
	sp := serviceProcedure{
		service:   service,
		procedure: procedure,
	}
	if procedure, ok := m.serviceProcedures[sp]; ok {
		return procedure.HandlerSpec, nil
	}

	return transport.HandlerSpec{}, transport.UnrecognizedProcedureError(req)
}

type proceduresByServiceProcedure []transport.Procedure

func (sp proceduresByServiceProcedure) Len() int {
	return len(sp)
}

func (sp proceduresByServiceProcedure) Less(i int, j int) bool {
	return sp[i].Less(sp[j])
}

func (sp proceduresByServiceProcedure) Swap(i int, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}
