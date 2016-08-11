package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/thriftrw/thriftrw-go/plugin"
	"github.com/thriftrw/thriftrw-go/plugin/api"
)

func lowerFirst(s string) string {
	head, headIndex := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(head)) + string(s[headIndex:])
}

func formatType(t *api.Type, extraImports map[string]struct{}) string {
	switch {
	case t.SimpleType != nil:
		switch *t.SimpleType {
		case api.SimpleTypeBool:
			return "bool"
		case api.SimpleTypeByte:
			return "byte"
		case api.SimpleTypeInt8:
			return "int8"
		case api.SimpleTypeInt16:
			return "int16"
		case api.SimpleTypeInt32:
			return "int32"
		case api.SimpleTypeInt64:
			return "int64"
		case api.SimpleTypeFloat64:
			return "float64"
		case api.SimpleTypeString:
			return "string"
		case api.SimpleTypeStructEmpty:
			return "struct{}"
		default:
			log.Fatalf("unknown simple type: %v", *t.SimpleType)
		}
	case t.SliceType != nil:
		return "[]" + formatType(t.SliceType, extraImports)
	case t.KeyValueSliceType != nil:
		k := formatType(t.KeyValueSliceType.Left, extraImports)
		v := formatType(t.KeyValueSliceType.Right, extraImports)
		return fmt.Sprintf("[]struct{Key %v; Value %v}", k, v)
	case t.MapType != nil:
		k := formatType(t.MapType.Left, extraImports)
		v := formatType(t.MapType.Right, extraImports)
		return fmt.Sprintf("map[%v]%v", k, v)
	case t.ReferenceType != nil:
		extraImports[t.ReferenceType.Package] = struct{}{}
		// TODO(abg): What if the base name doesn't match the package name?
		return filepath.Base(t.ReferenceType.Package) + "." + t.ReferenceType.Name
	case t.PointerType != nil:
		return "*" + formatType(t.PointerType, extraImports)
	default:
		log.Fatalf("unknown type: %v", t)
	}
	return ""
}

// addExtraImports replaces the string "/*EXTRA_IMPORTS*/" with the given list
// of imports in a deterministic order.
func addExtraImports(src []byte, extraImports map[string]struct{}) []byte {
	imports := make([]string, 0, len(extraImports))
	for i := range extraImports {
		imports = append(imports, strconv.Quote(i))
	}
	sort.Strings(imports)

	out := strings.Replace(string(src), "/*EXTRA_IMPORTS*/", strings.Join(imports, "\n"), 1)
	return []byte(out)
}

const serverTemplate = `
package <lower .Name>server

import (
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/thriftrw/thriftrw-go/protocol"
	/*EXTRA_IMPORTS*/
	_<basename .Package> "<.Package>"
	<if .ParentId>
		<$parent := getService .ParentId>
		"<(getModule $parent.ModuleId).Package>/yarpc/<lower $parent.Name>server"
	<end>
)

// Interface is the server-side interface for the <.Name> service.
type Interface interface {
	<if .ParentId>
		<lower (getService .ParentId).Name>server.Interface
	<end>

	<range .Functions>
		<.Name>(
			reqMeta yarpc.ReqMeta, <range .Arguments>
			<lowerFirst .Name> <formatType .Type>,<end>
		) <if .ReturnType> (<formatType .ReturnType>, yarpc.ResMeta, error)
		<else> (yarpc.ResMeta, error)
		<end>
	<end>
}

// New prepares an implementation of the <.Name> service for registration.
//
// 	handler := <.Name>Handler{}
// 	thrift.Register(dispatcher, <lower .Name>server.New(handler))
func New(impl Interface) thrift.Service {
	return service{handler{impl}}
}

type service struct{ h handler }

func (service) Name() string {
	 return "<.Name>"
 }

func (service) Protocol() protocol.Protocol {
	return protocol.Binary
}

func (s service) Handlers() map[string]thrift.Handler {
	return map[string]thrift.Handler{<range .Functions>
			"<.ThriftName>": thrift.HandlerFunc(s.h.<.Name>),
	<end>}
}

type handler struct{ impl Interface }

<$servicePackage := printf "_%s" (basename .Package)>
<range .Functions>
func (h handler) <.Name>(reqMeta yarpc.ReqMeta, body wire.Value) (thrift.Response, error) {
	var args <$servicePackage>.<.Name>Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	<if .ReturnType>
		success, resMeta, err := h.impl.<.Name>(reqMeta, <range .Arguments>args.<.Name>,<end>)
	<else>
		resMeta, err := h.impl.<.Name>(reqMeta, <range .Arguments>args.<.Name>,<end>)
	<end>

	hadError := err != nil
	result, err := <$servicePackage>.<.Name>Helper.WrapResponse(<if .ReturnType>success,<end> err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
<end>
`

func generateServer(req *api.GenerateRequest, service *api.Service) (string, []byte, error) {
	module := req.Modules[service.ModuleId]
	packageName := strings.ToLower(service.Name) + "server"
	path := filepath.Join(module.Directory, "yarpc", packageName, "server.go")

	extraImports := make(map[string]struct{})
	tmpl, err := template.New("server").Delims("<", ">").Funcs(template.FuncMap{
		"getModule": func(moduleId int32) *api.Module {
			return req.Modules[moduleId]
		},
		"getService": func(serviceId int32) *api.Service {
			return req.Services[serviceId]
		},
		"lower":      strings.ToLower,
		"lowerFirst": lowerFirst,
		"basename":   filepath.Base,
		"formatType": func(t *api.Type) string {
			return formatType(t, extraImports)
		},
	}).Parse(serverTemplate)
	if err != nil {
		return path, nil, err
	}

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, service); err != nil {
		return "", nil, err
	}

	return path, addExtraImports(buff.Bytes(), extraImports), nil
}

const clientTemplate = `
package <lower .Name>client

import (
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/thriftrw/thriftrw-go/protocol"
	/*EXTRA_IMPORTS*/
	_<basename .Package> "<.Package>"
	<if .ParentId>
		<$parent := getService .ParentId>
		"<(getModule $parent.ModuleId).Package>/yarpc/<lower $parent.Name>client"
	<end>
)

// Interface is a client for the <.Name> service.
type Interface interface {
	<if .ParentId>
		<lower (getService .ParentId).Name>client.Interface
	<end>

	<range .Functions>
		<.Name>(
			reqMeta yarpc.CallReqMeta, <range .Arguments>
				<lowerFirst .Name> <formatType .Type>,<end>
		) <if .ReturnType> (<formatType .ReturnType>, yarpc.CallResMeta, error)
		<else> (yarpc.CallResMeta, error)
		<end>
	<end>
}

</* TODO(abg): Pull the default routing name from a Thrift annotation? */>

// New builds a new client for the <.Name> service.
//
// 	client := <lower .Name>client.New(dispatcher.Channel("<lower .Name>"))
func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{
		Service: "<.Name>",
		Channel: c,
		Protocol: protocol.Binary,
	}, opts...)}
}

type client struct{ c thrift.Client }

<$servicePackage := printf "_%s" (basename .Package)>
<range .Functions>
func (c client) <.Name>(
	reqMeta yarpc.CallReqMeta, <range .Arguments>
	_<.Name> <formatType .Type>,<end>
) (<if .ReturnType>success <formatType .ReturnType>,<end> resMeta yarpc.CallResMeta, err error) {
	args := <$servicePackage>.<.Name>Helper.Args(<range .Arguments>_<.Name>, <end>)

	var body wire.Value
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

func generateClient(req *api.GenerateRequest, service *api.Service) (string, []byte, error) {
	module := req.Modules[service.ModuleId]
	packageName := strings.ToLower(service.Name) + "client"
	path := filepath.Join(module.Directory, "yarpc", packageName, "client.go")

	extraImports := make(map[string]struct{})
	tmpl, err := template.New("client").Delims("<", ">").Funcs(template.FuncMap{
		"getModule": func(moduleId int32) *api.Module {
			return req.Modules[moduleId]
		},
		"getService": func(serviceId int32) *api.Service {
			return req.Services[serviceId]
		},
		"lower":      strings.ToLower,
		"lowerFirst": lowerFirst,
		"basename":   filepath.Base,
		"formatType": func(t *api.Type) string {
			return formatType(t, extraImports)
		},
	}).Parse(clientTemplate)
	if err != nil {
		return path, nil, err
	}

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, service); err != nil {
		return "", nil, err
	}

	return path, addExtraImports(buff.Bytes(), extraImports), nil
}

type generator struct{}

func (generator) Generate(req *api.GenerateRequest) (*api.GenerateResponse, error) {
	files := make(map[string][]byte)
	for _, serviceID := range req.RootServices {
		service := req.Services[serviceID]

		// .../yarpc/myserviceserver
		path, body, err := generateServer(req, service)
		if err != nil {
			return nil, err
		}
		files[path] = body

		// .../yarpc/myserviceclient
		path, body, err = generateClient(req, service)
		if err != nil {
			return nil, err
		}
		files[path] = body
	}
	return &api.GenerateResponse{Files: files}, nil
}

func main() {
	plugin.Main(&plugin.Plugin{Name: "yarpc", Generator: generator{}})
}
