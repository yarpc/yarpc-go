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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/yarpc/api/transport"
	errs "go.uber.org/yarpc/internal/errors"
)

// TransportSpec specifies the configuration parameters for a transport. These
// specifications are registered against a Configurator to teach it how to
// parse the configuration for that transport and build instances of it.
//
// Every TransportSpec MUST have a BuildTransport function. BuildInbound,
// BuildUnaryOutbound, and BuildOnewayOutbound functions may be provided if
// the Transport supports that functoinality. For example, if a transport only
// supports incoming and outgoing Oneway requests, it will provide a
// BuildTransport, BuildInbound, and BuildOnewayOutbound function.
//
// Besides BuildTransport which accepts just its configuration struct, each
// function mentioned above has the shape,
//
// 	func(C, transport.Transport) (X, error)
//
// Where C is a struct defining the configuration parameters of that entity
// and X is the result type. For example,
//
// 	func(HttpOutboundConfig, transport.Transport) (transport.UnaryOutbound, error)
//
// Is a function to build an HTTP unary outbound from its outbound
// configuration and the corresponding transport.
//
// The Configurator will decode and fill the requested struct type from the
// input configuration. For example, given,
//
// 	type HttpOutboundConfig struct {
// 		URL string
// 	}
//
// Configurator expects the outbound configuration for HTTP to have a 'url'
// field. In YAML, the following,
//
// 	outbounds:
// 	  myservice:
// 	    http:
// 	      url: http://localhost:8080
//
// Will be decoded into,
//
//	HttpOutboundConfig{URL: "http://localhost:8080"}
//
// A case-insensitive match is performed to map fields from configuration data
// to structs.
//
// Configuration structs can use standard Go primitive types, time.Duration,
// maps, slices, and other similar structs. For example,
//
// 	type Peer struct {
// 		Host string
// 		Port int
// 	}
//
// 	type MyOutboundConfig struct{ Peers []Peer }
//
// Will expect the following YAML.
//
// 	myoutbound:
// 	  peers:
// 		- host: localhost
// 		  port: 8080
// 		- host: anotherhost
// 		  port: 8080
//
// If a field name differs from the name of the field inside the
// configuration data, a `config` tag may be added to the struct to specify a
// different name.
//
// 	type MyInboundConfig struct {
// 		Address string `config:"addr"`
// 	}
//
// The configuration for this struct will be in the shape,
//
// 	myinbound:
// 	  addr: foo
type TransportSpec struct {
	// Name of the transport
	Name string

	// A function in the shape,
	//
	// 	func(C) (transport.Transport, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters accepted by this transport.
	//
	// This function will be called with the parsed configuration to build
	// Transport defined by this spec.
	BuildTransport interface{}

	// TODO(abg): Make error returns optional -- if the function doesn't
	// return an error value, we can just wrap it to always return nil there.

	// A function in the shape,
	//
	// 	func(C, transport.Transport) (transport.Inbound, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters for the inbound.
	//
	// This may be nil if this transport does not support inbounds.
	//
	// This function will be called with the parsed configuration and the
	// transport built by BuildTransport to build the inbound for this
	// transport.
	BuildInbound interface{}

	// The following two are functions in the shapes,
	//
	// 	func(C, transport.Transport) (transport.UnaryOutbound, error)
	// 	func(C, transport.Transport) (transport.OnewayOutbound, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters for outbounds of that RPC type.
	//
	// Either value may be nil to indicate that the transport does not support
	// unary or oneway outbounds.
	//
	// These functions will be called with the parsed configurations and the
	// transport built by BuildTransport to build the unary and oneway
	// outbounds for this transport.
	BuildUnaryOutbound  interface{}
	BuildOnewayOutbound interface{}

	// TODO(abg): Allow functions to return and accept specific
	// implementations. Instead of returning a transport.Transport and
	// accepting a transport.Transport, we could make it so that
	//
	// 	BuildTransport: func(...) (*http.Transport, error)
	// 	BuildInbound: func(..., t *http.Transport) (*http.Inbound, error)
	//
	// This will get rid of the `t.(*http.Transport)` users will have to do
	// the first thing inside their BuildInbound.
}

var (
	_typeOfError          = reflect.TypeOf((*error)(nil)).Elem()
	_typeOfTransport      = reflect.TypeOf((*transport.Transport)(nil)).Elem()
	_typeOfInbound        = reflect.TypeOf((*transport.Inbound)(nil)).Elem()
	_typeOfUnaryOutbound  = reflect.TypeOf((*transport.UnaryOutbound)(nil)).Elem()
	_typeOfOnewayOutbound = reflect.TypeOf((*transport.OnewayOutbound)(nil)).Elem()
)

// Compiled internal representation of a user-specified TransportSpec.
type compiledTransportSpec struct {
	Name string // name of the transport

	// configSpec of the top-level transport object
	Transport *configSpec

	// The following are non-nil only if the transport supports that specific
	// functionality.

	Inbound        *configSpec
	UnaryOutbound  *configSpec
	OnewayOutbound *configSpec
}

func (s *compiledTransportSpec) SupportsUnaryOutbound() bool {
	return s.UnaryOutbound != nil
}

func (s *compiledTransportSpec) SupportsOnewayOutbound() bool {
	return s.OnewayOutbound != nil
}

func compileTransportSpec(spec *TransportSpec) (_ *compiledTransportSpec, err error) {
	out := compiledTransportSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("Name is required")
	}

	switch strings.ToLower(spec.Name) {
	case "unary", "oneway":
		return nil, fmt.Errorf("transport name cannot be %q: %q is a reserved name", spec.Name, spec.Name)
	}

	if spec.BuildTransport == nil {
		return nil, errors.New("BuildTransport is required")
	}

	var errors []error

	// Helper to chain together the compile calls
	appendError := func(cs *configSpec, err error) *configSpec {
		if err != nil {
			errors = append(errors, err)
		}
		return cs
	}

	out.Transport = appendError(compileTransportConfig(spec.BuildTransport))
	if spec.BuildInbound != nil {
		out.Inbound = appendError(compileInboundConfig(spec.BuildInbound))
	}
	if spec.BuildUnaryOutbound != nil {
		out.UnaryOutbound = appendError(compileUnaryOutboundConfig(spec.BuildUnaryOutbound))
	}
	if spec.BuildOnewayOutbound != nil {
		out.OnewayOutbound = appendError(compileOnewayOutboundConfig(spec.BuildOnewayOutbound))
	}
	return &out, errs.CombineErrors(errors...)
}

func compileTransportConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	var err error
	switch {
	case t.Kind() != reflect.Func:
		err = errors.New("must be a function")
	case t.NumIn() != 1:
		err = fmt.Errorf("must accept exactly one argument, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		err = fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.NumOut() != 2:
		err = fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != _typeOfTransport:
		err = fmt.Errorf("must return a transport.Transport as its first result, found %v", t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid BuildTransport %v: %v", t, err)
	}

	return &configSpec{inputType: t.In(0), builder: v}, nil
}

func compileInboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfInbound); err != nil {
		return nil, fmt.Errorf("invalid BuildInbound: %v", err)
	}

	inputType := t.In(0)

	fields := fieldNames(inputType)
	if _, hasType := fields["Type"]; hasType {
		return nil, errors.New("inbound configurations must not have a Type field: Type is a reserved field name")
	}

	if _, hasDisabled := fields["Disabled"]; hasDisabled {
		return nil, errors.New("inbound configurations must not have a Disabled field: Disabled is a reserved field name")
	}

	return &configSpec{inputType: inputType, builder: v}, nil
}

func compileUnaryOutboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfUnaryOutbound); err != nil {
		return nil, fmt.Errorf("invalid BuildUnaryOutbound: %v", err)
	}

	return &configSpec{inputType: t.In(0), builder: v}, nil
}

func compileOnewayOutboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfOnewayOutbound); err != nil {
		return nil, fmt.Errorf("invalid BuildOnewayOutbound: %v", err)
	}

	return &configSpec{inputType: t.In(0), builder: v}, nil
}

// Common validation for all build functions except Tranport.
func validateConfigFunc(t reflect.Type, outputType reflect.Type) error {
	switch {
	case t.Kind() != reflect.Func:
		return errors.New("must be a function")
	case t.NumIn() != 2:
		return fmt.Errorf("must accept exactly two arguments, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		return fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfTransport:
		// TODO: We can make this smarter by making transport.Transport
		// optional and either the first or the second argument instead of
		// requiring it as the second argument.
		return fmt.Errorf("must accept a transport.Transport as its second argument, found %v", t.In(1))
	case t.NumOut() != 2:
		return fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != outputType:
		return fmt.Errorf("must return a %v as its first result, found %v", outputType, t.Out(0))
	case t.Out(1) != _typeOfError:
		return fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	return nil
}

// Validated representation of a configuration function specified by the user.
type configSpec struct {
	inputType reflect.Type  // type of config object expected by the function
	builder   reflect.Value // function to call
}

// Decode the configuration for this type from the data map.
func (cs *configSpec) Decode(attrs attributeMap) (*configuredValue, error) {
	result := reflect.New(cs.inputType)
	if err := attrs.Decode(result.Interface()); err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", cs.inputType, err)
	}
	return &configuredValue{builder: cs.builder, data: result.Elem()}, nil
}

// A single parsed configuration.
type configuredValue struct {
	// Decoded configuration data
	data reflect.Value

	// A function that accepts Config as its first argument and returns a
	// result and an error.
	//
	// 	func(Config, ...) (Out, error)
	//
	// Build(...) will call this function and interpret the result.
	builder reflect.Value
}

// Build the object configured by this value. The arguments are passed to the
// build function with the underlying configuration as the first parameter.
//
// Arguments may be reflect.Value objects or any other type.
func (cv *configuredValue) Build(args ...interface{}) (interface{}, error) {
	callArgs := make([]reflect.Value, len(args)+1)
	callArgs[0] = cv.data

	for i, v := range args {
		if value, ok := v.(reflect.Value); ok {
			callArgs[i+1] = value
		} else {
			callArgs[i+1] = reflect.ValueOf(v)
		}
	}

	result := cv.builder.Call(callArgs)
	err, _ := result[1].Interface().(error)
	return result[0].Interface(), err
}

// Returns a list of struct fields for the given type. The type may be a
// struct or a pointer to a struct (arbitrarily deep).
func fieldNames(t reflect.Type) map[string]struct{} {
	for ; t.Kind() == reflect.Ptr; t = t.Elem() {
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	fields := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue // unexported field
		}
		fields[field.Name] = struct{}{}
	}
	return fields
}

func isDecodable(t reflect.Type) bool {
	for ; t.Kind() == reflect.Ptr; t = t.Elem() {
	}

	// TODO(abg): Do we want to support top-level map types for configuration

	return t.Kind() == reflect.Struct
}
