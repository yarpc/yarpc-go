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
	"fmt"
	"reflect"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

var (
	_contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	_metaType    = reflect.TypeOf((*yarpc.Meta)(nil)).Elem()
	_errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// Registrant is used for types that define or know about different JSON
// procedures.
type Registrant interface {
	// Gets a mapping from procedure name to the handler for that procedure for
	// all procedures provided by this registrant.
	//
	// Handlers must have a signature similar to the following or the system
	// will panic.
	//
	// 	f(context.Context, yarpc.Meta, req $request) ($response, yarpc.Meta, error)
	//
	// Where $request and $response are a map[string]interface{} or pointers to
	// structs.
	GetHandlers() map[string]interface{}
}

// registrant is a simple Registrant that has a hard-coded list of handlers.
type registrant struct {
	handlers map[string]interface{}
}

func (r registrant) GetHandlers() map[string]interface{} {
	return r.handlers
}

// Procedure builds a Registrant with a single procedure in it.
//
// handler must be a function with a signature similar to,
//
// 	f(context.Context, yarpc.Meta, req $request) ($response, yarpc.Meta, error)
//
// Where $request and $response are a map[string]interface{} or pointers to
// structs.
func Procedure(name string, handler interface{}) Registrant {
	return registrant{handlers: map[string]interface{}{name: handler}}
}

// Register registers the procedures defined by the given JSON registrant with
// the given YARPC.
func Register(reg transport.Registry, registrant Registrant) {
	for name, handler := range registrant.GetHandlers() {
		reqType, resType := verifySignature(name, reflect.TypeOf(handler))
		reg.Register(name, jsonHandler{
			RequestType:  reqType,
			ResponseType: resType,
			Handler:      handler,
		})
	}
}

// verifySignature verifies that the given type matches what we expect from
// JSON handlers and returns the request and response types.
func verifySignature(n string, t reflect.Type) (reflect.Type, reflect.Type) {
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

	if t.In(0) != _contextType || t.In(1) != _metaType {
		panic(fmt.Sprintf(
			"the first two arguments of the handler for %q must be "+
				"context.Context and yarpc.Meta, and not: %v, %v",
			n, t.In(0), t.In(1),
		))
	}

	if t.Out(1) != _metaType || t.Out(2) != _errorType {
		panic(fmt.Sprintf(
			"the last two results of the handler for %q must be "+
				"yarpc.Meta and error, and not: %v, %v",
			n, t.Out(1), t.Out(2),
		))
	}

	reqType := t.In(2)
	resType := t.Out(0)

	if !isValidReqResType(reqType) {
		panic(fmt.Sprintf(
			"the third argument of the handler for %q must be "+
				"a struct pointer or a map[string]interface{}, and not: %v",
			n, reqType,
		))
	}

	if !isValidReqResType(resType) {
		panic(fmt.Sprintf(
			"the first result of the handler for %q must be "+
				"a struct pointer or a map[string]interface{}, and not: %v",
			n, resType,
		))
	}

	return reqType, resType
}

// isValidReqResType checks if the given type is a pointer to a struct or a
// map[string]interface{}.
func isValidReqResType(t reflect.Type) bool {
	return (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) ||
		(t.Kind() == reflect.Map && t.Key().Kind() == reflect.String)
}
