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
	"github.com/gogo/protobuf/proto"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcprocedure"
)

// UnaryProceduresParams contains the parameters for constructing unary Procedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
type UnaryProceduresParams struct {
	Service string
	Unary   []UnaryProcedure
}

// UnaryProcedure contains the parameters for constructing Unary procedures.
type UnaryProcedure struct {
	Method      string
	Handler     yarpc.UnaryEncodingHandler
	RequestType func() proto.Message
}

// StreamProceduresParams contains the parameters for constructing stream Procedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
type StreamProceduresParams struct {
	Service string
	Stream  []StreamProcedure
}

// StreamProcedure contains the parameters for constructing Stream procedures.
type StreamProcedure struct {
	Method  string
	Handler yarpc.StreamTransportHandler
}

// UnaryProcedures builds a slice of yarpc.EncodingProcedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
func UnaryProcedures(params UnaryProceduresParams) []yarpc.EncodingProcedure {
	procedures := make([]yarpc.EncodingProcedure, 0, 2*len(params.Unary))
	for _, u := range params.Unary {
		u := u // needed in the closure
		procedures = append(
			procedures,
			yarpc.EncodingProcedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewUnaryEncodingHandlerSpec(u.Handler),
				Encoding:    Encoding,
				Codec: func() yarpc.InboundCodec {
					return newProtoCodec(u.RequestType)
				},
			},
			yarpc.EncodingProcedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewUnaryEncodingHandlerSpec(u.Handler),
				Encoding:    yarpcjson.Encoding,
				Codec: func() yarpc.InboundCodec {
					return newJSONCodec(u.RequestType)
				},
			},
		)
	}
	return procedures
}

// StreamProcedures builds a slice of yarpc.TransportProcedures.
// This is used to construct a slice of yarpc.Procedures from the context
// of a protoc plugin, i.e. protoc-gen-yarpc-go.
func StreamProcedures(params StreamProceduresParams) []yarpc.TransportProcedure {
	procedures := make([]yarpc.TransportProcedure, 0, 2*len(params.Stream))
	for _, u := range params.Stream {
		procedures = append(
			procedures,
			yarpc.TransportProcedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewStreamTransportHandlerSpec(u.Handler),
				Encoding:    Encoding,
			},
			yarpc.TransportProcedure{
				Name:        yarpcprocedure.ToName(params.Service, u.Method),
				HandlerSpec: yarpc.NewStreamTransportHandlerSpec(u.Handler),
				Encoding:    yarpcjson.Encoding,
			},
		)
	}
	return procedures
}
