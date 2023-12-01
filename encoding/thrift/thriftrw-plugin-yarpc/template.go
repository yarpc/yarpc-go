// Copyright (c) 2025 Uber Technologies, Inc.
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
	"strings"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
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
}

// moduleTemplateData contains the data for code gen templates. This should be
// used by templates that operate on types
//
// use serviceTemplateData for generators that rely on service definitions
type moduleTemplateData struct {
	Module *api.Module

	ContextImportPath string
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

// Default options for the template
var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
}
