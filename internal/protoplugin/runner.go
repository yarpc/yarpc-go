// Copyright (c) 2024 Uber Technologies, Inc.
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
	"strings"
	"text/template"

	"github.com/gogo/protobuf/proto"
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

type runner struct {
	tmpl                 *template.Template
	templateInfoChecker  func(*TemplateInfo) error
	baseImports          []string
	fileToOutputFilename func(*File) (string, error)
	unknownFlagHandler   func(key string, value string) error
}

func newRunner(
	tmpl *template.Template,
	templateInfoChecker func(*TemplateInfo) error,
	baseImports []string,
	fileToOutputFilename func(*File) (string, error),
	unknownFlagHandler func(key string, value string) error,
) *runner {
	return &runner{
		tmpl:                 tmpl,
		templateInfoChecker:  templateInfoChecker,
		baseImports:          baseImports,
		fileToOutputFilename: fileToOutputFilename,
		unknownFlagHandler:   unknownFlagHandler,
	}
}

func (r *runner) Run(request *plugin_go.CodeGeneratorRequest) *plugin_go.CodeGeneratorResponse {
	registry := newRegistry()
	if request.Parameter != nil {
		for _, p := range strings.Split(request.GetParameter(), ",") {
			spec := strings.SplitN(p, "=", 2)
			if len(spec) == 1 {
				continue
			}
			name, value := spec[0], spec[1]
			switch {
			case name == "import_prefix":
				registry.SetPrefix(value)
			case strings.HasPrefix(name, "M"):
				registry.AddPackageMap(name[1:], value)
			default:
				if r.unknownFlagHandler != nil {
					if err := r.unknownFlagHandler(name, value); err != nil {
						return newResponseError(err)
					}
				}
			}
		}
	}

	generator := newGenerator(
		registry,
		r.tmpl,
		r.templateInfoChecker,
		r.baseImports,
		r.fileToOutputFilename,
	)
	if err := registry.Load(request); err != nil {
		return newResponseError(err)
	}

	var targets []*File
	for _, target := range request.FileToGenerate {
		file, err := registry.LookupFile(target)
		if err != nil {
			return newResponseError(err)
		}
		targets = append(targets, file)
	}

	out, err := generator.Generate(targets)
	if err != nil {
		return newResponseError(err)
	}
	return newResponseFiles(out)
}

func newResponseFiles(files []*plugin_go.CodeGeneratorResponse_File) *plugin_go.CodeGeneratorResponse {
	return &plugin_go.CodeGeneratorResponse{File: files}
}

func newResponseError(err error) *plugin_go.CodeGeneratorResponse {
	return &plugin_go.CodeGeneratorResponse{Error: proto.String(err.Error())}
}
