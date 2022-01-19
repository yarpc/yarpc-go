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

// Package protobuf implements Protocol Buffers encoding support for YARPC.
//
// To use this package, you must have protoc installed, as well as the
// Golang protoc plugin from either github.com/golang/protobuf or
// github.com/gogo/protobuf. We recommend github.com/gogo/protobuf.
//
//   go get github.com/gogo/protobuf/protoc-gen-gogoslick
//
// You must also install the Protobuf plugin for YARPC:
//
//   go get go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go
//
// To generate YARPC compatible code from a Protobuf file, use the command:
//
//   protoc --gogoslick_out=. --yarpc-go_out=. foo.proto
//
// The Golang protoc plugin will generate the Golang types in foo.pb.go,
// while the YARPC plugin will generate the YARPC types in foo.pb.yarpc.go.
//
// The client interface for a service named Bar will be generated with
// the name BarYARPCClient, and can be instantiated with a
// transport.ClientConfig.
//
//   barClient := foo.NewBarYARPCClient(dispatcher.ClientConfig("myservice"))
//
// The server interface will be generated with the name BarYARPCServer. This
// is the interface that should be implemented on the server-side. Procedures
// can be constructed from an implementation of BarYARPCServer using the
// BuildBarYARPCProcedures method.
//
//   dispatcher.Register(foo.BuildBarYARPCProcedures(barServer))
//
// Proto3 defines a mapping to JSON, so for every RPC method, two Procedures
// are created for every RPC method: one that will handle the standard Protobuf
// binary encoding, and one that will handle the JSON encoding.
//
// If coupled with an HTTP Inbound, Protobuf procedures can be called using
// curl. Given the following Protobuf definition:
//
//   syntax = "proto3;
//
//   package foo.bar;
//
//   message EchoRequest {
//     string value = 1;
//   }
//
//   message EchoResponse {
//     string value = 1;
//   }
//
//   service Baz {
//     rpc Echo(EchoRequest) returns (EchoResponse) {}
//   }
//
// And the following configuration:
//
//   service:
//     name: hello
//   yarpc:
//     inbounds:
//       http:
//          address: ":8080"
//
// If running locally, one could make the following call:
//
//   curl \
//     http://0.0.0.0:8080 \
//     -H 'context-ttl-ms: 2000' \
//     -H "rpc-caller: curl-$(whoami)" \
//     -H 'rpc-service: hello' \
//     -H 'rpc-encoding: json' \
//     -H 'rpc-procedure: foo.bar.Baz::Echo' \
//     -d '{"value":"sample"}'
//
// Where context-ttl-ms is the timeout in milliseconds, rpc-caller is the name of the entity making the request,
// rpc-service is the name of the configured service, rpc-encoding is json, rpc-procedure is the name of the
// Protobuf method being called in the form proto_package.proto_service::proto_method, and the data is the JSON
// representation of the request.
//
// If using Yab, one can also use:
//
//   yab -p http://0.0.0.0:8080 -e json -s hello -p foo.bar.Baz::Echo -r '{"value":"sample"}'
//
// See https://github.com/yarpc/yab for more details.
//
// Except for any ClientOptions (such as UseJSON), the types and functions
// defined in this package should not be directly used in applications,
// instead use the code generated from protoc-gen-yarpc-go.
package protobuf
