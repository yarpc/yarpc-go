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

package yarpc

import (
	"context"

	"go.uber.org/zap/zapcore"
)

// TODO: Until golang/mock#4 is fixed, imports in the generated code have to
// be fixed by hand. They use vendor/* import paths rather than direct.

// TransportProcedure specifies a single transport-level handler registered in the RouteTable.
type TransportProcedure struct {
	// Name of the procedure.
	Name string

	// Service or empty to use the default service name.
	Service string

	// HandlerSpec specifying which handler and rpc type.
	HandlerSpec TransportHandlerSpec

	// Encoding of the handler.
	// (if present).
	Encoding Encoding

	// Human-readable signature of the handler.
	Signature string
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (p TransportProcedure) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// Passing a TransportProcedure as a zap.ObjectMarshaler allocates, so we shouldn't
	// do it on the request path.
	enc.AddString("name", p.Name)
	enc.AddString("service", p.Service)
	enc.AddString("encoding", string(p.Encoding))
	enc.AddString("signature", p.Signature)
	return enc.AddObject("handler", p.HandlerSpec)
}

// Less orders procedures lexicographically on (Service, Name, Encoding).
func (p TransportProcedure) Less(o TransportProcedure) bool {
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
	Procedures() []TransportProcedure

	// Choose decides a handler based on a context and transport request
	// metadata, or returns an UnrecognizedProcedureError if no handler exists
	// for the request.  This is the interface for use in inbound transports to
	// select a handler for a request.
	Choose(ctx context.Context, req *Request) (TransportHandlerSpec, error)
}
