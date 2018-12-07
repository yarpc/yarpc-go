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

package yarpcthrift

import (
	"context"

	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/thriftreflect"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcprocedure"
)

// EncodingHandler is a convenience type alias for functions that act as Handlers.
type EncodingHandler func(context.Context, wire.Value) (envelope.Enveloper, error)

// Method represents a Thrift service method.
type Method struct {
	// Name of the method itself.
	Name string

	// The handler to call.
	Handler EncodingHandler

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
func BuildProcedures(s Service, opts ...RegisterOption) []yarpc.EncodingProcedure {
	var rc registerConfig
	for _, opt := range opts {
		opt.applyRegisterOption(&rc)
	}

	proto := protocol.Binary
	if rc.Protocol != nil {
		proto = rc.Protocol
	}

	rs := make([]yarpc.EncodingProcedure, 0, len(s.Methods))

	for _, method := range s.Methods {
		var spec yarpc.EncodingHandlerSpec
		spec = yarpc.NewUnaryEncodingHandlerSpec(unaryEncodingHandler{h: method.Handler})
		codec := func() yarpc.InboundCodec {
			return newCodec(proto, rc.Enveloping)
		}
		rs = append(rs, yarpc.EncodingProcedure{
			Name:        yarpcprocedure.ToName(s.Name, method.Name),
			HandlerSpec: spec,
			Encoding:    Encoding,
			Signature:   method.Signature,
			Codec:       codec,
		})
	}
	return rs
}
