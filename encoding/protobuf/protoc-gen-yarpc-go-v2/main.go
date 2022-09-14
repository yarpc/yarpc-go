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
Package main provides a protoc plugin that generates code for the protobuf encoding for YARPC.
To use:
	go get github.com/golang/protobuf/protoc-gen-go
	go get go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go-v2
	protoc --go_out=plugins=grpc:. foo.proto
	protoc --yarpc-go-v2_out=. foo.proto
*/
package main

import (
	"log"
	"os"

	"go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go-v2/internal/lib"
	"go.uber.org/yarpc/internal/protoplugin"
)

func main() {
	if err := protoplugin.Do(lib.Runner, os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}