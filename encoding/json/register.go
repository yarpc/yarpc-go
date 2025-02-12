// Copyright (c) 2025 Uber Technologies, Inc.
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

package json

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/yarpc/api/transport"
)

var (
	_ctxType            = reflect.TypeOf((*context.Context)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
	_interfaceEmptyType = reflect.TypeOf((*interface{})(nil)).Elem()
)

// Register calls the RouteTable's Register method.
//
// This function exists for backwards compatibility only. It will be removed
// in a future version.
//
// Deprecated: Use the RouteTable's Register method directly.
func Register(r transport.RouteTable, rs []transport.Procedure) {
	r.Register(rs)
}

// Procedure builds a Procedure from the given JSON handler. handler must be
// a function with a signature similar to,
//
//	f(ctx context.Context, body $reqBody) ($resBody, error)
//
// Where $reqBody and $resBody are a map[string]interface{} or pointers to
// structs.
func Procedure(name string, handler interface{}) []transport.Procedure {
	return []transport.Procedure{
		{
			Name: name,
			HandlerSpec: transport.NewUnaryHandlerSpec(
				wrapUnaryHandler(name, handler),
			),
			Encoding: Encoding,
		},
	}
}

// OnewayProcedure builds a Procedure from the given JSON handler. handler must be
// a function with a signature similar to,
//
//	f(ctx context.Context, body $reqBody) error
//
// Where $reqBody is a map[string]interface{} or pointer to a struct.
func OnewayProcedure(name string, handler interface{}) []transport.Procedure {
	return []transport.Procedure{
		{
			Name: name,
			HandlerSpec: transport.NewOnewayHandlerSpec(
				wrapOnewayHandler(name, handler)),
			Encoding: Encoding,
		},
	}
}

// wrapUnaryHandler takes a valid JSON handler function and converts it into a
// transport.UnaryHandler.
func wrapUnaryHandler(name string, handler interface{}) transport.UnaryHandler {
	reqBodyType := verifyUnarySignature(name, reflect.TypeOf(handler))
	return newJSONHandler(reqBodyType, handler)
}

// wrapOnewayHandler takes a valid JSON handler function and converts it into a
// transport.OnewayHandler.
func wrapOnewayHandler(name string, handler interface{}) transport.OnewayHandler {
	reqBodyType := verifyOnewaySignature(name, reflect.TypeOf(handler))
	return newJSONHandler(reqBodyType, handler)
}

func newJSONHandler(reqBodyType reflect.Type, handler interface{}) jsonHandler {
	var r requestReader
	if reqBodyType == _interfaceEmptyType {
		r = ifaceEmptyReader{}
	} else if reqBodyType.Kind() == reflect.Map {
		r = mapReader{reqBodyType}
	} else {
		// struct ptr
		r = structReader{reqBodyType.Elem()}
	}

	return jsonHandler{
		reader:  r,
		handler: reflect.ValueOf(handler),
	}
}

// verifyUnarySignature verifies that the given type matches what we expect from
// JSON unary handlers and returns the request type.
func verifyUnarySignature(n string, t reflect.Type) reflect.Type {
	reqBodyType := verifyInputSignature(n, t)

	if t.NumOut() != 2 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 2 results but it had %v",
			n, t.NumOut(),
		))
	}

	if t.Out(1) != _errorType {
		panic(fmt.Sprintf(
			"handler for %q must return error as its second reuslt, not %v",
			n, t.Out(1),
		))
	}

	resBodyType := t.Out(0)

	if !isValidReqResType(resBodyType) {
		panic(fmt.Sprintf(
			"the first result of the handler for %q must be "+
				"a struct pointer, a map[string]interface{}, or interface{], and not: %v",
			n, resBodyType,
		))
	}

	return reqBodyType
}

// verifyOnewaySignature verifies that the given type matches what we expect
// from oneway JSON handlers.
//
// Returns the request type.
func verifyOnewaySignature(n string, t reflect.Type) reflect.Type {
	reqBodyType := verifyInputSignature(n, t)

	if t.NumOut() != 1 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 1 result but it had %v",
			n, t.NumOut(),
		))
	}

	if t.Out(0) != _errorType {
		panic(fmt.Sprintf(
			"the result of the handler for %q must be of type error, and not: %v",
			n, t.Out(0),
		))
	}

	return reqBodyType
}

// verifyInputSignature verifies that the given input argument types match
// what we expect from JSON handlers and returns the request body type.
func verifyInputSignature(n string, t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"handler for %q is not a function but a %v", n, t.Kind(),
		))
	}

	if t.NumIn() != 2 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 2 arguments but it had %v",
			n, t.NumIn(),
		))
	}

	if t.In(0) != _ctxType {
		panic(fmt.Sprintf(
			"the first argument of the handler for %q must be of type "+
				"context.Context, and not: %v", n, t.In(0),
		))

	}

	reqBodyType := t.In(1)

	if !isValidReqResType(reqBodyType) {
		panic(fmt.Sprintf(
			"the second argument of the handler for %q must be "+
				"a struct pointer, a map[string]interface{}, or interface{}, and not: %v",
			n, reqBodyType,
		))
	}

	return reqBodyType
}

// isValidReqResType checks if the given type is a pointer to a struct, a
// map[string]interface{}, or a interface{}.
func isValidReqResType(t reflect.Type) bool {
	return (t == _interfaceEmptyType) ||
		(t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) ||
		(t.Kind() == reflect.Map && t.Key().Kind() == reflect.String)
}
