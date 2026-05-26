// Copyright (c) 2026 Uber Technologies, Inc.
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
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
	"go.uber.org/yarpc/api/transport"
)

// Svc is a Thrift service.
type Svc struct {
	*api.Service

	Module *api.Module

	// Ordered list of parents of this service. If the list is non-empty, the
	// immediate parent of this service is the first item in the list, its
	// parent service is next, and so on.
	Parents []*Svc
}

// AllFunctions returns a list of all functions for this service including
// inherited functions.
func (s *Svc) AllFunctions() []*api.Function {
	var (
		functions []*api.Function
		added     = make(map[string]struct{})
		services  = append([]*Svc{s}, s.Parents...)
	)

	for _, s := range services {
		for _, f := range s.Functions {
			if _, taken := added[f.ThriftName]; taken {
				continue
			}

			functions = append(functions, f)
		}
	}

	return functions
}

// Parent returns the immediate parent of this service or nil if it doesn't
// have any.
func (s *Svc) Parent() *api.Service {
	if len(s.Parents) > 0 {
		return s.Parents[0].Service
	}
	return nil
}

// ServerPackagePath returns the import path to the server package for this
// service.
func (s *Svc) ServerPackagePath() string {
	return fmt.Sprintf("%s/%sserver", s.Module.ImportPath, strings.ToLower(s.Name))
}

// ClientPackagePath returns the import path to the server package for this
// service.
func (s *Svc) ClientPackagePath() string {
	return fmt.Sprintf("%s/%sclient", s.Module.ImportPath, strings.ToLower(s.Name))
}

// TestPackagePath returns the import path to the testpackage for this
// service.
func (s *Svc) TestPackagePath() string {
	return fmt.Sprintf("%s/%stest", s.Module.ImportPath, strings.ToLower(s.Name))
}

// FxPackagePath returns the import path to the Fx package for this service.
func (s *Svc) FxPackagePath() string {
	return fmt.Sprintf("%s/%sfx", s.Module.ImportPath, strings.ToLower(s.Name))
}

// serviceTemplateData contains the data for code gen templates that operate on
// a Thrift service.
type serviceTemplateData struct {
	*Svc

	ContextImportPath   string
	UnaryWrapperImport  string
	UnaryWrapperFunc    string
	OnewayWrapperImport string
	OnewayWrapperFunc   string
	SanitizeTChannel    bool
	MockLibrary         string

	// CompiledModule is the result of compile.Compile on the service's Thrift
	// file. It is shared across all generators so the file is compiled at most
	// once per request.
	CompiledModule *compile.Module
}

// moduleTemplateData contains the data for code gen templates. This should be
// used by templates that operate on types
//
// use serviceTemplateData for generators that rely on service definitions
type moduleTemplateData struct {
	Module *api.Module

	ContextImportPath string

	// CompiledModule is the result of compile.Compile on the module's Thrift
	// file. It is shared across all generators so the file is compiled at most
	// once per request.
	CompiledModule *compile.Module
}

// ParentServerPackagePath returns the import path for the immediate parent
// service's YARPC server package or an empty string if this service doesn't
// extend another service.
func (d *serviceTemplateData) ParentServerPackagePath() string {
	if len(d.Parents) == 0 {
		return ""
	}
	return d.Parents[0].ServerPackagePath()
}

// ParentClientPackagePath returns the import path for the immediate parent
// service's YARPC client package or an empty string if this service doesn't
// extend another service.
func (d *serviceTemplateData) ParentClientPackagePath() string {
	if len(d.Parents) == 0 {
		return ""
	}
	return d.Parents[0].ClientPackagePath()
}

// moduleGenFunc is a function that generates some part of the code needed by the
// plugin.
type moduleGenFunc func(*moduleTemplateData, map[string][]byte) error

// serviceGenFunc is a function that generates some part of the code needed by the
// plugin.
type serviceGenFunc func(*serviceTemplateData, map[string][]byte) error

// exceptionTypeReference follows PointerType wrappers to the underlying
// TypeReference for a throws clause (Thrift exceptions are pointers in Go).
func exceptionTypeReference(t *api.Type) *api.TypeReference {
	for t != nil {
		if t.ReferenceType != nil {
			return t.ReferenceType
		}
		if t.PointerType != nil {
			t = t.PointerType
			continue
		}
		break
	}
	return nil
}

func lookupCompiledFunction(root *compile.Module, serviceThriftName, fnThriftName string) *compile.FunctionSpec {
	if root == nil || serviceThriftName == "" || fnThriftName == "" {
		return nil
	}
	svc := root.Services[serviceThriftName]
	if svc == nil {
		return nil
	}
	return svc.Functions[fnThriftName]
}

// thriftExceptionMapKey returns the map key string for thrift.Method.Exceptions:
// how the exception type is referenced from the service's Thrift file — either
// the bare exception name (same file as the service) or includeAlias.TypeName
// when the exception lives in another included Thrift file (includeAlias is the
// basename of the included file without .thrift, per ThriftRW compile.Module).
func thriftExceptionMapKey(serviceModule *compile.Module, exc *compile.StructSpec) string {
	if serviceModule == nil || exc == nil {
		return ""
	}
	excPath := filepath.Clean(exc.ThriftFile())
	svcPath := filepath.Clean(serviceModule.ThriftPath)
	if excPath == svcPath {
		return exc.Name
	}
	for _, inc := range serviceModule.Includes {
		if inc == nil || inc.Module == nil {
			continue
		}
		if filepath.Clean(inc.Module.ThriftPath) == excPath {
			return inc.Name + "." + exc.Name
		}
	}
	base := strings.TrimSuffix(filepath.Base(excPath), ".thrift")
	if base != "" {
		return base + "." + exc.Name
	}
	return exc.Name
}

// methodExceptionsMapLiteral returns a Go expression for thrift.Method.Exceptions
// for the given Thrift function (exception type name -> rpc.code or the
// "__not_set__" sentinel as a string literal).
// serviceMod is compile.Compile output for the Thrift file that declares this
// service (from main.go's per-request memo); nil skips compiled metadata and
// keys fall back to the short type name from the plugin API.
// serviceThriftName is the Thrift IDL service name.
func methodExceptionsMapLiteral(f *api.Function, serviceMod *compile.Module, serviceThriftName string) string {
	if f == nil || len(f.Exceptions) == 0 {
		return "nil"
	}

	compileFn := lookupCompiledFunction(serviceMod, serviceThriftName, f.ThriftName)

	// Deduplicate by map key (last wins) so duplicate throws entries do not emit
	// duplicate composite literal keys.
	entries := make(map[string]string)
	for i, ex := range f.Exceptions {
		if ex == nil || ex.Type == nil {
			continue
		}
		ref := exceptionTypeReference(ex.Type)
		if ref == nil {
			continue
		}

		keyStr := ""
		if compileFn != nil && compileFn.ResultSpec != nil && i < len(compileFn.ResultSpec.Exceptions) {
			if ss, ok := compileFn.ResultSpec.Exceptions[i].Type.(*compile.StructSpec); ok {
				keyStr = thriftExceptionMapKey(serviceMod, ss)
			}
		}
		if keyStr == "" {
			keyStr = ref.Name
		}

		if code, ok := ref.Annotations[_errorCodeAnnotationKey]; ok && code != "" {
			entries[keyStr] = code
		} else {
			entries[keyStr] = transport.RPCCodeNotSetLiteral
		}
	}
	if len(entries) == 0 {
		return "nil"
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, strconv.Quote(k)+": "+strconv.Quote(entries[k]))
	}
	return "map[string]string{" + strings.Join(parts, ", ") + "}"
}

// Default options for the template
var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
	plugin.TemplateFunc("methodExceptionsMapLiteral", methodExceptionsMapLiteral),
}
