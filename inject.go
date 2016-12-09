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

package yarpc

import (
	"fmt"
	"reflect"

	"go.uber.org/yarpc/api/transport"
)

var (
	_clientBuilders     = make(map[reflect.Type]reflect.Value)
	_typeOfClientConfig = reflect.TypeOf((*transport.ClientConfig)(nil)).Elem()
)

func getBuilderType(f interface{}) reflect.Type {
	if f == nil {
		panic("f must not be nil")
	}

	fT := reflect.TypeOf(f)
	if fT.Kind() != reflect.Func {
		panic(fmt.Sprintf("f must be a function, not %T", f))
	}

	if fT.NumIn() != 1 || fT.In(0) != _typeOfClientConfig {
		panic(fmt.Sprintf("%v must accept only a transport.ClientConfig", fT))
	}

	if fT.NumOut() != 1 || fT.Out(0).Kind() != reflect.Interface {
		panic(fmt.Sprintf("%v must return a single interface result", fT))
	}

	return fT.Out(0)
}

// RegisterClientBuilder registers a builder function for a specific client
// type.
//
// Functions must have the signature,
//
// 	func(transport.ClientConfig) T
//
// Where T is the type of the client. T MUST be an interface.
//
// This function panics if a client for the given type has already been
// registered.
//
// After a builder function for a client type is registered, these objects can
// be instantiated automatically using InjectClients.
//
// A function to unregister the builder function is returned. Note that the
// function will clear whatever the corresponding type's builder function is
// at the time it is called, regardless of whether the value matches what was
// passed to this function or not.
func RegisterClientBuilder(f interface{}) (forget func()) {
	t := getBuilderType(f)
	if _, conflict := _clientBuilders[t]; conflict {
		panic(fmt.Sprintf("a builder for %v has already been registered", t))
	}
	_clientBuilders[t] = reflect.ValueOf(f)
	return func() { delete(_clientBuilders, t) }
}

// InjectClients injects clients from the given Dispatcher into the given
// struct. dest must be a pointer to a struct with zero or more exported
// fields Thrift client fields. Only fields with nil values and a `service`
// tag will be populated; everything else will be left unchanged.
//
// 	type Handler struct {
// 		KeyValueClient keyvalueclient.Interface `service:"keyvalue"`
// 		UserClient json.Client `service:"users"`
// 		TagClient tagclient.Interface  // will not be changed
// 	}
//
// 	var h Handler
// 	yarpc.InjectClients(dispatcher, &h)
//
// 	// InjectClients above is equivalent to,
//
// 	h.KeyValueClient = keyvalueclient.New(dispatcher.ClientConfig("keyvalue"))
// 	h.UserClient = json.New(dispatcher.ClientConfig("users"))
//
// Builder functions for different client types may be registered using the
// RegisterClientBuilder function. This function panics if an empty client
// field without a registered constructor is encountered.
func InjectClients(src transport.ClientConfigProvider, dest interface{}) {
	destV := reflect.ValueOf(dest)
	destT := reflect.TypeOf(dest)
	if destT.Kind() != reflect.Ptr || destT.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("dest must be a pointer to a struct, not %T", dest))
	}

	structV := destV.Elem()
	structT := destT.Elem()
	for i := 0; i < structV.NumField(); i++ {
		fieldInfo := structT.Field(i)
		fieldV := structV.Field(i)

		if !fieldV.CanSet() {
			continue
		}

		fieldT := fieldInfo.Type
		if fieldT.Kind() != reflect.Interface {
			continue
		}

		service := fieldInfo.Tag.Get("service")
		if service == "" {
			continue
		}

		if !fieldV.IsNil() {
			continue
		}

		constructor, ok := _clientBuilders[fieldT]
		if !ok {
			panic(fmt.Sprintf("a constructor for %v has not been registered", fieldT))
		}

		clientConfigV := reflect.ValueOf(src.ClientConfig(service))
		client := constructor.Call([]reflect.Value{clientConfigV})[0]
		fieldV.Set(client)
	}
}
