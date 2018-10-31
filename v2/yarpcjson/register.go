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

package yarpcjson

import (
	"context"
	"fmt"
	"reflect"

	yarpc "go.uber.org/yarpc/v2"
)

var (
	_ctxType            = reflect.TypeOf((*context.Context)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
	_interfaceEmptyType = reflect.TypeOf((*interface{})(nil)).Elem()
)

// Procedure builds an EncodingProcedure from the given JSON handler. handler must be
// a function with a signature similar to,
//
// 	f(ctx context.Context, body $reqBody) ($resBody, error)
//
// Where $reqBody and $resBody are of type map[string]interface{}, interface{}, or
// struct pointers.
func Procedure(name string, handler interface{}) []yarpc.EncodingProcedure {
	return []yarpc.EncodingProcedure{
		{
			Name: name,
			HandlerSpec: yarpc.NewUnaryEncodingHandlerSpec(
				wrapUnaryHandler(name, handler),
			),
			Encoding: Encoding,
			Codec:    newCodec(handler),
		},
	}
}

// wrapUnaryHandler takes a valid JSON handler function and converts it into a
// yarpc.UnaryEncodingHandler.
func wrapUnaryHandler(name string, handler interface{}) yarpc.UnaryEncodingHandler {
	verifyUnarySignature(name, reflect.TypeOf(handler))
	return jsonHandler{
		handler: reflect.ValueOf(handler),
	}
}

// verifyUnarySignature verifies that the given type matches what we expect from
// JSON unary handlers.
func verifyUnarySignature(n string, t reflect.Type) {
	verifyInputSignature(n, t)

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
}

// verifyInputSignature verifies that the given input argument types match
// what we expect from JSON handlers.
func verifyInputSignature(n string, t reflect.Type) {
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
}

// isValidReqResType checks if the given type is a pointer to a struct, a
// map[string]interface{}, or a interface{}.
func isValidReqResType(t reflect.Type) bool {
	return (t == _interfaceEmptyType) ||
		(t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) ||
		(t.Kind() == reflect.Map && t.Key().Kind() == reflect.String)
}
