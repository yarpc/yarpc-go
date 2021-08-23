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

<$thrift      := import "go.uber.org/yarpc/encoding/thrift">
<$transport   := import "go.uber.org/yarpc/api/transport">

<$contextImportPath   := .ContextImportPath>
<$onewayWrapperImport := .OnewayWrapperImport>
<$onewayWrapperFunc   := .OnewayWrapperFunc>
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
	<range .Functions>
		<$context := import $contextImportPath>
		<.Name>(
			ctx <$context>.Context, <range .Arguments>
			<.Name> <formatType .Type>,<end>
		)<if .OneWay> error
		<else if .ReturnType> (<formatType .ReturnType>, error)
		<else> error
		<end>
	<end>
}

<$module := .Module>

// New prepares an implementation of the <.Name> service for
// registration.
//
// 	handler := <.Name>Handler{}
// 	dispatcher.Register(<$pkgname>.New(handler))
func New(impl Interface, opts ...<$thrift>.RegisterOption) []<$transport>.Procedure {
	<if .Functions>h := handler{impl}<end>
	service := <$thrift>.Service{
		Name: "<.Name>",
		Methods: []<$thrift>.Method{
		<range .Functions>
			<$thrift>.Method{
				Name: "<.ThriftName>",
				HandlerSpec: <$thrift>.HandlerSpec{
				<if .OneWay>
					Type: <$transport>.Oneway,
					Oneway: <import $onewayWrapperImport>.<$onewayWrapperFunc>(h.<.Name>),
				<else>
					Type: <$transport>.Unary,
					Unary: <import $unaryWrapperImport>.<$unaryWrapperFunc>(h.<.Name>),
				<end>
					NoWire: <.Name>_NoWireHandler{impl},
				},
				Signature: "<.Name>(<range $i, $v := .Arguments><if ne $i 0>, <end><.Name> <formatType .Type><end>)<if not .OneWay | and .ReturnType> (<formatType .ReturnType>)<end>",
				ThriftModule: <import $module.ImportPath>.ThriftModule,
				},
		<end>},
	}

	procedures := make([]<$transport>.Procedure, 0, <len .Functions>)
	<if .Parent>
	procedures = append(
		procedures,
		<import .ParentServerPackagePath>.New(
			impl,
			append(
				opts,
				<$thrift>.Named(<printf "%q" .Name>),
			)...,
		)...,
	)
	<end ->
	procedures = append(procedures, <$thrift>.BuildProcedures(service, opts...)...)
	return procedures
}

<if .Functions>
type handler struct{ impl Interface }

<$yarpcerrors := import "go.uber.org/yarpc/yarpcerrors">

type yarpcErrorNamer interface { YARPCErrorName() string }

type yarpcErrorCoder interface { YARPCErrorCode() *yarpcerrors.Code }
<end>

<$service := .>
<$module := .Module>
<range .Functions>
<$context := import $contextImportPath>
<$prefix := printf "%s.%s_%s_" (import $module.ImportPath) $service.Name .Name>

<$wire := import "go.uber.org/thriftrw/wire">

<if .OneWay>
func (h handler) <.Name>(ctx <$context>.Context, body <$wire>.Value) error {
	var args <$prefix>Args
	if err := args.FromWire(body); err != nil {
		return err
	}

	return h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
}
<else>
<$yarpcerrors := import "go.uber.org/yarpc/yarpcerrors">
func (h handler) <.Name>(ctx <$context>.Context, body <$wire>.Value) (<$thrift>.Response, error) {
	var args <$prefix>Args
	if err := args.FromWire(body); err != nil {
		return <$thrift>.Response{}, <$yarpcerrors>.InvalidArgumentErrorf(
			"could not decode Thrift request for service '<$service.Name>' procedure '<.Name>': %w", err)
	}

	<if .ReturnType>
		success, appErr := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<else>
		appErr := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := appErr != nil
	result, err := <$prefix>Helper.WrapResponse(<if .ReturnType>success,<end> appErr)

	var response <$thrift>.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
		if namer, ok := appErr.(yarpcErrorNamer); ok {
			response.ApplicationErrorName = namer.YARPCErrorName()
 		}
		if extractor, ok := appErr.(yarpcErrorCoder); ok {
			response.ApplicationErrorCode = extractor.YARPCErrorCode()
		}
		if appErr != nil {
			response.ApplicationErrorDetails = appErr.Error()
		}
	}

	return response, err
}
<end>
<end>

<range .Functions>
<$context := import $contextImportPath>
<$yarpcerrors := import "go.uber.org/yarpc/yarpcerrors">
<$prefix := printf "%s.%s_%s_" (import $module.ImportPath) $service.Name .Name>

type <.Name>_NoWireHandler struct{ impl Interface }

func (h <.Name>_NoWireHandler) HandleNoWire(ctx <$context>.Context, nwc *<$thrift>.NoWireCall) (<$thrift>.NoWireResponse, error) {
	var (
		args <$prefix>Args
		<if not .OneWay>rw <import "go.uber.org/thriftrw/protocol/stream">.ResponseWriter<end>
		err error
	)

	<if .OneWay>
	if _, err = nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, &args); err != nil {
		return <$thrift>.NoWireResponse{}, <$yarpcerrors>.InvalidArgumentErrorf(
			"could not decode (via no wire) Thrift request for service '<$service.Name>' procedure '<.Name>': %w", err)
	}

	return <$thrift>.NoWireResponse{}, h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<else>
	rw, err = nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, &args)
	if err != nil {
		return <$thrift>.NoWireResponse{}, <$yarpcerrors>.InvalidArgumentErrorf(
			"could not decode (via no wire) Thrift request for service '<$service.Name>' procedure '<.Name>': %w", err)
	}

	<if .ReturnType>
	success, appErr := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<else>
	appErr := h.impl.<.Name>(ctx, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := appErr != nil
	result, err := <$prefix>Helper.WrapResponse(<if .ReturnType>success,<end> appErr)
	response := <$thrift>.NoWireResponse{ResponseWriter: rw}
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
		if namer, ok := appErr.(yarpcErrorNamer); ok {
			response.ApplicationErrorName = namer.YARPCErrorName()
		}
		if extractor, ok := appErr.(yarpcErrorCoder); ok {
			response.ApplicationErrorCode = extractor.YARPCErrorCode()
		}
		if appErr != nil {
			response.ApplicationErrorDetails = appErr.Error()
		}
	}
	return response, err
	<end>
}
<end>
`

func serverGenerator(data *serviceTemplateData, files map[string][]byte) (err error) {
	packageName := filepath.Base(data.ServerPackagePath())
	// kv.thrift => .../kv/keyvalueserver/server.go
	path := filepath.Join(data.Module.Directory, packageName, "server.go")
	files[path], err = plugin.GoFileFromTemplate(path, serverTemplate, data, templateOptions...)
	return
}
