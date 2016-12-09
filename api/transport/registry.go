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

import "context"

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

//go:generate mockgen -destination=transporttest/register.go -package=transporttest go.uber.org/yarpc/api/transport Registry,Registrar

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

	// HandlerSpec specifiying which handler and rpc type.
	HandlerSpec HandlerSpec
}

// Registry maintains and provides access to a collection of procedures and
// their handlers.
type Registry interface {
	// ServiceProcedures returns a list of services and their procedures that
	// have been registered so far.
	ServiceProcedures() []ServiceProcedure

	// Choose decides a handler based on a context and transport request
	// metadata, or returns an UnrecognizedProcedureError if no handler exists
	// for the request.  This is the interface for use in inbound transports to
	// select a handler for a request.
	Choose(ctx context.Context, req *Request) (HandlerSpec, error)
}

// Registrar provides access to a collection of procedures and their handlers.
type Registrar interface {
	Registry

	// Registers zero or more registrants with the registry.
	Register([]Registrant)
}
