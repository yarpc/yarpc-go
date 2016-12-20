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

// BuildRegistrants builds the transport.Registrants.
//
// Users should use the generated BuildRegistrants rather than using this directly.
func BuildRegistrants(serviceName string, serviceMethods map[string]UnaryHandler, opts ...RegisterOption) []transport.Registrant {
	registerConfig := &registerConfig{}
	for _, opt := range opts {
		opt.applyRegisterOption(registerConfig)
	}
	registrants := make([]transport.Registrant, 0, len(serviceMethods))
	for methodName, unaryHandler := range serviceMethods {
		registrants = append(
			registrants,
			transport.Registrant{
				Procedure: toProcedureName(serviceName, methodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(
					newTransportUnaryHandler(
						unaryHandler,
					),
				),
			},
		)
	}
	return registrants
}
