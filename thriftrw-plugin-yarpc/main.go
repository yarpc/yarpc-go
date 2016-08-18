package main

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/thriftrw/thriftrw-go/plugin"
	"github.com/thriftrw/thriftrw-go/plugin/api"
)

const serverTemplate = `
<$pkgname := printf "%sserver" (lower .Service.Name)>
package <$pkgname>

<$yarpc 	 := import "github.com/yarpc/yarpc-go">
<$thrift	 := import "github.com/yarpc/yarpc-go/encoding/thrift">
<$protocol := import "github.com/thriftrw/thriftrw-go/protocol">

// Interface is the server-side interface for the <.Service.Name> service.
type Interface interface {
	<if .Parent>
		<$parentPath := printf "%s/yarpc/%sserver" .Parent.Package (lower .Parent.Name)>
		<import $parentPath>.Interface
	<end>

	<range .Service.Functions>
		<.Name>(
			reqMeta <$yarpc>.ReqMeta, <range .Arguments>
			<lowerFirst .Name> <formatType .Type>,<end>
		) <if .ReturnType> (<formatType .ReturnType>, <$yarpc>.ResMeta, error)
		<else> (<$yarpc>.ResMeta, error)
		<end>
	<end>
}

// New prepares an implementation of the <.Service.Name> service for
// registration.
//
// 	handler := <.Service.Name>Handler{}
// 	thrift.Register(dispatcher, <$pkgname>.New(handler))
func New(impl Interface) <$thrift>.Service {
	return service{handler{impl}}
}

type service struct{ h handler }

func (service) Name() string {
	return "<.Service.Name>"
}

func (service) Protocol() <$protocol>.Protocol {
	return <$protocol>.Binary
}

func (s service) Handlers() map[string]<$thrift>.Handler {
	return map[string]<$thrift>.Handler{<range .Service.Functions>
			"<.ThriftName>": <$thrift>.HandlerFunc(s.h.<.Name>),
	<end>}
}

type handler struct{ impl Interface }

<$service := .Service>
<range .Service.Functions>

<$servicePackage := import $service.Package>
<$wire := import "github.com/thriftrw/thriftrw-go/wire">

func (h handler) <.Name>(reqMeta <$yarpc>.ReqMeta, body <$wire>.Value) (<$thrift>.Response, error) {
	var args <$servicePackage>.<.Name>Args
	if err := args.FromWire(body); err != nil {
		return <$thrift>.Response{}, err
	}

	<if .ReturnType>
		success, resMeta, err := h.impl.<.Name>(reqMeta, <range .Arguments>args.<.Name>,<end>)
	<else>
		resMeta, err := h.impl.<.Name>(reqMeta, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := err != nil
	result, err := <$servicePackage>.<.Name>Helper.WrapResponse(<if .ReturnType>success,<end> err)

	var response <$thrift>.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
<end>
`

const clientTemplate = `
<$pkgname := printf "%sclient" (lower .Service.Name)>
package <$pkgname>

<$yarpc     := import "github.com/yarpc/yarpc-go">
<$transport := import "github.com/yarpc/yarpc-go/transport">
<$thrift    := import "github.com/yarpc/yarpc-go/encoding/thrift">
<$protocol  := import "github.com/thriftrw/thriftrw-go/protocol">

// Interface is a client for the <.Service.Name> service.
type Interface interface {
	<if .Parent>
		<$parentPath := printf "%s/yarpc/%sclient" .Parent.Package (lower .Parent.Name)>
		<import $parentPath>.Interface
	<end>

	<range .Service.Functions>
		<.Name>(
			reqMeta <$yarpc>.CallReqMeta, <range .Arguments>
				<lowerFirst .Name> <formatType .Type>,<end>
		) <if .ReturnType> (<formatType .ReturnType>, <$yarpc>.CallResMeta, error)
		<else> (<$yarpc>.CallResMeta, error)
		<end>
	<end>
}

</* TODO(abg): Pull the default routing name from a Thrift annotation? */>

// New builds a new client for the <.Service.Name> service.
//
// 	client := <$pkgname>.New(dispatcher.Channel("<lower .Service.Name>"))
func New(c <$transport>.Channel, opts ...<$thrift>.ClientOption) Interface {
	return client{c: <$thrift>.New(<$thrift>.Config{
		Service: "<.Service.Name>",
		Channel: c,
		Protocol: <$protocol>.Binary,
	}, opts...)}
}

type client struct{ c <$thrift>.Client }

<$service := .Service>
<range .Service.Functions>

<$servicePackage := import $service.Package>
<$wire := import "github.com/thriftrw/thriftrw-go/wire">

func (c client) <.Name>(
	reqMeta <$yarpc>.CallReqMeta, <range .Arguments>
	_<.Name> <formatType .Type>,<end>
) (<if .ReturnType>success <formatType .ReturnType>,<end> resMeta <$yarpc>.CallResMeta, err error) {
	args := <$servicePackage>.<.Name>Helper.Args(<range .Arguments>_<.Name>, <end>)

	var body <$wire>.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
	if err != nil {
		return
	}

	var result <$servicePackage>.<.Name>Result
	if err = result.FromWire(body); err != nil {
		return
	}

	<if .ReturnType>success, <end>err = <$servicePackage>.<.Name>Helper.UnwrapResponse(&result)
	return
}
<end>
`

func lowerFirst(s string) string {
	head, headIndex := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(head)) + string(s[headIndex:])
}

var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
	plugin.TemplateFunc("lowerFirst", lowerFirst),
}

type generator struct{}

func (generator) Generate(req *api.GenerateServiceRequest) (*api.GenerateServiceResponse, error) {
	files := make(map[string][]byte)
	for _, serviceID := range req.RootServices {
		service := req.Services[serviceID]
		module := req.Modules[service.ModuleID]

		var parent *api.Service
		if service.ParentID != nil {
			parent = req.Services[*service.ParentID]
		}

		templateData := struct {
			Service *api.Service
			Parent  *api.Service
		}{Service: service, Parent: parent}

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

func main() {
	plugin.Main(&plugin.Plugin{Name: "yarpc", ServiceGenerator: generator{}})
}
