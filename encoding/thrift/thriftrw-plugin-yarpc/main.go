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

// thriftrw-plugin-yarpc implements a plugin for ThriftRW that generates code
// compatible with YARPC.
//
// For more information, check the documentation of the parent package.
package main

import (
	"flag"
	"path/filepath"
	"strings"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
)

var (
	_context = flag.String("context-import-path",
		"context",
		"Import path at which Context is available")
	_unaryHandlerWrapper = flag.String("unary-handler-wrapper",
		"go.uber.org/yarpc/encoding/thrift.UnaryHandlerFunc",
		"Function used to wrap generic Thrift unary function handlers into YARPC handlers")
	_onewayHandlerWrapper = flag.String("oneway-handler-wrapper",
		"go.uber.org/yarpc/encoding/thrift.OnewayHandlerFunc",
		"Function used to wrap generic Thrift oneway function handlers into YARPC handlers")
)

const serverTemplate = `
// Code generated by thriftrw-plugin-yarpc
// @generated

<$pkgname := printf "%sserver" (lower .Service.Name)>
package <$pkgname>
<$yarpc     := import "go.uber.org/yarpc">
<$thrift    := import "go.uber.org/yarpc/encoding/thrift">
<$transport := import "go.uber.org/yarpc/api/transport">
<$context   := import .ContextImportPath>

// Interface is the server-side interface for the <.Service.Name> service.
type Interface interface {
	<if .Parent>
		<$parentPath := printf "%s/yarpc/%sserver" .ParentModule.ImportPath (lower .Parent.Name)>
		<import $parentPath>.Interface
	<end>

	<range .Service.Functions>
		<.Name>(
			ctx <$context>.Context,
			reqMeta <$yarpc>.ReqMeta, <range .Arguments>
			<.Name> <formatType .Type>,<end>
		)<if .OneWay> error
		<else if .ReturnType> (<formatType .ReturnType>, <$yarpc>.ResMeta, error)
		<else> (<$yarpc>.ResMeta, error)
		<end>
	<end>
}

// New prepares an implementation of the <.Service.Name> service for
// registration.
//
// 	handler := <.Service.Name>Handler{}
// 	dispatcher.Register(<$pkgname>.New(handler))
func New(impl Interface, opts ...<$thrift>.RegisterOption) []<$transport>.Procedure {
	h := handler{impl}
	service := <$thrift>.Service{
		Name: "<.Service.Name>",
			Methods: map[string]<$thrift>.UnaryHandler{
				<$unaryWrapperImport := .UnaryWrapperImport>
				<$unaryWrapperFunc := .UnaryWrapperFunc>
				<range .Service.Functions>
					<if not .OneWay>"<.ThriftName>": <import $unaryWrapperImport>.<$unaryWrapperFunc>(h.<.Name>),<end>
			<end>},
			OnewayMethods: map[string]<$thrift>.OnewayHandler{
				<$onewayWrapperImport := .OnewayWrapperImport>
				<$onewayWrapperFunc := .OnewayWrapperFunc>
				<range .Service.Functions>
					<if .OneWay>"<.ThriftName>": <import $onewayWrapperImport>.<$onewayWrapperFunc>(h.<.Name>),<end>
			<end>},
	}
	return <$thrift>.BuildProcedures(service, opts...)
}

type handler struct{ impl Interface }

<$service := .Service>
<$module := .Module>
<range .Service.Functions>
<$prefix := printf "%s.%s_%s_" (import $module.ImportPath) $service.Name .Name>

<$wire := import "go.uber.org/thriftrw/wire">

<if .OneWay>
func (h handler) <.Name>(
	ctx <$context>.Context,
	reqMeta <$yarpc>.ReqMeta,
	body <$wire>.Value,
) error {
	var args <$prefix>Args
	if err := args.FromWire(body); err != nil {
		return err
	}

	return h.impl.<.Name>(ctx, reqMeta, <range .Arguments>args.<.Name>,<end>)
}
<else>
func (h handler) <.Name>(
	ctx <$context>.Context,
	reqMeta <$yarpc>.ReqMeta,
	body <$wire>.Value,
) (<$thrift>.Response, error) {
	var args <$prefix>Args
	if err := args.FromWire(body); err != nil {
		return <$thrift>.Response{}, err
	}

	<if .ReturnType>
		success, resMeta, err := h.impl.<.Name>(ctx, reqMeta, <range .Arguments>args.<.Name>,<end>)
	<else>
		resMeta, err := h.impl.<.Name>(ctx, reqMeta, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := err != nil
	result, err := <$prefix>Helper.WrapResponse(<if .ReturnType>success,<end> err)

	var response <$thrift>.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
<end>
<end>
`

const clientTemplate = `
// Code generated by thriftrw-plugin-yarpc
// @generated

<$pkgname := printf "%sclient" (lower .Service.Name)>
package <$pkgname>

<$yarpc     := import "go.uber.org/yarpc">
<$transport := import "go.uber.org/yarpc/api/transport">
<$thrift    := import "go.uber.org/yarpc/encoding/thrift">
<$context   := import "context">

// Interface is a client for the <.Service.Name> service.
type Interface interface {
	<if .Parent>
		<$parentPath := printf "%s/yarpc/%sclient" .ParentModule.ImportPath (lower .Parent.Name)>
		<import $parentPath>.Interface
	<end>

	<range .Service.Functions>
		<.Name>(
			ctx <$context>.Context, <range .Arguments>
			<.Name> <formatType .Type>,<end>
			opts ...<$yarpc>.CallOption,
		)<if .OneWay> (<$yarpc>.Ack, error)
		<else if .ReturnType> (<formatType .ReturnType>, error)
		<else> error
		<end>
	<end>
}

</* TODO(abg): Pull the default routing name from a Thrift annotation? */>

// New builds a new client for the <.Service.Name> service.
//
// 	client := <$pkgname>.New(dispatcher.ClientConfig("<lower .Service.Name>"))
func New(c <$transport>.ClientConfig, opts ...<$thrift>.ClientOption) Interface {
	return client{c: <$thrift>.New(<$thrift>.Config{
		Service: "<.Service.Name>",
		ClientConfig: c,
	}, opts...)}
}

func init() {
	<$yarpc>.RegisterClientBuilder(func(c <$transport>.ClientConfig) Interface {
		return New(c)
	})
}

type client struct{ c <$thrift>.Client }

<$service := .Service>
<$module := .Module>
<range .Service.Functions>
<$prefix := printf "%s.%s_%s_" (import $module.ImportPath) $service.Name .Name>

func (c client) <.Name>(
	ctx <$context>.Context, <range .Arguments>
	_<.Name> <formatType .Type>,<end>
	opts ...<$yarpc>.CallOption,
<if .OneWay>) (<$yarpc>.Ack, error) {
	args := <$prefix>Helper.Args(<range .Arguments>_<.Name>, <end>)
	return c.c.CallOneway(ctx, args, opts...)
}
<else>) (<if .ReturnType>success <formatType .ReturnType>,<end> err error) {
	<$wire := import "go.uber.org/thriftrw/wire">
	args := <$prefix>Helper.Args(<range .Arguments>_<.Name>, <end>)

	var body <$wire>.Value
	body, err = c.c.Call(ctx, args, opts...)
	if err != nil {
		return
	}

	var result <$prefix>Result
	if err = result.FromWire(body); err != nil {
		return
	}

	<if .ReturnType>success, <end>err = <$prefix>Helper.UnwrapResponse(&result)
	return
}
<end>
<end>
`

var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
}

type generator struct{}

func (generator) Generate(req *api.GenerateServiceRequest) (*api.GenerateServiceResponse, error) {
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

		unaryWrapperImport, unaryWrapperFunc := splitFunctionPath(*_unaryHandlerWrapper)
		onewayWrapperImport, onewayWrapperFunc := splitFunctionPath(*_onewayHandlerWrapper)

		templateData := struct {
			Module       *api.Module
			Service      *api.Service
			Parent       *api.Service
			ParentModule *api.Module

			ContextImportPath   string
			UnaryWrapperImport  string
			UnaryWrapperFunc    string
			OnewayWrapperImport string
			OnewayWrapperFunc   string
		}{
			Module:       module,
			Service:      service,
			Parent:       parent,
			ParentModule: parentModule,

			ContextImportPath:   *_context,
			UnaryWrapperImport:  unaryWrapperImport,
			UnaryWrapperFunc:    unaryWrapperFunc,
			OnewayWrapperImport: onewayWrapperImport,
			OnewayWrapperFunc:   onewayWrapperFunc,
		}

		// kv.thrift => .../kv/yarpc
		baseDir := filepath.Join(module.Directory, "yarpc")

		serverPackageName := strings.ToLower(service.Name) + "server"
		clientPackageName := strings.ToLower(service.Name) + "client"

		// kv.thrift =>
		//   .../yarpc/keyvalueserver/server.go
		//   .../yarpc/keyvalueclient/client.go
		serverFilePath := filepath.Join(baseDir, serverPackageName, "server.go")
		clientFilePath := filepath.Join(baseDir, clientPackageName, "client.go")

		serverContents, err := plugin.GoFileFromTemplate(
			serverFilePath, serverTemplate, templateData, templateOptions...)
		if err != nil {
			return nil, err
		}

		clientContents, err := plugin.GoFileFromTemplate(
			clientFilePath, clientTemplate, templateData, templateOptions...)
		if err != nil {
			return nil, err
		}

		files[serverFilePath] = serverContents
		files[clientFilePath] = clientContents
	}
	return &api.GenerateServiceResponse{Files: files}, nil
}

func splitFunctionPath(input string) (string, string) {
	i := strings.LastIndex(input, ".")
	return input[:i], input[i+1:]
}

func main() {
	flag.Parse()
	plugin.Main(&plugin.Plugin{Name: "yarpc", ServiceGenerator: generator{}})
}
