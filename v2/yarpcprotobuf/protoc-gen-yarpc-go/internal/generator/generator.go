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

package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"golang.org/x/tools/go/ast/astutil"
)

const _plugin = "protoc-gen-yarpc-go"

// Generate uses the given *descriptor.FileDescriptorProto to generate
// YARPC client and server stubs.
func Generate(req *plugin.CodeGeneratorRequest) (*plugin.CodeGeneratorResponse, error) {
	r, err := newRegistry(req)
	if err != nil {
		return nil, err
	}

	targets := getTargetFiles(req.GetFileToGenerate())
	var files []*plugin.CodeGeneratorResponse_File
	for _, f := range req.GetProtoFile() {
		filename := f.GetName()
		if _, ok := targets[filename]; !ok {
			continue
		}
		data, err := r.GetData(filename)
		if err != nil {
			return nil, err
		}
		raw, err := execTemplate(data)
		if err != nil {
			return nil, err
		}
		formatted, err := removeUnusedImports(filename, raw)
		if err != nil {
			return nil, err
		}
		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(fmt.Sprintf("%s.pb.yarpc.go", strings.TrimSuffix(filename, filepath.Ext(filename)))),
			Content: proto.String(string(formatted)),
		})

	}
	return &plugin.CodeGeneratorResponse{
		File: files,
	}, nil
}

func getTargetFiles(ts []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ts))
	for _, t := range ts {
		m[t] = struct{}{}
	}
	return m
}

// removeUnusedImports parses the buffer, interpreting it as Go code,
// and removes all unused imports. If successful, the result is then
// formatted.
func removeUnusedImports(filename string, buf []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, buf, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go code: %v", err)
	}

	// Note: This will do a bunch of passes over the AST. We can make it
	// faster if this ever becomes a problem.

	// Map from import path to import alias.
	for _, spec := range f.Imports {
		// spec.Path.Value is a quoted version of the import path. So we get
		// "sync", not sync. We need to unquote it.
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			// Unreachable. If the file parsed successfully, the unquote will
			// never fail.
			return nil, err
		}

		if !astutil.UsesImport(f, importPath) {
			var name string
			if spec.Name != nil {
				name = spec.Name.Name
			}
			astutil.DeleteNamedImport(fset, f, name, importPath)
		}
	}

	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, f); err != nil {
		return nil, fmt.Errorf("failed to format Go code: %v", err)
	}
	return buffer.Bytes(), nil
}
