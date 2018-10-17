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
	"fmt"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// Data holds the information required for
// the protoc-gen-yarpc-go plugin.
type Data struct {
	File    *File
	Imports Imports
}

// File represents a Protobuf file descriptor.
type File struct {
	descriptor *descriptor.FileDescriptorProto

	Name         string
	Package      *Package
	Reflection   *Reflection
	Services     []*Service
	Dependencies []*File
}

// Package holds information with respect
// to a Proto type's package.
type Package struct {
	alias string
	name  string

	GoPackage string
}

// fqn returns the fully-qualified name for
// the given name based on this package.
//
//  p := &Package{name: "foo.bar"}
//  p.fqn("Baz") -> "foo.bar.Baz"
func (p *Package) fqn(name string) string {
	return fmt.Sprintf("%s.%s", p.name, name)
}

// Service represents a Protobuf service definition.
//
//  {
//    Name:       "Baz",
//    FQN:        "foo.bar.Baz",
//    Client:     "BazYARPCClient",
//    ClientImpl: "_BazYARPCClient",
//    FxClient:   "FxBazYARPCClient",
//    Server:     "BazYARPCServer",
//    ServerImpl: "_BazYARPCServer",
//    FxServer:   "FxBazYARPCServer",
//    Procedures: "BazYARPCProcedures",
//  }
type Service struct {
	Name       string
	FQN        string
	Client     string
	ClientImpl string
	FxClient   string
	Server     string
	ServerImpl string
	FxServer   string
	Procedures string
	Methods    []*Method
}

// Method represents a standard RPC method.
//
//  {
//    Name:             "FooBar",
//    StreamClient:     "FooBarYARPCStreamClient",
//    StreamClientImpl: "_FooBarYARPCStreamClient",
//    StreamServer:     "FooBarYARPCStreamServer",
//    StreamServerImpl: "_FooBarYARPCStreamServer",
//  }
type Method struct {
	Name             string
	StreamClient     string
	StreamClientImpl string
	StreamServer     string
	StreamServerImpl string
	Request          *Message
	Response         *Message
	ClientStreaming  bool
	ServerStreaming  bool
}

// Message represents a Protobuf message definition.
type Message struct {
	Name    string
	Package *Package
}

// Reflection represents the server reflection data.
type Reflection struct {
	Var      string
	Encoding string
}
