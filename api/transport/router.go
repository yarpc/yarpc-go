// Copyright (c) 2025 Uber Technologies, Inc.
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
	"context"

	"go.uber.org/zap/zapcore"
)

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

// Procedure specifies a single handler registered in the RouteTable.
type Procedure struct {
	// Name of the procedure.
	Name string

	// Service or empty to use the default service name.
	Service string

	// HandlerSpec specifying which handler and rpc type.
	HandlerSpec HandlerSpec

	// Encoding of the handler (optional) used for introspection and routing
	// (if present).
	Encoding Encoding

	// Signature of the handler, for introspection. This should be a snippet of
	// Go code representing the function definition.
	Signature string
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (p Procedure) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// Passing a Procedure as a zap.ObjectMarshaler allocates, so we shouldn't
	// do it on the request path.
	enc.AddString("name", p.Name)
	enc.AddString("service", p.Service)
	enc.AddString("encoding", string(p.Encoding))
	enc.AddString("signature", p.Signature)
	return enc.AddObject("handler", p.HandlerSpec)
}

// Less orders procedures lexicographically on (Service, Name, Encoding).
func (p Procedure) Less(o Procedure) bool {
	if p.Service != o.Service {
		return p.Service < o.Service
	}
	if p.Name != o.Name {
		return p.Name < o.Name
	}
	return p.Encoding < o.Encoding
}

// Router maintains and provides access to a collection of procedures
type Router interface {
	// Procedures returns a list of procedures that
	// have been registered so far.
	Procedures() []Procedure

	// Choose decides a handler based on a context and transport request
	// metadata, or returns an UnrecognizedProcedureError if no handler exists
	// for the request.  This is the interface for use in inbound transports to
	// select a handler for a request.
	Choose(ctx context.Context, req *Request) (HandlerSpec, error)
}

// RouteTable is an mutable interface for a Router that allows Registering new
// Procedures
type RouteTable interface {
	Router

	// Registers zero or more procedures with the route table.
	Register([]Procedure)
}
