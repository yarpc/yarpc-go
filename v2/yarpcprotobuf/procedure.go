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

package yarpcprotobuf

import (
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcprocedure"
)

// ProceduresParams contains the parameters for constructing Procedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
type ProceduresParams struct {
	Service string
	Unary   []UnaryProceduresParams
	Stream  []StreamProceduresParams
}

// UnaryProceduresParams contains the parameters for constructing Unary procedures.
type UnaryProceduresParams struct {
	Method  string
	Handler yarpc.UnaryTransportHandler
}

// StreamProceduresParams contains the parameters for constructing Stream procedures.
type StreamProceduresParams struct {
	Method  string
	Handler yarpc.StreamTransportHandler
}

// Procedures builds a slice of yarpc.Procedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
func Procedures(params ProceduresParams) []yarpc.Procedure {
	procedures := make([]yarpc.Procedure, 0, 2*(len(params.Unary)+len(params.Stream)))
	for _, u := range params.Unary {
		procedures = append(
			procedures,
			yarpc.Procedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(u.Handler),
				Encoding:    Encoding,
			},
			yarpc.Procedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(u.Handler),
				Encoding:    yarpcjson.Encoding,
			},
		)
	}
	for _, s := range params.Stream {
		procedures = append(
			procedures,
			yarpc.Procedure{
				Name:        yarpcprocedure.ToName(params.Service, s.Method),
				HandlerSpec: yarpc.NewStreamTransportHandlerSpec(s.Handler),
				Encoding:    Encoding,
			},
			yarpc.Procedure{
				Name:        yarpcprocedure.ToName(params.Service, s.Method),
				HandlerSpec: yarpc.NewStreamTransportHandlerSpec(s.Handler),
				Encoding:    yarpcjson.Encoding,
			},
		)
	}
	return procedures
}
