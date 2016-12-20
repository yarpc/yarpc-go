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

package protobuf

import "go.uber.org/yarpc/api/transport"

// BuildProcedures builds the transport.Procedures.
//
// Users should use the generated BuildProcedures rather than using this directly.
func BuildProcedures(serviceName string, serviceMethods map[string]UnaryHandler, opts ...RegisterOption) []transport.Procedure {
	registerConfig := &registerConfig{}
	for _, opt := range opts {
		opt.applyRegisterOption(registerConfig)
	}
	procedures := make([]transport.Procedure, 0, len(serviceMethods))
	for methodName, unaryHandler := range serviceMethods {
		procedures = append(
			procedures,
			transport.Procedure{
				Name: toProcedureName(serviceName, methodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(
					newTransportUnaryHandler(
						unaryHandler,
					),
				),
			},
		)
	}
	return procedures
}
