// Copyright (c) 2016 Uber Technologies, Inc.
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

// Package thrift implements Thrift encoding support for YARPC.
//
// To use this package, you must install ThriftRW 0.2.0 or newer.
//
// 	go get go.uber.org/thriftrw
//
// You must also install the ThriftRW plugin for YARPC.
//
// 	go get go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc
//
// To generate YARPC compatible code from a Thrift file, use the command,
//
// 	thriftrw --plugin yarpc myservice.thrift
//
// In addition to generating code for types specified in your THrift file,
// this will generate two packages for each service in the file: a client
// package and a server package.
//
// 	myservice
// 	 |- yarpc
// 	     |- myserviceclient
// 	     |- myserviceserver
//
// The client package allows sending requests through a YARPC dispatcher.
//
// 	client := myserviceclient.New(dispatcher.ClientConfig("myservice"))
//
// The server package facilitates registration of service implementations with
// a YARPC dispatcher.
//
// 	handler := myHandler{}
// 	dispatcher.Register(myserviceserver.New(handler))
package thrift
