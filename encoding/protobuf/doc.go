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

// Package protobuf implements Protocol Buffers encoding support for YARPC.
//
// To use this package, you must have protoc installed, as well as the
// golang protoc plugin from either github.com/golang/protobuf or
// github.com/gogo/protobuf. We recommend github.com/gogo/protobuf.
//
//   go get github.com/gogo/protobuf/protoc-gen-gogoslick
//
// You must also install the Protobuf plugin for YARPC.
//
//   go get go.uber.org/yarpc/encoding/protoc-gen-yarpc-go
//
// To generate YARPC compatible code from a Protobuf file:
//
//   protoc --gogoslick_out=. --yarpc-go_out=. foo.proto
package protobuf
