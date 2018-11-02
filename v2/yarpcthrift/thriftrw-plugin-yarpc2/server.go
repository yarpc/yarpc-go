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

package main

import (
	"path/filepath"

	"go.uber.org/thriftrw/plugin"
)

const serverTemplate = `
// Code generated by thriftrw-plugin-yarpc
// @generated

<$pkgname := printf "%sserver" (lower .Name)>
package <$pkgname>

<$yarpc     := import "go.uber.org/yarpc/v2">
<$yarpcthrift    := import "go.uber.org/yarpc/v2/yarpcthrift">

<$contextImportPath   := .ContextImportPath>
<$unaryWrapperImport  := .UnaryWrapperImport>
<$unaryWrapperFunc    := .UnaryWrapperFunc>

</* Note that we import things like "context" inside loops rather than at the
    top-level because they will end up unused if the service does not have any
    functions.
 */>

// Interface is the server-side interface for the <.Name> service.
type Interface interface {
	<if .Parent><import .ParentServerPackagePath>.Interface
	<end>
	<range .Functions><if not .OneWay>
		<$context := import $contextImportPath>
		<.Name>(
			ctx <$context>.Context, <range .Arguments>
			<.Name> <formatType .Type>,<end>
		)<if .ReturnType> (<formatType .ReturnType>, error)
		<else> error
		<end>
	<end><end>
}

<$module := .Module>

// New prepares an implementation of the <.Name> service for
// registration.
//
// 	handler := <.Name>Handler{}
// 	dispatcher.Register(<$pkgname>.New(handler))
func New(impl Interface, opts ...<$yarpcthrift>.RegisterOption) []<$yarpc>.TransportProcedure {
	<if .Functions>h := handler{impl}<end>
	service := <$yarpcthrift>.Service{
		Name: "<.Name>",
		Methods: []<$yarpcthrift>.Method{
		<range .Functions><if not .OneWay>
			<$yarpcthrift>.Method{
				Name: "<.ThriftName>",
				Handler: <import $unaryWrapperImport>.<$unaryWrapperFunc>(h.<.Name>),
				Signature: "<.Name>(<range $i, $v := .Arguments><if ne $i 0>, <end><.Name> <formatType .Type><end>)<if .ReturnType> (<formatType .ReturnType>)<end>",
				ThriftModule: <import $module.ImportPath>.ThriftModule,
				},
		<end><end>},
	}

	procedures := make([]<$yarpc>.TransportProcedure, 0, <len .Functions>)
	<if .Parent> procedures = append(procedures, <import .ParentServerPackagePath>.New(impl, opts...)...)
	<end>         procedures = append(procedures, <$yarpcthrift>.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }

<$service := .>
<$module := .Module>
<range .Functions><if not .OneWay>
<$context := import $contextImportPath>
<$prefix := printf "%s.%s_%s_" (import $module.ImportPath) $service.Name .Name>

<$wire := import "go.uber.org/thriftrw/wire">

func (h handler) <.Name>(ctx <$context>.Context, body <$wire>.Value) (<$yarpcthrift>.Response, error) {
	var args <$prefix>Args
	if err := args.FromWire(body); err != nil {
		return <$yarpcthrift>.Response{}, err
	}

	<if .ReturnType>
		success, err := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<else>
		err := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := err != nil
	result, err := <$prefix>Helper.WrapResponse(<if .ReturnType>success,<end> err)

	var response <$yarpcthrift>.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
	}
	return response, err
}
<end><end>
`

func serverGenerator(data *templateData, files map[string][]byte) (err error) {
	packageName := filepath.Base(data.ServerPackagePath())
	// kv.thrift => .../kv/keyvalueserver/server.go
	path := filepath.Join(data.Module.Directory, packageName, "server.go")
	files[path], err = plugin.GoFileFromTemplate(path, serverTemplate, data, templateOptions...)
	return
}
