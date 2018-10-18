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
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

const (
	_implPrefix = "_"
	_client     = "Client"
	_server     = "Server"
	_stream     = "Stream"
	_fx         = "Fx"
	_yarpc      = "YARPC"
	_procedures = "Procedures"
)

// registry is used to collect and register all
// of the Protobuf types relevant to protoc-gen-yarpc-go.
type registry struct {
	files    map[string]*File
	messages map[string]*Message
	imports  Imports

	/* Plugin parameters */
	packages map[string]string
}

func newRegistry(req *plugin.CodeGeneratorRequest) (*registry, error) {
	r := &registry{
		files:    make(map[string]*File),
		messages: make(map[string]*Message),
		packages: make(map[string]string),
		imports: Imports{
			"context":                                       "context",
			"fmt":                                           "fmt",
			"go.uber.org/fx":                                "fx",
			"github.com/gogo/protobuf/proto":                "proto",
			"go.uber.org/yarpc/v2":                          "yarpc",
			"go.uber.org/yarpc/v2/yarpcprotobuf":            "yarpcprotobuf",
			"go.uber.org/yarpc/v2/yarpcprotobuf/reflection": "reflection",
		},
	}
	// Process command-line plugin parameters.
	if req.Parameter != nil {
		for _, p := range strings.Split(req.GetParameter(), ",") {
			param := strings.SplitN(p, "=", 2)
			if len(param) != 2 {
				return nil, fmt.Errorf("package modifiers should include a single '='")
			}
			flag, value := param[0], param[1]
			switch {
			case strings.HasPrefix(flag, "M"):
				r.packages[flag[1:]] = value
			default:
				return nil, fmt.Errorf("unknown flag: %v", flag)
			}
		}
	}
	return r, r.Load(req)
}

// Load registers all of the Proto types provided in the
// CodeGeneratorRequest with the registry.
func (r *registry) Load(req *plugin.CodeGeneratorRequest) error {
	for _, f := range req.GetProtoFile() {
		if err := r.loadFile(f); err != nil {
			return err
		}
	}
	for _, name := range req.FileToGenerate {
		target, ok := r.files[name]
		if !ok {
			return fmt.Errorf("file target %q was not registered", name)
		}
		for _, s := range target.descriptor.GetService() {
			if err := r.loadService(target, s); err != nil {
				return err
			}
		}
		if err := r.loadDependencies(target); err != nil {
			return err
		}
	}
	return nil
}

// GetData returns the template data the corresponds
// to the given filename.
func (r *registry) GetData(filename string) (*Data, error) {
	f, err := r.lookupFile(filename)
	if err != nil {
		return nil, err
	}
	return &Data{
		File:    f,
		Imports: r.imports,
	}, nil
}

// lookupFile returns the File that corresponds to the
// given filename.
func (r *registry) lookupFile(filename string) (*File, error) {
	f, ok := r.files[filename]
	if !ok {
		return nil, fmt.Errorf("file %q was not found", filename)
	}
	return f, nil
}

// lookupMessage returns the Message that corresponds to the
// given name. This method expects the input to be formed
// as an input or output type, such as .foo.Bar.
func (r *registry) lookupMessage(name string) (*Message, error) {
	// All input and output types are represented as
	// .$(Package).$(Message), so we explicitly trim
	// the leading '.' prefix.
	msg := strings.TrimPrefix(name, ".")
	m, ok := r.messages[msg]
	if !ok {
		return nil, fmt.Errorf("message %q was not found", msg)
	}
	return m, nil
}

// loadFile registers the given file's message types.
// Note that we load the messages for all files up-front so that
// all of the message types potentially referenced in the proto
// services can reference these types.
func (r *registry) loadFile(f *descriptor.FileDescriptorProto) error {
	reflection, err := getReflection(f)
	if err != nil {
		return err
	}
	file := &File{
		descriptor: f,
		Name:       f.GetName(),
		Package:    r.newPackage(f),
		Reflection: reflection,
	}
	r.files[file.Name] = file
	for _, m := range f.GetMessageType() {
		r.loadMessage(file, m)
	}
	return nil
}

func (r *registry) loadMessage(f *File, m *descriptor.DescriptorProto) {
	name := m.GetName()
	msg := &Message{
		Name:    name,
		Package: f.Package,
	}
	r.messages[f.Package.fqn(name)] = msg

	for _, n := range m.GetNestedType() {
		r.loadMessage(f, n)
	}
}

func (r *registry) loadService(f *File, s *descriptor.ServiceDescriptorProto) error {
	name := s.GetName()
	service := &Service{
		Name:       name,
		FQN:        f.Package.fqn(name),
		Client:     join(name, _yarpc, _client),
		ClientImpl: join(_implPrefix, name, _yarpc, _client),
		FxClient:   join(_fx, name, _yarpc, _client),
		Server:     join(name, _yarpc, _server),
		ServerImpl: join(_implPrefix, name, _yarpc, _server),
		FxServer:   join(_fx, name, _yarpc, _server),
		Procedures: join(name, _yarpc, _procedures),
	}
	for _, m := range s.GetMethod() {
		method, err := r.newMethod(m, name)
		if err != nil {
			return err
		}
		service.Methods = append(service.Methods, method)
	}
	f.Services = append(f.Services, service)
	return nil
}

// loadDependencies collects all of the File's dependencies, both
// direct and transitive, and assigns them to file.Dependencies.
func (r *registry) loadDependencies(file *File) error {
	seen := make(map[string]struct{})
	files, err := r.loadDependenciesRecurse(file, seen)
	if err != nil {
		return err
	}
	file.Dependencies = files
	return nil
}

func (r *registry) loadDependenciesRecurse(file *File, seen map[string]struct{}) ([]*File, error) {
	seen[file.descriptor.GetName()] = struct{}{}
	var deps []*File
	for _, filename := range file.descriptor.GetDependency() {
		if _, ok := seen[filename]; ok {
			continue
		}
		f, err := r.lookupFile(filename)
		if err != nil {
			return nil, err
		}
		deps = append(deps, f)

		files, err := r.loadDependenciesRecurse(f, seen)
		if err != nil {
			return nil, err
		}
		deps = append(deps, files...)
	}
	return deps, nil
}

func (r *registry) newMethod(m *descriptor.MethodDescriptorProto, service string) (*Method, error) {
	request, err := r.lookupMessage(m.GetInputType())
	if err != nil {
		return nil, err
	}
	response, err := r.lookupMessage(m.GetOutputType())
	if err != nil {
		return nil, err
	}
	name := m.GetName()
	return &Method{
		Name:             name,
		Request:          request,
		Response:         response,
		ClientStreaming:  m.GetClientStreaming(),
		ServerStreaming:  m.GetServerStreaming(),
		StreamClient:     join(service, name, _yarpc, _stream, _client),
		StreamClientImpl: join(_implPrefix, service, name, _yarpc, _stream, _client),
		StreamServer:     join(service, name, _yarpc, _stream, _server),
		StreamServerImpl: join(_implPrefix, service, name, _yarpc, _stream, _server),
	}, nil
}

func (r *registry) newPackage(f *descriptor.FileDescriptorProto) *Package {
	return &Package{
		alias:     r.imports.Add(r.importPath(f)),
		name:      f.GetPackage(),
		GoPackage: goPackage(f),
	}
}

// getReflection returns the reflection details for the given FileDescriptorProto,
// which includes the reflection variable name and the individual file's serialized
// source representation.
func getReflection(fd *descriptor.FileDescriptorProto) (*Reflection, error) {
	encoding, err := getEncodedFileDescriptor(fd)
	if err != nil {
		return nil, err
	}
	return &Reflection{
		Var:      getReflectionVar(fd),
		Encoding: encoding,
	}, nil
}

// getReflectionVar returns the generated reflection closure's variable name.
func getReflectionVar(fd *descriptor.FileDescriptorProto) string {
	// Use a sha256 of the filename instead of the filename to prevent any characters that are illegal
	// as golang identifiers and to discourage external usage of this constant.
	h := sha256.Sum256([]byte(fd.GetName()))
	return fmt.Sprintf("yarpcFileDescriptorClosure%s", hex.EncodeToString(h[:8]))
}

// getEncodedFileDescriptor returns a string representation of the FileDescriptorProto's
// serialized bytes.
func getEncodedFileDescriptor(fd *descriptor.FileDescriptorProto) (string, error) {
	fdBytes, err := getSerializedFileDescriptor(fd)
	if err != nil {
		return "", err
	}

	// Create string that contains a golang byte slice literal containing the
	// serialized file descriptor:
	//
	// []byte{
	//     0x00, 0x01, 0x02, ..., 0xFF,	// Up to 16 bytes per line
	// }
	//
	var buf bytes.Buffer
	buf.WriteString("[]byte{\n")
	for len(fdBytes) > 0 {
		n := 16
		if n > len(fdBytes) {
			n = len(fdBytes)
		}
		for _, c := range fdBytes[:n] {
			fmt.Fprintf(&buf, "0x%02x,", c)
		}
		buf.WriteString("\n")
		fdBytes = fdBytes[n:]
	}
	buf.WriteString("}")
	return buf.String(), nil
}

// getSerializedFileDescriptor returns a gzipped marshalled representation of the FileDescriptorProto.
func getSerializedFileDescriptor(fd *descriptor.FileDescriptorProto) ([]byte, error) {
	pb := proto.Clone(fd).(*descriptor.FileDescriptorProto)
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

// goPackage determines the go package for the
// the given FileDescriptorProto.
//
// We prioritize the go_package option, using the
// base element as the value.
//
// If the option is not set, we default to the
// proto file's package.
//
// If neither of these fields are set, we default
// to the file's name, excluding the file extension.
//
// In the latter cases, we replace all '.' literals
// with '_' so that they represent valid package names.
func goPackage(f *descriptor.FileDescriptorProto) string {
	if f.Options != nil && f.Options.GoPackage != nil {
		gopkg := f.Options.GetGoPackage()
		idx := strings.LastIndex(gopkg, "/")
		if idx < 0 {
			return gopkg
		}

		return gopkg[idx+1:]
	}

	pkg := f.GetPackage()
	if f.Package == nil {
		base := filepath.Base(f.GetName())
		ext := filepath.Ext(base)
		pkg = strings.TrimSuffix(base, ext)
	}
	return strings.Replace(pkg, ".", "_", -1)
}

// importPath returns the package import path that corresponds to
// the given file descriptor.
//
// We first determine whether the user has provided a package modifier for the
// file, and use its value if so.
//
// Otherwise, we use the go_package option if it is explicitly configured.
//
// If neither of these values are set, we cannot confidently determine a valid
// import path; default to the file source's directory in this case.
//
//   protoc --yarpc-go_out=Mfoo/bar=path/to/foo/bar:.
//   -> "path/to/foo/bar"
//
//   option go_package = "foo/bar:bazpb";
//   -> "foo/bar"
func (r *registry) importPath(f *descriptor.FileDescriptorProto) string {
	if path, ok := r.packages[f.GetName()]; ok {
		return path
	}

	gopkg := f.Options.GetGoPackage()
	if idx := strings.LastIndex(gopkg, "/"); idx >= 0 {
		return gopkg[:idx]
	}

	return filepath.Dir(f.GetName())
}

func join(s ...string) string {
	return strings.Join(s, "")
}
