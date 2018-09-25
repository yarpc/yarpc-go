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

// Package yarpcprocedure contains utilities for handling procedure name mappings.
//
// gRPC and Thrift, in particular, decompose all procedure names into a "service
// interface" component and a "method name" component. YARPC conventionally delimits
// these components with "::", such that an example procedure is represented as
// "FooService::FooMethod".
package yarpcprocedure

import (
	"fmt"
	"strings"
)

// ToName gets the procedure name we should use for a method
// with the given service name and method name.
func ToName(service, method string) string {
	return fmt.Sprintf("%s::%s", service, method)
}

// FromName gets the service name and method name from a procdure name.
// Not all encodings separate their procedure names into service and method
// components. In these cases, such as JSON, the entire procedure name is
// returned in the first result.
func FromName(name string) (service, method string) {
	parts := strings.SplitN(name, "::", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
