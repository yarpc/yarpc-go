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
Package protoplugin provides utilities for protoc plugins.

The only functions that should be called as of now is Main. The rest are
not guaranteed to stay here.

This was HEAVILY adapted from github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway.

Eventually, a rewrite of this to be simplier for what we need would be nice, but this was
available to get us here, especially with handling go imports.
*/
package protoplugin

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	protogenerator "github.com/golang/protobuf/protoc-gen-go/generator"
)

// Run is the main function for a protobuf plugin to call.
func Run(
	tmpl *template.Template,
	templateInfoChecker func(*TemplateInfo) error,
	baseImports []string,
	fileSuffix string,
) error {
	return run(tmpl, templateInfoChecker, baseImports, fileSuffix)
}

// TemplateInfo is the info passed to a template.
type TemplateInfo struct {
	*File
	Imports []*GoPackage
}

// GoPackage represents a golang package.
type GoPackage struct {
	Path string
	Name string
	// Alias is an alias of the package unique within the current invocation of the generator.
	Alias string
}

// Standard returns whether the import is a golang standard package.
func (g *GoPackage) Standard() bool {
	return !strings.Contains(g.Path, ".")
}

// String returns a string representation of this package in the form of import line in golang.
func (g *GoPackage) String() string {
	if g.Alias == "" {
		return fmt.Sprintf("%q", g.Path)
	}
	return fmt.Sprintf("%s %q", g.Alias, g.Path)
}

// File wraps descriptor.FileDescriptorProto for richer features.
type File struct {
	*descriptor.FileDescriptorProto
	GoPackage *GoPackage
	Messages  []*Message
	Enums     []*Enum
	Services  []*Service
}

// IsProto2 determines if the syntax of the file is proto2.
func (f *File) IsProto2() bool {
	return f.Syntax == nil || f.GetSyntax() == "proto2"
}

// Message describes a protocol buffer message types.
type Message struct {
	*descriptor.DescriptorProto
	File *File
	// Outers is a list of outer messages if this message is a nested type.
	Outers []string
	Fields []*Field
	// Index is proto path index of this message in File.
	Index int
}

// FQMN returns a fully qualified message name of this message.
func (m *Message) FQMN() string {
	components := []string{""}
	if m.File.Package != nil {
		components = append(components, m.File.GetPackage())
	}
	components = append(components, m.Outers...)
	components = append(components, m.GetName())
	return strings.Join(components, ".")
}

// DefaultGoType calls GoType with m.File.GoPackage.Path.
func (m *Message) DefaultGoType() string {
	return m.GoType(m.File.GoPackage.Path)
}

// GoType returns a go type name for the message type.
// It prefixes the type name with the package alias if
// its belonging package is not "currentPackage".
func (m *Message) GoType(currentPackage string) string {
	var components []string
	components = append(components, m.Outers...)
	components = append(components, m.GetName())

	name := strings.Join(components, "_")
	if m.File.GoPackage.Path == currentPackage {
		return name
	}
	pkg := m.File.GoPackage.Name
	if alias := m.File.GoPackage.Alias; alias != "" {
		pkg = alias
	}
	return fmt.Sprintf("%s.%s", pkg, name)
}

// Enum describes a protocol buffer enum type.
type Enum struct {
	*descriptor.EnumDescriptorProto
	File *File
	// Outers is a list of outer messages if this enum is a nested type.
	Outers []string
	Index  int
}

// FQEN returns a fully qualified enum name of this enum.
func (e *Enum) FQEN() string {
	components := []string{""}
	if e.File.Package != nil {
		components = append(components, e.File.GetPackage())
	}
	components = append(components, e.Outers...)
	components = append(components, e.GetName())
	return strings.Join(components, ".")
}

// Service wraps descriptor.ServiceDescriptorProto for richer features.
type Service struct {
	*descriptor.ServiceDescriptorProto
	File    *File
	Methods []*Method
}

// UnaryMethods returns the Methods that are not streaming.
func (s *Service) UnaryMethods() []*Method {
	methods := make([]*Method, 0, len(s.Methods))
	for _, method := range s.Methods {
		if !method.IsStreaming() {
			methods = append(methods, method)
		}
	}
	return methods
}

// StreamingMethods returns the Methods that are streaming.
func (s *Service) StreamingMethods() []*Method {
	methods := make([]*Method, 0, len(s.Methods))
	for _, method := range s.Methods {
		if method.IsStreaming() {
			methods = append(methods, method)
		}
	}
	return methods
}

// Method wraps descriptor.MethodDescriptorProto for richer features.
type Method struct {
	*descriptor.MethodDescriptorProto
	Service      *Service
	RequestType  *Message
	ResponseType *Message
}

// IsStreaming returns true if this Method is client or server streaming.
func (m *Method) IsStreaming() bool {
	return m.GetClientStreaming() || m.GetServerStreaming()
}

// Field wraps descriptor.FieldDescriptorProto for richer features.
type Field struct {
	*descriptor.FieldDescriptorProto
	// Message is the message type which this field belongs to.
	Message *Message
	// FieldMessage is the message type of the field.
	FieldMessage *Message
}

// FieldPath is a path to a field from a request message.
type FieldPath []*FieldPathComponent

// String returns a string representation of the field path.
func (p FieldPath) String() string {
	var components []string
	for _, c := range p {
		components = append(components, c.Name)
	}
	return strings.Join(components, ".")
}

// IsNestedProto3 indicates whether the FieldPath is a nested Proto3 path.
func (p FieldPath) IsNestedProto3() bool {
	return len(p) > 1 && !p[0].Target.Message.File.IsProto2()
}

// RHS is a right-hand-side expression in go to be used to assign a value to the target field.
// It starts with "msgExpr", which is the go expression of the method request object.
func (p FieldPath) RHS(msgExpr string) string {
	l := len(p)
	if l == 0 {
		return msgExpr
	}
	components := []string{msgExpr}
	for i, c := range p {
		if i == l-1 {
			components = append(components, c.RHS())
			continue
		}
		components = append(components, c.LHS())
	}
	return strings.Join(components, ".")
}

// FieldPathComponent is a path component in FieldPath
type FieldPathComponent struct {
	// Name is a name of the proto field which this component corresponds to.
	Name string
	// Target is the proto field which this component corresponds to.
	Target *Field
}

// RHS returns a right-hand-side expression in go for this field.
func (c *FieldPathComponent) RHS() string {
	return protogenerator.CamelCase(c.Name)
}

// LHS returns a left-hand-side expression in go for this field.
func (c *FieldPathComponent) LHS() string {
	if c.Target.Message.File.IsProto2() {
		return fmt.Sprintf("Get%s()", protogenerator.CamelCase(c.Name))
	}
	return protogenerator.CamelCase(c.Name)
}
