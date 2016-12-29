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

package protoplugin

import (
	"flag"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

var (
	importPrefix = flag.String("import_prefix", "", "prefix to be added to go package paths for imported proto files")
)

func run(
	template *template.Template,
	baseImports []string,
	fileSuffix string,
) error {
	flag.Parse()
	request, err := parseRequest(os.Stdin)
	if err != nil {
		return err
	}

	registry := newRegistry()
	if request.Parameter != nil {
		for _, p := range strings.Split(request.GetParameter(), ",") {
			spec := strings.SplitN(p, "=", 2)
			if len(spec) == 1 {
				if err := flag.CommandLine.Set(spec[0], ""); err != nil {
					return err
				}
				continue
			}
			name, value := spec[0], spec[1]
			if strings.HasPrefix(name, "M") {
				registry.AddPackageMap(name[1:], value)
				continue
			}
			if err := flag.CommandLine.Set(name, value); err != nil {
				return err
			}
		}
	}

	generator := newGenerator(
		registry,
		newTemplater(template),
		baseImports,
		fileSuffix,
	)
	registry.SetPrefix(*importPrefix)
	if err := registry.Load(request); err != nil {
		return emitError(err)
	}

	var targets []*File
	for _, target := range request.FileToGenerate {
		file, err := registry.LookupFile(target)
		if err != nil {
			return err
		}
		targets = append(targets, file)
	}

	out, err := generator.Generate(targets)
	if err != nil {
		return emitError(err)
	}
	return emitFiles(out)
}

func parseRequest(reader io.Reader) (*plugin_go.CodeGeneratorRequest, error) {
	input, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	request := &plugin_go.CodeGeneratorRequest{}
	if err := proto.Unmarshal(input, request); err != nil {
		return nil, err
	}
	return request, nil
}

func emitFiles(files []*plugin_go.CodeGeneratorResponse_File) error {
	return emitResponse(&plugin_go.CodeGeneratorResponse{File: files})
}

func emitError(err error) error {
	return emitResponse(&plugin_go.CodeGeneratorResponse{Error: proto.String(err.Error())})
}

func emitResponse(response *plugin_go.CodeGeneratorResponse) error {
	buf, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		return err
	}
	return nil
}
