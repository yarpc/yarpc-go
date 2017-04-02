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

package config

import (
	"reflect"
	"strings"

	"github.com/golang/mock/gomock"
)

// builderFunc builds a `build` function that calls into the controller.
//
// The generated function has the signature,
//
// 	func $name($args[0], $args[1], ..., $args[N]) ($output, error)
//
// This function may be fed as an argument to BuildTransport, BuildInbound,
// etc. and it will be interpreted correctly.
func builderFunc(c *gomock.Controller, o interface{}, name string, argTypes []reflect.Type, output reflect.Type) interface{} {
	// We dynamically generate a function with the correct arg type rather
	// than interface{} because we want to verify we're getting the correct
	// type decoded.

	resultTypes := []reflect.Type{output, _typeOfError}
	return reflect.MakeFunc(
		reflect.FuncOf(argTypes, resultTypes, false),
		func(callArgs []reflect.Value) []reflect.Value {
			args := make([]interface{}, len(callArgs))
			for i, a := range callArgs {
				args[i] = a.Interface()
			}

			results := c.Call(o, name, args...)
			callResults := make([]reflect.Value, len(results))
			for i, r := range results {
				// Use zero-value where the result is nil because
				// reflect.ValueOf(nil) is an error.
				if r == nil {
					callResults[i] = reflect.Zero(resultTypes[i])
					continue
				}

				callResults[i] = reflect.ValueOf(r).Convert(resultTypes[i])
			}

			return callResults
		},
	).Interface()
}

// converts leading tabs to two spaces, such that YAML accepts the whole block.
func expand(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = expandLine(l)
	}
	return strings.Join(lines, "\n")
}

func expandLine(l string) string {
	for i, c := range l {
		if c != '\t' {
			return strings.Repeat("  ", i) + l[i:]
		}
	}
	return ""
}
