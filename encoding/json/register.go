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

	"github.com/yarpc/yarpc-go/transport"
)

var (
	_requestType        = reflect.TypeOf((*Request)(nil))
	_responseType       = reflect.TypeOf((*Response)(nil))
	_errorType          = reflect.TypeOf((*error)(nil)).Elem()
	_interfaceEmptyType = reflect.TypeOf((*interface{})(nil)).Elem()
)

// Registrant is used for types that define or know about different JSON
// procedures.
type Registrant interface {
	// Gets a mapping from procedure name to the handler for that procedure for
	// all procedures provided by this registrant.
	getHandlers() map[string]interface{}
}

// procedure is a simple Registrant that has a single procedure.
type procedure struct {
	Name    string
	Handler interface{}
}

func (p procedure) getHandlers() map[string]interface{} {
	return map[string]interface{}{p.Name: p.Handler}
}

// Procedure builds a Registrant with a single procedure in it. handler must
// be a function with a signature similar to,
//
// 	f(req *json.Request, body $reqBody) ($resBody, *json.Response, error)
//
// Where $reqBody and $resBody are a map[string]interface{} or pointers to
// structs.
func Procedure(name string, handler interface{}) Registrant {
	return procedure{Name: name, Handler: handler}
}

// Register registers the procedures defined by the given JSON registrant with
// the given registry.
//
// Handlers must have a signature similar to the following or the system will
// panic.
//
// 	f(req *json.Request, body $reqBody) ($resBody, *json.Response, error)
//
// Where $reqBody and $resBody are a map[string]interface{} or pointers to
// structs.
func Register(reg transport.Registry, registrant Registrant) {
	for name, handler := range registrant.getHandlers() {
		reg.Register("", name, wrapHandler(name, handler))
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

	if t.NumIn() != 2 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 2 arguments but it had %v",
			n, t.NumIn(),
		))
	}

	if t.NumOut() != 3 {
		panic(fmt.Sprintf(
			"expected handler for %q to have 3 results but it had %v",
			n, t.NumOut(),
		))
	}

	if t.In(0) != _requestType {
		panic(fmt.Sprintf(
			"the first argument of the handler for %q must be of type "+
				"*json.Request, and not: %v", n, t.In(0),
		))
	}

	if t.Out(1) != _responseType || t.Out(2) != _errorType {
		panic(fmt.Sprintf(
			"the last two results of the handler for %q must be of type "+
				"*json.Response and error, and not: %v, %v",
			n, t.Out(1), t.Out(2),
		))
	}

	reqBodyType := t.In(1)
	resBodyType := t.Out(0)

	if !isValidReqResType(reqBodyType) {
		panic(fmt.Sprintf(
			"the second argument of the handler for %q must be "+
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
