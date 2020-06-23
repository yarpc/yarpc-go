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

package protoplugin

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"path"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	errNoTargetService = errors.New("no target service defined in the file")
)

type generator struct {
	registry             *registry
	tmpl                 *template.Template
	templateInfoChecker  func(*TemplateInfo) error
	baseImports          []*GoPackage
	fileToOutputFilename func(*File) (string, error)
}

func newGenerator(
	registry *registry,
	tmpl *template.Template,
	templateInfoChecker func(*TemplateInfo) error,
	baseImportStrings []string,
	fileToOutputFilename func(*File) (string, error),
) *generator {
	var baseImports []*GoPackage
	for _, pkgpath := range baseImportStrings {
		pkg := &GoPackage{
			Path: pkgpath,
			Name: path.Base(pkgpath),
		}
		if err := registry.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
			for i := 0; ; i++ {
				alias := fmt.Sprintf("%s_%d", pkg.Name, i)
				if err := registry.ReserveGoPackageAlias(alias, pkg.Path); err != nil {
					continue
				}
				pkg.Alias = alias
				break
			}
		}
		baseImports = append(baseImports, pkg)
	}
	return &generator{
		registry,
		tmpl,
		templateInfoChecker,
		baseImports,
		fileToOutputFilename,
	}
}

func (g *generator) Generate(targets []*File) ([]*plugin_go.CodeGeneratorResponse_File, error) {
	var files []*plugin_go.CodeGeneratorResponse_File
	for _, file := range targets {
		code, err := g.generate(file)
		if err == errNoTargetService {
			continue
		}
		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(code))
		if err != nil {
			return nil, fmt.Errorf("could not format go code: %v\n%s", err, code)
		}
		output, err := g.fileToOutputFilename(file)
		if err != nil {
			return nil, err
		}
		files = append(files, &plugin_go.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})
	}
	return files, nil
}

func (g *generator) generate(file *File) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []*GoPackage
	for _, pkg := range g.baseImports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	for _, svc := range file.Services {
		for _, m := range svc.Methods {
			for _, pkg := range []*GoPackage{m.RequestType.File.GoPackage, m.ResponseType.File.GoPackage} {
				if pkg.Path == file.GoPackage.Path {
					continue
				}
				if pkgSeen[pkg.Path] {
					continue
				}
				pkgSeen[pkg.Path] = true
				imports = append(imports, pkg)
			}
		}
	}
	templateInfo := &TemplateInfo{file, imports}
	if err := g.templateInfoChecker(templateInfo); err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(nil)
	if err := g.tmpl.Execute(buffer, templateInfo); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
