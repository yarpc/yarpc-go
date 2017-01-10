// Copyright (c) 2017 Uber Technologies, Inc.
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
// For more information, check the documentation of the parent package.
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
)

type g struct{}

func (g) Generate(req *api.GenerateServiceRequest) (*api.GenerateServiceResponse, error) {
	generators := []genFunc{clientGenerator, serverGenerator}

	unaryWrapperImport, unaryWrapperFunc := splitFunctionPath(*_unaryHandlerWrapper)
	onewayWrapperImport, onewayWrapperFunc := splitFunctionPath(*_onewayHandlerWrapper)

	files := make(map[string][]byte)
	for _, serviceID := range req.RootServices {
		service := req.Services[serviceID]
		module := req.Modules[service.ModuleID]

		var (
			parent       *api.Service
			parentModule *api.Module
		)
		if service.ParentID != nil {
			parent = req.Services[*service.ParentID]
			parentModule = req.Modules[parent.ModuleID]
		}

		data := templateData{
			ContextImportPath:   *_context,
			UnaryWrapperImport:  unaryWrapperImport,
			UnaryWrapperFunc:    unaryWrapperFunc,
			OnewayWrapperImport: onewayWrapperImport,
			OnewayWrapperFunc:   onewayWrapperFunc,
			Module:              module,
			Service:             service,
			Parent:              parent,
			ParentModule:        parentModule,
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

func main() {
	flag.Parse()
	plugin.Main(&plugin.Plugin{Name: "yarpc", ServiceGenerator: g{}})
}
