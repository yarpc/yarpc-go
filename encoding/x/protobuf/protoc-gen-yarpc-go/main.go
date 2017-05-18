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

/*
Package main provides a protoc plugin that generates code for the protobuf encoding for YARPC.

To use:
	go get github.com/gogo/protobuf/protoc-gen-gogoslick
	go get go.uber.org/yarpc/encoding/x/protobuf/protoc-gen-yarpc-go
	protoc --gogoslick_out=. foo.proto
	protoc --yarpc-go_out=. foo.proto
*/
package main

// TODO: there is some crazy bug with protobuf that if you declare:
//
//   func bar() (kv.GetValueResponse, error) {
//     return nil, errors.New("nil response and non-nil error")
//   }
//
//   func foo() (proto.Message, error) {
//     return bar()
//   }
//
//   response, err := foo()
//   fmt.Printf("%v %v\n", response, response == nil)
//
// This will print "<nil>, false". If you try to do something with response (ie call a function on it),
// if will panic because response is nil. Something similar happens in golang/protobuf
// too, so this is insane. If in bar(), you do response == nil, it will be true.
// The generated code handles this.

import (
	"fmt"
	"log"
	"strings"
	"text/template"

	"go.uber.org/yarpc/internal/protoplugin"
)

const tmpl = `{{$packagePath := .GoPackage.Path}}
// Code generated by protoc-gen-yarpc-go
// source: {{.GetName}}
// DO NOT EDIT!

package {{.GoPackage.Name}}

import (
	{{range $i := .Imports}}{{if $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}

	{{range $i := .Imports}}{{if not $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}
)

{{range $service := .Services }}
// {{$service.GetName}}YarpcClient is the yarpc client-side interface for the {{$service.GetName}} service.
type {{$service.GetName}}YarpcClient interface {
	{{range $method := unaryMethods $service}}{{$method.GetName}}(context.Context, *{{$method.RequestType.GoType $packagePath}}, ...yarpc.CallOption) (*{{$method.ResponseType.GoType $packagePath}}, error)
	{{end}}
	{{range $method := onewayMethods $service}}{{$method.GetName}}(context.Context, *{{$method.RequestType.GoType $packagePath}}, ...yarpc.CallOption) (yarpc.Ack, error)
	{{end}}
}

// New{{$service.GetName}}YarpcClient builds a new yarpc client for the {{$service.GetName}} service.
func New{{$service.GetName}}YarpcClient(clientConfig transport.ClientConfig) {{$service.GetName}}YarpcClient {
	return &_{{$service.GetName}}YarpcCaller{protobuf.NewClient("{{trimPrefixPeriod $service.FQSN}}", clientConfig)}
}

// {{$service.GetName}}YarpcServer is the yarpc server-side interface for the {{$service.GetName}} service.
type {{$service.GetName}}YarpcServer interface {
	{{range $method := unaryMethods $service}}{{$method.GetName}}(context.Context, *{{$method.RequestType.GoType $packagePath}}) (*{{$method.ResponseType.GoType $packagePath}}, error)
	{{end}}
	{{range $method := onewayMethods $service}}{{$method.GetName}}(context.Context, *{{$method.RequestType.GoType $packagePath}}) error
	{{end}}
}

// Build{{$service.GetName}}YarpcProcedures prepares an implementation of the {{$service.GetName}} service for yarpc registration.
func Build{{$service.GetName}}YarpcProcedures(server {{$service.GetName}}YarpcServer) []transport.Procedure {
	handler := &_{{$service.GetName}}YarpcHandler{server}
	return protobuf.BuildProcedures(
		"{{trimPrefixPeriod $service.FQSN}}",
		map[string]transport.UnaryHandler{
		{{range $method := unaryMethods $service}}"{{$method.GetName}}": protobuf.NewUnaryHandler(handler.{{$method.GetName}}, new{{$service.GetName}}_{{$method.GetName}}YarpcRequest),
		{{end}}
		},
		map[string]transport.OnewayHandler{
		{{range $method := onewayMethods $service}}"{{$method.GetName}}": protobuf.NewOnewayHandler(handler.{{$method.GetName}}, new{{$service.GetName}}_{{$method.GetName}}YarpcRequest),
		{{end}}
		},
	)
}

type _{{$service.GetName}}YarpcCaller struct {
	client protobuf.Client
}

{{range $method := unaryMethods $service}}
func (c *_{{$service.GetName}}YarpcCaller) {{$method.GetName}}(ctx context.Context, request *{{$method.RequestType.GoType $packagePath}}, options ...yarpc.CallOption) (*{{$method.ResponseType.GoType $packagePath}}, error) {
	responseMessage, err := c.client.Call(ctx, "{{$method.GetName}}", request, new{{$service.GetName}}_{{$method.GetName}}YarpcResponse, options...)
	if responseMessage == nil {
		return nil, err
	}
	response, ok := responseMessage.(*{{$method.ResponseType.GoType $packagePath}})
	if !ok {
		return nil, protobuf.CastError(empty{{$service.GetName}}_{{$method.GetName}}YarpcResponse, responseMessage)
	}
	return response, err
}
{{end}}
{{range $method := onewayMethods $service}}
func (c *_{{$service.GetName}}YarpcCaller) {{$method.GetName}}(ctx context.Context, request *{{$method.RequestType.GoType $packagePath}}, options ...yarpc.CallOption) (yarpc.Ack, error) {
	return c.client.CallOneway(ctx, "{{$method.GetName}}", request, options...)
}
{{end}}

type _{{$service.GetName}}YarpcHandler struct {
	server {{$service.GetName}}YarpcServer
}

{{range $method := unaryMethods $service}}
func (h *_{{$service.GetName}}YarpcHandler) {{$method.GetName}}(ctx context.Context, requestMessage proto.Message) (proto.Message, error) {
	var request *{{$method.RequestType.GoType $packagePath}}
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*{{$method.RequestType.GoType $packagePath}})
		if !ok {
			return nil, protobuf.CastError(empty{{$service.GetName}}_{{$method.GetName}}YarpcRequest, requestMessage)
		}
	}
	response, err := h.server.{{$method.GetName}}(ctx, request)
	if response == nil {
		return nil, err
	}
	return response, err
}
{{end}}
{{range $method := onewayMethods $service}}
func (h *_{{$service.GetName}}YarpcHandler) {{$method.GetName}}(ctx context.Context, requestMessage proto.Message) error {
	var request *{{$method.RequestType.GoType $packagePath}}
	var ok bool
	if requestMessage != nil {
		request, ok = requestMessage.(*{{$method.RequestType.GoType $packagePath}})
		if !ok {
			return protobuf.CastError(empty{{$service.GetName}}_{{$method.GetName}}YarpcRequest, requestMessage)
		}
	}
	return h.server.{{$method.GetName}}(ctx, request)
}
{{end}}

{{range $method := $service.Methods}}
func new{{$service.GetName}}_{{$method.GetName}}YarpcRequest() proto.Message {
	return &{{$method.RequestType.GoType $packagePath}}{}
}

func new{{$service.GetName}}_{{$method.GetName}}YarpcResponse() proto.Message {
	return &{{$method.ResponseType.GoType $packagePath}}{}
}
{{end}}
var (
{{range $method := $service.Methods}}
	empty{{$service.GetName}}_{{$method.GetName}}YarpcRequest = &{{$method.RequestType.GoType $packagePath}}{}
	empty{{$service.GetName}}_{{$method.GetName}}YarpcResponse = &{{$method.ResponseType.GoType $packagePath}}{}{{end}}
)
{{end}}
`

var runner = protoplugin.NewRunner(
	template.Must(template.New("tmpl").Funcs(
		template.FuncMap{
			"unaryMethods":     unaryMethods,
			"onewayMethods":    onewayMethods,
			"trimPrefixPeriod": trimPrefixPeriod,
		}).Parse(tmpl)),
	checkTemplateInfo,
	[]string{
		"context",
		"github.com/gogo/protobuf/proto",
		"go.uber.org/yarpc",
		"go.uber.org/yarpc/api/transport",
		"go.uber.org/yarpc/encoding/x/protobuf",
	},
	"pb.yarpc.go",
)

func main() {
	if err := protoplugin.Do(runner); err != nil {
		log.Fatal(err)
	}
}

func checkTemplateInfo(templateInfo *protoplugin.TemplateInfo) error {
	for _, service := range templateInfo.Services {
		for _, method := range service.Methods {
			if method.GetClientStreaming() || method.GetServerStreaming() {
				return fmt.Errorf("yarpc does not support streaming methods and %s:%s is a streaming method", service.GetName(), method.GetName())
			}
		}
	}
	return nil
}

func unaryMethods(service *protoplugin.Service) ([]*protoplugin.Method, error) {
	methods := make([]*protoplugin.Method, 0, len(service.Methods))
	for _, method := range service.Methods {
		if !method.GetClientStreaming() && !method.GetServerStreaming() && method.ResponseType.FQMN() != ".uber.yarpc.Oneway" {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

func onewayMethods(service *protoplugin.Service) ([]*protoplugin.Method, error) {
	methods := make([]*protoplugin.Method, 0, len(service.Methods))
	for _, method := range service.Methods {
		if !method.GetClientStreaming() && !method.GetServerStreaming() && method.ResponseType.FQMN() == ".uber.yarpc.Oneway" {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

func trimPrefixPeriod(s string) string {
	return strings.TrimPrefix(s, ".")
}
