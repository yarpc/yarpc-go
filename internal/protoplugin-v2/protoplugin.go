// Copyright (c) 2022 Uber Technologies, Inc.
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
Package protopluginv2 provides utilities for protoc plugins.
*/
package protopluginv2

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"
)

// Do is a helper function for protobuf plugins.
//
//   func main() {
//     if err := protoplugin.Do(runner, os.Stdin, os.Stdout); err != nil {
//       log.Fatal(err)
//     }
//   }
func Do(runner Runner, reader io.Reader, writer io.Writer) error {
	request, err := ReadRequest(reader)
	if err != nil {
		return err
	}
	return WriteResponse(writer, runner.Run(request))
}

// ReadRequest reads the request from the reader.
func ReadRequest(reader io.Reader) (*plugin_go.CodeGeneratorRequest, error) {
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

// WriteResponse writes the response to the writer.
func WriteResponse(writer io.Writer, response *plugin_go.CodeGeneratorResponse) error {
	buf, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	if _, err := writer.Write(buf); err != nil {
		return err
	}
	return nil
}

// Runner runs the plugin logic.
type Runner interface {
	Run(*plugin_go.CodeGeneratorRequest) *plugin_go.CodeGeneratorResponse
}

// NewRunner returns a new Runner.
func NewRunner(
	tmpl *template.Template,
	templateInfoChecker func(*TemplateInfo) error,
	baseImports []string,
	fileToOutputFilename func(*File) (string, error),
	unknownFlagHandler func(key string, value string) error,
) Runner {
	return newRunner(tmpl, templateInfoChecker, baseImports, fileToOutputFilename, unknownFlagHandler)
}

// NewMultiRunner returns a new Runner that executes all the given Runners and
// merges the resulting CodeGeneratorResponses.
func NewMultiRunner(runners ...Runner) Runner {
	return newMultiRunner(runners...)
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
	GoPackage              *GoPackage
	Messages               []*Message
	Enums                  []*Enum
	Services               []*Service
	TransitiveDependencies []*File
}

// SerializedFileDescriptor returns a gzipped marshalled representation of the FileDescriptor.
func (f *File) SerializedFileDescriptor() ([]byte, error) {
	pb := proto.Clone(f.FileDescriptorProto).(*descriptor.FileDescriptorProto)
	pb.SourceCodeInfo = nil

	b, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = w.Write(b)
	if err != nil {
		return nil, err
	}
	w.Close()
	return buf.Bytes(), nil
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

// GoType returns a go type name for the message type.
// It prefixes the type name with the package alias if
// its belonging package is not "currentPackage".
func (m *Message) GoType(currentPackage string) string {
	var components []string
	components = append(components, m.Outers...)
	components = append(components, GoCamelCase(m.GetName()))

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

// FQSN returns a fully qualified service name of this service.
func (s *Service) FQSN() string {
	components := []string{""}
	if s.File.Package != nil {
		components = append(components, s.File.GetPackage())
	}
	components = append(components, s.GetName())
	return strings.Join(components, ".")
}

// Method wraps descriptor.MethodDescriptorProto for richer features.
type Method struct {
	*descriptor.MethodDescriptorProto
	Service      *Service
	RequestType  *Message
	ResponseType *Message
}

// Field wraps descriptor.FieldDescriptorProto for richer features.
type Field struct {
	*descriptor.FieldDescriptorProto
	// Message is the message type which this field belongs to.
	Message *Message
	// FieldMessage is the message type of the field.
	FieldMessage *Message
}

// GoCamelCase camel-cases a protobuf name for use as a Go identifier.
//
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
func GoCamelCase(s string) string {
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '.' in ".{{lowercase}}".
		case c == '.':
			b = append(b, '_') // convert '.' to '_'
		case c == '_' && (i == 0 || s[i-1] == '.'):
			// Convert initial '_' to ensure we start with a capital letter.
			// Do the same for '_' after '.' to match historic behavior.
			b = append(b, 'X') // convert '_' to 'X'
		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '_' in "_{{lowercase}}".
		case isASCIIDigit(c):
			b = append(b, c)
		default:
			// Assume we have a letter now - if not, it's a bogus identifier.
			// The next word is a sequence of characters that must start upper case.
			if isASCIILower(c) {
				c -= 'a' - 'A' // convert lowercase to uppercase
			}
			b = append(b, c)

			// Accept lower case sequence that follows.
			for ; i+1 < len(s) && isASCIILower(s[i+1]); i++ {
				b = append(b, s[i+1])
			}
		}
	}
	return string(b)
}

func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
