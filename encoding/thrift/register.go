// Copyright (c) 2021 Uber Technologies, Inc.
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

package thrift

import (
	"context"
	"fmt"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/thriftreflect"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/procedure"
)

// Register calls the RouteTable's Register method.
//
// This function exists for backwards compatibility only. It will be removed
// in a future version.
//
// Deprecated: Use the RouteTable's Register method directly.
func Register(r transport.RouteTable, rs []transport.Procedure) {
	r.Register(rs)
}

// UnaryHandler is a convenience type alias for functions that act as Handlers.
type UnaryHandler func(context.Context, wire.Value) (Response, error)

// OnewayHandler is a convenience type alias for functions that act as OnewayHandlers.
type OnewayHandler func(context.Context, wire.Value) error

// UnaryStreamHandler is a convenience type alias for functions that act as StreamHandlers.
type UnaryStreamHandler func(context.Context, stream.Reader, stream.Writer) (Response, error)

// OnewayStreamHandler is a convenience type alias for functions that act as OnewayStreamHandlers.
type OnewayStreamHandler func(context.Context, stream.Reader) error

// HandlerSpec represents the handler behind a Thrift service method.
type HandlerSpec struct {
	Type   transport.Type
	Unary  UnaryHandler
	Oneway OnewayHandler
}

// Method represents a Thrift service method.
type Method struct {
	// Name of the method itself.
	Name string

	// The handler to call.
	HandlerSpec HandlerSpec

	// Snippet of Go code representing the function definition of the handler.
	// This is useful for introspection.
	Signature string

	// ThriftModule, if non-nil, refers to the Thrift module from where this
	// method is coming from.
	ThriftModule *thriftreflect.ThriftModule
}

// Service is a generic Thrift service implementation.
type Service struct {
	// Name of the Thrift service. This is the name specified for the service
	// in the IDL.
	Name    string
	Methods []Method
}

// BuildProcedures builds a list of Procedures from a Thrift service
// specification.
func BuildProcedures(s Service, opts ...RegisterOption) []transport.Procedure {
	var rc registerConfig
	for _, opt := range opts {
		opt.applyRegisterOption(&rc)
	}

	proto := protocol.Binary
	if rc.Protocol != nil {
		proto = rc.Protocol
	}

	svc := s.Name
	if rc.ServiceName != "" {
		svc = rc.ServiceName
	}

	rs := make([]transport.Procedure, 0, len(s.Methods))

	for _, method := range s.Methods {
		var spec transport.HandlerSpec
		switch method.HandlerSpec.Type {
		case transport.Unary:
			spec = transport.NewUnaryHandlerSpec(thriftUnaryHandler{
				UnaryHandler: method.HandlerSpec.Unary,
				Protocol:     proto,
				Enveloping:   rc.Enveloping,
			})
		case transport.Oneway:
			spec = transport.NewOnewayHandlerSpec(thriftOnewayHandler{
				OnewayHandler: method.HandlerSpec.Oneway,
				Protocol:      proto,
				Enveloping:    rc.Enveloping,
			})
		default:
			panic(fmt.Sprintf("Invalid handler type for %T", method))
		}

		rs = append(rs, transport.Procedure{
			Name:        procedure.ToName(svc, method.Name),
			HandlerSpec: spec,
			Encoding:    Encoding,
			Signature:   method.Signature,
		})
	}
	return rs
}
