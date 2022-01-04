// Copyright (c) 2022 Uber Technologies, Inc.
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
	// _clientBuilders is a map from type of our desired client 'T' to a
	// (reflected) function with one of the following signatures,
	//
	// 	func(transport.ClientConfig) T
	// 	func(transport.ClientConfig, reflect.StructField) T
	//
	// Where T is the same as the key type for that entry.
	_clientBuilders = make(map[reflect.Type]reflect.Value)

	_typeOfClientConfig = reflect.TypeOf((*transport.ClientConfig)(nil)).Elem()
	_typeOfStructField  = reflect.TypeOf(reflect.StructField{})
)

func validateClientBuilder(f interface{}) reflect.Value {
	if f == nil {
		panic("must not be nil")
	}

	fv := reflect.ValueOf(f)
	ft := fv.Type()
	switch {
	case ft.Kind() != reflect.Func:
		panic(fmt.Sprintf("must be a function, not %v", ft))

	// Validate number of arguments and results
	case ft.NumIn() == 0:
		panic("must accept at least one argument")
	case ft.NumIn() > 2:
		panic(fmt.Sprintf("must accept at most two arguments, got %v", ft.NumIn()))
	case ft.NumOut() != 1:
		panic(fmt.Sprintf("must return exactly one result, got %v", ft.NumOut()))

	// Validate input and output types
	case ft.In(0) != _typeOfClientConfig:
		panic(fmt.Sprintf("must accept a transport.ClientConfig as its first argument, got %v", ft.In(0)))
	case ft.NumIn() == 2 && ft.In(1) != _typeOfStructField:
		panic(fmt.Sprintf("if a second argument is accepted, it must be a reflect.StructField, got %v", ft.In(1)))
	case ft.Out(0).Kind() != reflect.Interface:
		panic(fmt.Sprintf("must return a single interface type as a result, got %v", ft.Out(0).Kind()))
	}

	return fv
}

// RegisterClientBuilder registers a builder function for a specific client
// type.
//
// Functions must have one of the following signatures:
//
// 	func(transport.ClientConfig) T
// 	func(transport.ClientConfig, reflect.StructField) T
//
// Where T is the type of the client. T MUST be an interface. In the second
// form, the function receives type information about the field being filled.
// It may inspect the struct tags to customize its behavior.
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
	fv := validateClientBuilder(f)
	t := fv.Type().Out(0)

	if _, conflict := _clientBuilders[t]; conflict {
		panic(fmt.Sprintf("a builder for %v has already been registered", t))
	}

	_clientBuilders[t] = fv
	return func() { delete(_clientBuilders, t) }
}

// InjectClients injects clients from a Dispatcher into the given struct. dest
// must be a pointer to a struct with zero or more exported fields which hold
// YARPC client types. This includes json.Client, raw.Client, and any
// generated Thrift service client. Fields with nil values and a `service` tag
// will be populated with clients using that service`s ClientConfig.
//
// Given,
//
// 	type Handler struct {
// 		KeyValueClient keyvalueclient.Interface `service:"keyvalue"`
// 		UserClient json.Client `service:"users"`
// 		TagClient tagclient.Interface  // no tag; will be left unchanged
// 	}
//
// The call,
//
// 	var h Handler
// 	yarpc.InjectClients(dispatcher, &h)
//
// Is equivalent to,
//
// 	var h Handler
// 	h.KeyValueClient = keyvalueclient.New(dispatcher.ClientConfig("keyvalue"))
// 	h.UserClient = json.New(dispatcher.ClientConfig("users"))
//
// Builder functions for different client types may be registered using the
// RegisterClientBuilder function.
//
// This function panics if a field with an unknown type and nil value has the
// `service` tag.
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

		builder, ok := _clientBuilders[fieldT]
		if !ok {
			panic(fmt.Sprintf("a constructor for %v has not been registered", fieldT))
		}
		builderT := builder.Type()

		args := make([]reflect.Value, 1, builderT.NumIn())
		args[0] = reflect.ValueOf(src.ClientConfig(service))
		if builderT.NumIn() > 1 {
			args = append(args, reflect.ValueOf(fieldInfo))
		}

		client := builder.Call(args)[0]
		fieldV.Set(client)
	}
}
