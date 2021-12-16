// Copyright (c) 2021 Uber Technologies, Inc.
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
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

type registry struct {
	// msgs is a mapping from fully-qualified message name to descriptor
	msgs map[string]*Message
	// enums is a mapping from fully-qualified enum name to descriptor
	enums map[string]*Enum
	// files is a mapping from file path to descriptor
	files map[string]*File
	// prefix is a prefix to be inserted to golang package paths generated from proto package names.
	prefix string
	// pkgMap is a user-specified mapping from file path to proto package.
	pkgMap map[string]string
	// pkgAliases is a mapping from package aliases to package paths in go which are already taken.
	pkgAliases map[string]string
}

func newRegistry() *registry {
	return &registry{
		msgs:       make(map[string]*Message),
		enums:      make(map[string]*Enum),
		files:      make(map[string]*File),
		pkgMap:     make(map[string]string),
		pkgAliases: make(map[string]string),
	}
}

func (r *registry) Load(req *plugin_go.CodeGeneratorRequest) error {
	for _, file := range req.GetProtoFile() {
		r.loadFile(file)
	}
	var targetPkg string
	for _, name := range req.FileToGenerate {
		target := r.files[name]
		if target == nil {
			return fmt.Errorf("no such file: %s", name)
		}
		name := packageIdentityName(target.FileDescriptorProto)
		if targetPkg == "" {
			targetPkg = name
		} else {
			if targetPkg != name {
				return fmt.Errorf("inconsistent package names: %s %s", targetPkg, name)
			}
		}
		if err := r.loadServices(target); err != nil {
			return err
		}
		if err := r.loadTransitiveFileDependencies(target); err != nil {
			return err
		}
	}
	return nil
}

func (r *registry) LookupMessage(location string, name string) (*Message, error) {
	if strings.HasPrefix(name, ".") {
		m, ok := r.msgs[name]
		if !ok {
			return nil, fmt.Errorf("no message found: %s", name)
		}
		return m, nil
	}

	if !strings.HasPrefix(location, ".") {
		location = fmt.Sprintf(".%s", location)
	}
	components := strings.Split(location, ".")
	for len(components) > 0 {
		fqmn := strings.Join(append(components, name), ".")
		if m, ok := r.msgs[fqmn]; ok {
			return m, nil
		}
		components = components[:len(components)-1]
	}
	return nil, fmt.Errorf("no message found: %s", name)
}

func (r *registry) LookupFile(name string) (*File, error) {
	f, ok := r.files[name]
	if !ok {
		return nil, fmt.Errorf("no such file given: %s", name)
	}
	return f, nil
}

func (r *registry) AddPackageMap(file, protoPackage string) {
	r.pkgMap[file] = protoPackage
}

func (r *registry) SetPrefix(prefix string) {
	r.prefix = prefix
}

func (r *registry) ReserveGoPackageAlias(alias, pkgpath string) error {
	if taken, ok := r.pkgAliases[alias]; ok {
		if taken == pkgpath {
			return nil
		}
		return fmt.Errorf("package name %s is already taken. Use another alias", alias)
	}
	r.pkgAliases[alias] = pkgpath
	return nil
}

// loadFile loads messages, enumerations and fields from "file".
// It does not loads services and methods in "file".  You need to call
// loadServices after loadFiles is called for all files to load services and methods.
func (r *registry) loadFile(file *descriptor.FileDescriptorProto) {
	pkg := &GoPackage{
		Path: r.goPackagePath(file),
		Name: defaultGoPackageName(file),
	}
	if err := r.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
		for i := 0; ; i++ {
			alias := fmt.Sprintf("%s_%d", pkg.Name, i)
			if err := r.ReserveGoPackageAlias(alias, pkg.Path); err == nil {
				pkg.Alias = alias
				break
			}
		}
	}
	f := &File{
		FileDescriptorProto: file,
		GoPackage:           pkg,
	}
	r.files[file.GetName()] = f
	r.registerMsg(f, nil, file.GetMessageType())
	r.registerEnum(f, nil, file.GetEnumType())
}

func (r *registry) registerMsg(file *File, outerPath []string, msgs []*descriptor.DescriptorProto) {
	for i, md := range msgs {
		m := &Message{
			DescriptorProto: md,
			File:            file,
			Outers:          outerPath,
			Index:           i,
		}
		for _, fd := range md.GetField() {
			m.Fields = append(m.Fields, &Field{
				FieldDescriptorProto: fd,
				Message:              m,
			})
		}
		file.Messages = append(file.Messages, m)
		r.msgs[m.FQMN()] = m

		var outers []string
		outers = append(outers, outerPath...)
		outers = append(outers, m.GetName())
		r.registerMsg(file, outers, m.GetNestedType())
		r.registerEnum(file, outers, m.GetEnumType())
	}
}

func (r *registry) registerEnum(file *File, outerPath []string, enums []*descriptor.EnumDescriptorProto) {
	for i, ed := range enums {
		e := &Enum{
			EnumDescriptorProto: ed,
			File:                file,
			Outers:              outerPath,
			Index:               i,
		}
		file.Enums = append(file.Enums, e)
		r.enums[e.FQEN()] = e
	}
}

// goPackagePath returns the go package path which go files generated from "f" should have.
// It respects the mapping registered by AddPkgMap if exists. Or use go_package as import path
// if it includes a slash,  Otherwide, it generates a path from the file name of "f".
func (r *registry) goPackagePath(f *descriptor.FileDescriptorProto) string {
	name := f.GetName()
	if pkg, ok := r.pkgMap[name]; ok {
		return path.Join(r.prefix, pkg)
	}
	gopkg := f.Options.GetGoPackage()
	idx := strings.LastIndex(gopkg, "/")
	if idx >= 0 {
		return gopkg
	}
	return path.Join(r.prefix, path.Dir(name))
}

// loadServices registers services and their methods from "targetFile" to "r".
// It must be called after loadFile is called for all files so that loadServices
// can resolve names of message types and their fields.
func (r *registry) loadServices(file *File) error {
	var svcs []*Service
	for _, sd := range file.GetService() {
		svc := &Service{
			ServiceDescriptorProto: sd,
			File:                   file,
		}
		for _, md := range sd.GetMethod() {
			meth, err := r.newMethod(svc, md)
			if err != nil {
				return err
			}
			svc.Methods = append(svc.Methods, meth)
		}
		if len(svc.Methods) == 0 {
			continue
		}
		svcs = append(svcs, svc)
	}
	file.Services = svcs
	return nil
}

func (r *registry) newMethod(svc *Service, md *descriptor.MethodDescriptorProto) (*Method, error) {
	requestType, err := r.LookupMessage(svc.File.GetPackage(), md.GetInputType())
	if err != nil {
		return nil, err
	}
	responseType, err := r.LookupMessage(svc.File.GetPackage(), md.GetOutputType())
	if err != nil {
		return nil, err
	}
	return &Method{
		MethodDescriptorProto: md,
		Service:               svc,
		RequestType:           requestType,
		ResponseType:          responseType,
	}, nil
}

// loadTransitiveFileDependencies registers services and their methods from "targetFile" to "r".
// It must be called after loadFile is called for all files so that loadTransitiveFileDependencies
// can resolve file descriptors as depdendencies.
func (r *registry) loadTransitiveFileDependencies(file *File) error {
	seen := make(map[string]struct{})
	files, err := r.loadTransitiveFileDependenciesRecurse(file, seen)
	if err != nil {
		return err
	}
	file.TransitiveDependencies = files
	return nil
}

func (r *registry) loadTransitiveFileDependenciesRecurse(file *File, seen map[string]struct{}) ([]*File, error) {
	seen[file.GetName()] = struct{}{}
	var deps []*File
	for _, fname := range file.GetDependency() {
		if _, ok := seen[fname]; ok {
			continue
		}
		f, err := r.LookupFile(fname)
		if err != nil {
			return nil, err
		}
		deps = append(deps, f)

		files, err := r.loadTransitiveFileDependenciesRecurse(f, seen)
		if err != nil {
			return nil, err
		}
		deps = append(deps, files...)
	}
	return deps, nil
}

// defaultGoPackageName returns the default go package name to be used for go files generated from "f".
// You might need to use an unique alias for the package when you import it.  Use ReserveGoPackageAlias to get a unique alias.
func defaultGoPackageName(f *descriptor.FileDescriptorProto) string {
	name := packageIdentityName(f)
	return strings.Replace(name, ".", "_", -1)
}

// packageIdentityName returns the identity of packages.
// protoc-gen-grpc-gateway rejects CodeGenerationRequests which contains more than one packages
// as protoc-gen-go does.
func packageIdentityName(f *descriptor.FileDescriptorProto) string {
	if f.Options != nil && f.Options.GoPackage != nil {
		gopkg := f.Options.GetGoPackage()
		// if go_package specifies an alias in the form of full/path/package;alias, use alias over package
		idx := strings.Index(gopkg, ";")
		if idx >= 0 {
			return gopkg[idx+1:]
		}
		idx = strings.LastIndex(gopkg, "/")
		if idx < 0 {
			return gopkg
		}

		return gopkg[idx+1:]
	}

	if f.Package == nil {
		base := filepath.Base(f.GetName())
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return f.GetPackage()
}
