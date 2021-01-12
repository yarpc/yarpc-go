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

/*
Package main provides a protoc plugin that generates code for the protobuf encoding for YARPC.

To use:
	go get github.com/gogo/protobuf/protoc-gen-gogoslick
	go get go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go
	protoc --gogoslick_out=. foo.proto
	protoc --yarpc-go_out=. foo.proto
*/
package main

// Note there is some crazy bug with protobuf that if you declare:
//
//   func bar() (kv.GetValueResponse, error) {
//     return nil, errors.New("nil response and non-nil error")
//   }
//
//   func foo() (proto.Message, error) {
//     return bar()
//   }
//
//   response, err := foo()
//   fmt.Printf("%v %v\n", response, response == nil)
//
// This will print "<nil>, false". If you try to do something with response (ie call a function on it),
// if will panic because response is nil. Something similar happens in golang/protobuf
// too, so this is insane. If in bar(), you do response == nil, it will be true.
// The generated code handles this.

import (
	"log"
	"os"

	"go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go/internal/lib"
	"go.uber.org/yarpc/internal/protoplugin"
)

func main() {
	if err := protoplugin.Do(lib.Runner, os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
