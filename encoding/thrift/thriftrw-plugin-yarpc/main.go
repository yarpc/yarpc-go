// Copyright (c) 2020 Uber Technologies, Inc.
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

// thriftrw-plugin-yarpc implements a plugin for ThriftRW that generates code
// compatible with YARPC.
//
// thriftrw-plugin-yarpc supports "rpc.code" annotations on Thrift exceptions.
// For example:
//
//  exception ExceptionWithCode {
//    1: required string val
//  } (
//    rpc.code = "INVALID_ARGUMENT"
//  )
//
// The "rpc.code" annotation can be any code matching the string name of gRPC
// status enum codes. YARPC error codes match 1-1 with these codes, however gRPC
// uses a different string name representation. We choose to use the raw gRPC
// enum code names instead to ensure cross-language compatibility with other
// languages, such as Java.
//  - https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
//
// Available string names method:
//  - "CANCELLED"
//  - "UNKNOWN"
//  - "INVALID_ARGUMENT"
//  - "DEADLINE_EXCEEDED"
//  - "NOT_FOUND"
//  - "ALREADY_EXISTS"
//  - "PERMISSION_DENIED"
//  - "RESOURCE_EXHAUSTED"
//  - "FAILED_PRECONDITION"
//  - "ABORTED"
//  - "OUT_OF_RANGE"
//  - "UNIMPLEMENTED"
//  - "INTERNAL"
//  - "UNAVAILABLE"
//  - "DATA_LOSS"
//  - "UNAUTHENTICATED"
//
// Adding codes will affect YARPC's observability middleware classification of
// client and server errors for Thrift exceptions.
//
// For more information on the Thrift encoding, check the documentation of the
// parent package.
package main

import (
	"flag"
	"strings"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
)

// Command line flags
var (
	_context = flag.String("context-import-path",
		"context",
		"Import path at which Context is available")
	_unaryHandlerWrapper = flag.String("unary-handler-wrapper",
		"go.uber.org/yarpc/encoding/thrift.UnaryHandler",
		"Function used to wrap generic Thrift unary function handlers into YARPC handlers")
	_onewayHandlerWrapper = flag.String("oneway-handler-wrapper",
		"go.uber.org/yarpc/encoding/thrift.OnewayHandler",
		"Function used to wrap generic Thrift oneway function handlers into YARPC handlers")
	_noGomock = flag.Bool("no-gomock", false,
		"Don't generate gomock mocks for service clients")
	_noFx             = flag.Bool("no-fx", false, "Don't generate Fx module")
	_sanitizeTChannel = flag.Bool("sanitize-tchannel", false, "Enable tchannel context sanitization")
)

type g struct {
	SanitizeTChannel bool
}

func (g g) Generate(req *api.GenerateServiceRequest) (*api.GenerateServiceResponse, error) {
	generators := []genFunc{clientGenerator, serverGenerator, yarpcErrorGenerator}
	if !*_noFx {
		generators = append(generators, fxGenerator)
	}
	if !*_noGomock {
		generators = append(generators, gomockGenerator)
	}

	unaryWrapperImport, unaryWrapperFunc := splitFunctionPath(*_unaryHandlerWrapper)
	onewayWrapperImport, onewayWrapperFunc := splitFunctionPath(*_onewayHandlerWrapper)

	files := make(map[string][]byte)
	for _, serviceID := range req.RootServices {
		svc := buildSvc(serviceID, req)
		data := templateData{
			Svc:                 svc,
			ContextImportPath:   *_context,
			UnaryWrapperImport:  unaryWrapperImport,
			UnaryWrapperFunc:    unaryWrapperFunc,
			OnewayWrapperImport: onewayWrapperImport,
			OnewayWrapperFunc:   onewayWrapperFunc,
			SanitizeTChannel:    g.SanitizeTChannel,
		}

		for _, gen := range generators {
			if err := gen(&data, files); err != nil {
				return nil, err
			}
		}
	}
	return &api.GenerateServiceResponse{Files: files}, nil
}

func splitFunctionPath(input string) (string, string) {
	i := strings.LastIndex(input, ".")
	return input[:i], input[i+1:]
}

func buildSvc(serviceID api.ServiceID, req *api.GenerateServiceRequest) *Svc {
	service := req.Services[serviceID]
	module := req.Modules[service.ModuleID]

	var parents []*Svc
	if service.ParentID != nil {
		parentSvc := buildSvc(*service.ParentID, req)
		parents = append(parents, parentSvc)
		parents = append(parents, parentSvc.Parents...)
	}

	return &Svc{
		Service: service,
		Module:  module,
		Parents: parents,
	}
}

func main() {
	flag.Parse()
	plugin.Main(&plugin.Plugin{Name: "yarpc", ServiceGenerator: g{
		SanitizeTChannel: *_sanitizeTChannel,
	}})
}
