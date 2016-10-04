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

	"go.uber.org/yarpc/transport"
)

var (
	_clientFactories = make(map[reflect.Type]reflect.Value)
	_typeOfChannel   = reflect.TypeOf((*transport.Channel)(nil)).Elem()
)

func getFactoryType(f interface{}) reflect.Type {
	if f == nil {
		panic("f must not be nil")
	}

	fT := reflect.TypeOf(f)
	if fT.Kind() != reflect.Func {
		panic(fmt.Sprintf("f must be a function, not %T", f))
	}

	if fT.NumIn() != 1 || fT.In(0) != _typeOfChannel {
		panic(fmt.Sprintf("%v must accept only a transport.Channel", fT))
	}

	if fT.NumOut() != 1 || fT.Out(0).Kind() != reflect.Interface {
		panic(fmt.Sprintf("%v must return a single interface result", fT))
	}

	return fT.Out(0)
}

// RegisterClientFactory registers a factory function for a specific client
// type.
//
// Functions must have the signature,
//
// 	func(transport.Channel) T
//
// Where T is the type of the client. T MUST be an interface.
//
// This function panics if a client for the given type has already been
// registered.
//
// After a factory function for a client type is registered, these objects can
// be instantiated automatically using InjectClients.
//
// A function to unregister the factory function is returned. Note that the
// function will clear whatever the corresponding type's factory function is
// at the time it is called, regardless of whether the value matches what was
// passed to this function or not.
func RegisterClientFactory(f interface{}) (forget func()) {
	t := getFactoryType(f)
	if _, conflict := _clientFactories[t]; conflict {
		panic(fmt.Sprintf("a factory for %v has already been registered", t))
	}
	_clientFactories[t] = reflect.ValueOf(f)
	return func() { delete(_clientFactories, t) }
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
// 	h.KeyValueClient = keyvalueclient.New(dispatcher.Channel("keyvalue"))
// 	h.UserClient = json.New(dispatcher.Channel("users"))
//
// Factory functions for different client types may be registered using the
// RegisterClientFactory function. This function panics if an empty client
// field without a registered constructor is encountered.
func InjectClients(src transport.ChannelProvider, dest interface{}) {
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

		constructor, ok := _clientFactories[fieldT]
		if !ok {
			panic(fmt.Sprintf("a constructor for %v has not been registered", fieldT))
		}

		channelV := reflect.ValueOf(src.Channel(service))
		client := constructor.Call([]reflect.Value{channelV})[0]
		fieldV.Set(client)
	}
}
