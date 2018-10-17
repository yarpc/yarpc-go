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
	"text/template"

	"go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/templatedata"
)

var _tmpl = template.Must(
	parseTemplates(
		templatedata.MustAsset("internal/template/base.tmpl"),
		templatedata.MustAsset("internal/template/client.tmpl"),
		templatedata.MustAsset("internal/template/client_impl.tmpl"),
		templatedata.MustAsset("internal/template/client_stream.tmpl"),
		templatedata.MustAsset("internal/template/fx.tmpl"),
		templatedata.MustAsset("internal/template/server.tmpl"),
		templatedata.MustAsset("internal/template/server_impl.tmpl"),
		templatedata.MustAsset("internal/template/server_stream.tmpl"),
	),
)

func parseTemplates(templates ...[]byte) (*template.Template, error) {
	t := template.New(_plugin).Funcs(
		template.FuncMap{
			"goType":        goType,
			"unaryMethods":  unaryMethods,
			"streamMethods": streamMethods,
		},
	)
	for _, tmpl := range templates {
		_, err := t.Parse(string(tmpl))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func execTemplate(data *Data) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := _tmpl.Execute(buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// goType returns a go type name for the message type.
// It prefixes the type name with the package's alias
// if it does not belong to the same package.
func goType(m *Message, pkg string) string {
	if m.Package.GoPackage != pkg && m.Package.alias != "" {
		return fmt.Sprintf("%s.%s", m.Package.alias, m.Name)
	}
	return m.Name
}

func unaryMethods(s *Service) []*Method {
	methods := make([]*Method, 0, len(s.Methods))
	for _, m := range s.Methods {
		if !m.ClientStreaming && !m.ServerStreaming {
			methods = append(methods, m)
		}
	}
	return methods
}

func streamMethods(s *Service) []*Method {
	methods := make([]*Method, 0, len(s.Methods))
	for _, m := range s.Methods {
		if m.ClientStreaming || m.ServerStreaming {
			methods = append(methods, m)
		}
	}
	return methods
}
