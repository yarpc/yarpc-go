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

package json

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"
)

var (
	_ctxType            = reflect.TypeOf((*context.Context)(nil)).Elem()
	_reqMetaType        = reflect.TypeOf((*yarpc.ReqMeta)(nil)).Elem()
	_resMetaType        = reflect.TypeOf((*yarpc.ResMeta)(nil)).Elem()
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
	_interfaceEmptyType = reflect.TypeOf((*interface{})(nil)).Elem()
)

// Register calls the Registrar's Register method.
//
// This function exists for backwards compatibility only. It will be removed
// in a future version.
//
// Deprecated: Use the Registrar's Register method directly.
func Register(r transport.Registrar, rs []transport.Registrant) {
	r.Register(rs)
}

// Procedure builds a Registrant from the given JSON handler. handler must be
// a function with a signature similar to,
//
// 	f(ctx context.Context, reqMeta yarpc.ReqMeta, body $reqBody) ($resBody, yarpc.ResMeta, error)
//
// Where $reqBody and $resBody are a map[string]interface{} or pointers to
// structs.
func Procedure(name string, handler interface{}) []transport.Registrant {
	return []transport.Registrant{
		{
			Procedure: name,
			Handler:   wrapHandler(name, handler),
		},
	}
}

// wrapHandler takes a valid JSON handler function and converts it into a
// transport.Handler.
func wrapHandler(name string, handler interface{}) transport.Handler {
	reqBodyType := verifySignature(name, reflect.TypeOf(handler))

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

// verifySignature verifies that the given type matches what we expect from
// JSON handlers and returns the request and response types.
//
// Returns the request type.
func verifySignature(n string, t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"handler for %q is not a function but a %v", n, t.Kind(),
		))
	}

	if t.NumIn() != 3 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 3 arguments but it had %v",
			n, t.NumIn(),
		))
	}

	if t.NumOut() != 3 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 3 results but it had %v",
			n, t.NumOut(),
		))
	}

	if t.In(0) != _ctxType {
		panic(fmt.Sprintf(
			"the first argument of the handler for %q must be of type "+
				"context.Context, and not: %v", n, t.In(0),
		))

	}

	if t.In(1) != _reqMetaType {
		panic(fmt.Sprintf(
			"the second argument of the handler for %q must be of type "+
				"yarpc.ReqMeta, and not: %v", n, t.In(0),
		))
	}

	if t.Out(1) != _resMetaType || t.Out(2) != _errorType {
		panic(fmt.Sprintf(
			"the last two results of the handler for %q must be of type "+
				"yarpc.ResMeta and error, and not: %v, %v",
			n, t.Out(1), t.Out(2),
		))
	}

	reqBodyType := t.In(2)
	resBodyType := t.Out(0)

	if !isValidReqResType(reqBodyType) {
		panic(fmt.Sprintf(
			"the thrifd argument of the handler for %q must be "+
				"a struct pointer, a map[string]interface{}, or interface{}, and not: %v",
			n, reqBodyType,
		))
	}

	if !isValidReqResType(resBodyType) {
		panic(fmt.Sprintf(
			"the first result of the handler for %q must be "+
				"a struct pointer, a map[string]interface{}, or interface{], and not: %v",
			n, resBodyType,
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
