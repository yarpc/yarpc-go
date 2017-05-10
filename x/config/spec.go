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

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"

	"github.com/uber-go/mapdecode"
	"go.uber.org/multierr"
)

// TransportSpec specifies the configuration parameters for a transport. These
// specifications are registered against a Configurator to teach it how to
// parse the configuration for that transport and build instances of it.
//
// Every TransportSpec MUST have a BuildTransport function. The spec may
// provide BuildInbound, BuildUnaryOutbound, and BuildOnewayOutbound functions
// if the Transport supports that functoinality. For example, if a transport
// only supports incoming and outgoing Oneway requests, its spec will provide a
// BuildTransport, BuildInbound, and BuildOnewayOutbound function.
//
// The signature of BuildTransport must have the shape:
//
//  func(C, *config.Kit) (T, error)
//
// Where C is a struct defining the configuration parameters for the transport,
// the kit carries information and tools from the configurator to this and
// other builders, and T is the transport type.
//
// The remaining Build* functions must have a similar interface, but also carry
// the transport instance.
//
// Each Build* function has the shape:
//
// 	func(C, transport.Transport, *config.Kit) (X, error)
//
// Where X is the entity type, albeit an inbound or a unary or oneway outbound.
// For example,
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
// maps, slices, and other similar structs. For example only, an outbound might
// accept a config containing an array of host:port structs (In practice, an
// outbound would use a PeerListConfig to build a peer.ChooserList).
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
// Customizing Field Names
//
// If a field name differs from the name of the field inside the
// configuration data, a `config` tag may be added to the struct to specify a
// different name.
//
// 	type MyInboundConfig struct {
// 		Peer string `config:"peer"`
// 	}
//
// The configuration for this struct will be in the shape,
//
// 	myinbound:
// 	  peer: foo
//
// Runtime Variables
//
// In addition to specifying the field name, the `config` tag may also include
// an `interpolate` option to request interpolation of variables in the form
// ${NAME} or ${NAME:default} at the time the value is decoded. By default,
// environment variables are used to fill the variables; this may be changed
// with the InterpolationResolver option. The `interpolate` option may be
// applied to primitive types (strings, integers, booleans, floats, and
// time.Duration) only.
//
// For example in,
//
// 	type MyInboundConfig struct {
// 		QueueName string `config:"queue,interpolate"`
// 		Timeout time.Duration `config:",interpolate"`
// 		// If the name is left empty, the default name is used.
// 	}
//
// The values for both QueueName and Timeout may contain strings in the form
// ${NAME} to be replaced with the value of that environment variable. The
// form ${NAME:default} may be used to provide a default value if the
// environment variable is not set.
//
// 	myinbound:
// 	  queue: inbound-requests-${STAGE:dev}
// 	  timeout: ${REQUEST_TIMEOUT:5s}
//
// The above states that the queue inbound-requests-dev should be used by
// default, but if the STAGE environment variable is set, the queue
// inbound-requests-${STAGE} should be used. Similarly, it also states that a
// timeout of 5 seconds should be used by default, unless the REQUEST_TIMEOUT
// environment variable is set in which case the timeout specified in the
// environment variable should be used.
type TransportSpec struct {
	// Name of the transport
	Name string

	// A function in the shape,
	//
	// 	func(C, *config.Kit) (transport.Transport, error)
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
	// 	func(C, transport.Transport, *config.Kit) (transport.Inbound, error)
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
	// 	func(C, transport.Transport, *config.Kit) (transport.UnaryOutbound, error)
	// 	func(C, transport.Transport, *config.Kit) (transport.OnewayOutbound, error)
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

	// Named presets.
	//
	// These may be used by specifying a `with` key in the outbound
	// configuration.
	PeerListPresets []PeerListPreset

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

// PeerListPreset defines a named preset for a peer list. Peer list presets
// may be used by specifying a `with` key in the outbound configuration.
//
// 	http:
// 	  with: mypreset
type PeerListPreset struct {
	Name string

	// A function in the shape,
	//
	//  func(peer.Transport, *config.Kit) (peer.Chooser, error)
	//
	// Where the first argument is the transport object for which this preset
	// is being built.
	BuildPeerList interface{}

	// NOTE(abg): BuildPeerList /could/ be a well-defined func type rather
	// than an interface{}. We've kept it as an interface{} so that we have
	// the freedom to add more information to the functions in the future.
}

// PeerListSpec specifies the configuration parameters for an outbound peer
// chooser (load balancer or sharding). These specifications are registered
// against a Configurator to teach it how to parse the configuration for that
// peer chooser and build instances of it.
//
// For example, if we register a "dns-srv" peer list binder and a "random" peer
// chooser, we can use "with: dns-srv" and "choose: random" to select a random
// task (by host and port) from DNS A and SRV records for each outbound
// request.
//
//  myoutbound:
//   peers:
//    with: dns-srv
//    choose: random
//    service: fortune.yarpc.io
type PeerListSpec struct {
	Name string

	// A function in the shape,
	//
	//  func(C, *config.Kit) (peer.List, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters accepted by this peer chooser.
	//
	// BuildPeerList is required.
	BuildPeerList interface{}
}

// PeerListUpdaterSpec specifies the configuration parameters for an outbound peer
// binding (like DNS). These specifications are registered against a
// Configurator to teach it how to parse the configuration for that peer binder
// and build instances of it.
//
// Every PeerListUpdaterSpec MUST have a BuildPeerListUpdater function.
//
// For example, if we register a "dns-srv" peer list binder and a "random" peer
// chooser, we can use "with: dns-srv" and "choose: random" to select a random
// task (by host and port) from DNS A and SRV records for each outbound
// request.
//
//  myoutbound:
//   peers:
//    with: dns-srv
//    choose: random
//    service: fortune.yarpc.io
type PeerListUpdaterSpec struct {
	// Name of the peer selection strategy
	Name string

	// A function in the shape,
	//
	//  func(C, *config.Kit) (peer.Binder, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters accepted by this peer chooser.
	//
	// This function will be called with the parsed configuration to build a
	// peer chooser for an outbound that uses a peer chooser.
	//
	// For example, the HTTP and TChannel outbound configurations embed a peer
	// chooser configuration. Peer choosers support a single peer or arrays of
	// peers.  Using the "with" property, an outbound can use an alternate peer
	// chooser registered by name on a YARPC Configurator using a PeerListSpec.
	//
	// BuildPeerListUpdater is required.
	BuildPeerListUpdater interface{}
}

var (
	_typeOfError          = reflect.TypeOf((*error)(nil)).Elem()
	_typeOfTransport      = reflect.TypeOf((*transport.Transport)(nil)).Elem()
	_typeOfInbound        = reflect.TypeOf((*transport.Inbound)(nil)).Elem()
	_typeOfUnaryOutbound  = reflect.TypeOf((*transport.UnaryOutbound)(nil)).Elem()
	_typeOfOnewayOutbound = reflect.TypeOf((*transport.OnewayOutbound)(nil)).Elem()
	_typeOfPeerTransport  = reflect.TypeOf((*peer.Transport)(nil)).Elem()
	_typeOfPeerList       = reflect.TypeOf((*peer.ChooserList)(nil)).Elem()
	_typeOfPeerChooser    = reflect.TypeOf((*peer.Chooser)(nil)).Elem()
	_typeOfBinder         = reflect.TypeOf((*peer.Binder)(nil)).Elem()
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

	PeerListPresets map[string]*compiledPeerListPreset
}

func (s *compiledTransportSpec) SupportsUnaryOutbound() bool {
	return s.UnaryOutbound != nil
}

func (s *compiledTransportSpec) SupportsOnewayOutbound() bool {
	return s.OnewayOutbound != nil
}

func compileTransportSpec(spec *TransportSpec) (*compiledTransportSpec, error) {
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

	var err error

	// Helper to chain together the compile calls
	appendError := func(cs *configSpec, e error) *configSpec {
		err = multierr.Append(err, e)
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

	if len(spec.PeerListPresets) == 0 {
		return &out, err
	}

	presets := make(map[string]*compiledPeerListPreset, len(spec.PeerListPresets))
	out.PeerListPresets = presets
	for _, p := range spec.PeerListPresets {
		if _, ok := presets[p.Name]; ok {
			err = multierr.Append(err, fmt.Errorf(
				"found multiple peer lists with the name %q under transport %q",
				p.Name, spec.Name))
			continue
		}

		cp, e := compilePeerListPreset(p)
		if e != nil {
			err = multierr.Append(err, fmt.Errorf(
				"failed to compile preset for transport %q: %v", spec.Name, e))
			continue
		}

		presets[p.Name] = cp
	}

	return &out, err
}

func compileTransportConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	var err error
	switch {
	case t.Kind() != reflect.Func:
		err = errors.New("must be a function")
	case t.NumIn() != 2:
		err = fmt.Errorf("must accept exactly two arguments, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		err = fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfKit:
		err = fmt.Errorf("must accept a %v as its second argument, found %v", _typeOfKit, t.In(1))
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

	return &configSpec{inputType: t.In(0), factory: v}, nil
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

	return &configSpec{inputType: inputType, factory: v}, nil
}

func compileUnaryOutboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfUnaryOutbound); err != nil {
		return nil, fmt.Errorf("invalid BuildUnaryOutbound: %v", err)
	}

	return &configSpec{inputType: t.In(0), factory: v}, nil
}

func compileOnewayOutboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfOnewayOutbound); err != nil {
		return nil, fmt.Errorf("invalid BuildOnewayOutbound: %v", err)
	}

	return &configSpec{inputType: t.In(0), factory: v}, nil
}

// Common validation for all build functions except Tranport.
func validateConfigFunc(t reflect.Type, outputType reflect.Type) error {
	switch {
	case t.Kind() != reflect.Func:
		return errors.New("must be a function")
	case t.NumIn() != 3:
		return fmt.Errorf("must accept exactly three arguments, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		return fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfTransport:
		// TODO: We can make this smarter by making transport.Transport
		// optional and either the first or the second argument instead of
		// requiring it as the second argument.
		return fmt.Errorf("must accept a transport.Transport as its second argument, found %v", t.In(1))
	case t.In(2) != _typeOfKit:
		return fmt.Errorf("must accept a %v as its third argument, found %v", _typeOfKit, t.In(2))
	case t.NumOut() != 2:
		return fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != outputType:
		return fmt.Errorf("must return a %v as its first result, found %v", outputType, t.Out(0))
	case t.Out(1) != _typeOfError:
		return fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	return nil
}

type compiledPeerListPreset struct {
	name    string
	factory reflect.Value
}

// Build builds the peer.Chooser from the compiled peer list preset.
func (c *compiledPeerListPreset) Build(t peer.Transport, k *Kit) (peer.Chooser, error) {
	results := c.factory.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(k)})
	chooser, _ := results[0].Interface().(peer.Chooser)
	err, _ := results[1].Interface().(error)
	return chooser, err
}

func compilePeerListPreset(preset PeerListPreset) (*compiledPeerListPreset, error) {
	if preset.Name == "" {
		return nil, errors.New("Name is required")
	}

	if preset.BuildPeerList == nil {
		return nil, errors.New("BuildPeerList is required")
	}

	v := reflect.ValueOf(preset.BuildPeerList)
	t := v.Type()

	var err error
	switch {
	case t.Kind() != reflect.Func:
		err = errors.New("must be a function")
	case t.NumIn() != 2:
		err = fmt.Errorf("must accept exactly two arguments, found %v", t.NumIn())
	case t.In(0) != _typeOfPeerTransport:
		err = fmt.Errorf("must accept a peer.Transport as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfKit:
		err = fmt.Errorf("must accept a %v as its second argument, found %v", _typeOfKit, t.In(1))
	case t.NumOut() != 2:
		err = fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != _typeOfPeerChooser:
		err = fmt.Errorf("must return a peer.Chooser as its first result, found %v", t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid BuildPeerList %v: %v", t, err)
	}

	return &compiledPeerListPreset{name: preset.Name, factory: v}, nil
}

// Compiled internal representation of a user-specified PeerListSpec.
type compiledPeerListSpec struct {
	Name     string
	PeerList *configSpec
}

func compilePeerListSpec(spec *PeerListSpec) (*compiledPeerListSpec, error) {
	out := compiledPeerListSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("Name is required")
	}

	if spec.BuildPeerList == nil {
		return nil, errors.New("BuildPeerList is required")
	}

	buildPeerList, err := compilePeerListConfig(spec.BuildPeerList)
	if err != nil {
		return nil, err
	}
	out.PeerList = buildPeerList

	return &out, nil
}

func compilePeerListConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	var err error
	switch {
	case t.Kind() != reflect.Func:
		err = errors.New("must be a function")
	case t.NumIn() != 3:
		err = fmt.Errorf("must accept exactly three arguments, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		err = fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfPeerTransport:
		err = fmt.Errorf("must accept a %v as its second argument, found %v", _typeOfPeerTransport, t.In(1))
	case t.In(2) != _typeOfKit:
		err = fmt.Errorf("must accept a %v as its third argument, found %v", _typeOfKit, t.In(2))
	case t.NumOut() != 2:
		err = fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != _typeOfPeerList:
		err = fmt.Errorf("must return a peer.ChooserList as its first result, found %v", t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid BuildPeerList %v: %v", t, err)
	}

	return &configSpec{inputType: t.In(0), factory: v}, nil
}

type compiledPeerListUpdaterSpec struct {
	Name            string
	PeerListUpdater *configSpec
}

func compilePeerListUpdaterSpec(spec *PeerListUpdaterSpec) (*compiledPeerListUpdaterSpec, error) {
	out := compiledPeerListUpdaterSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("Name is required")
	}

	if spec.BuildPeerListUpdater == nil {
		return nil, errors.New("BuildPeerListUpdater is required")
	}

	buildPeerListUpdater, err := compilePeerListUpdaterConfig(spec.Name, spec.BuildPeerListUpdater)
	if err != nil {
		return nil, err
	}
	out.PeerListUpdater = buildPeerListUpdater

	return &out, nil
}

func compilePeerListUpdaterConfig(name string, build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	var err error
	switch {
	case t.Kind() != reflect.Func:
		err = errors.New("must be a function")
	case t.NumIn() != 2:
		err = fmt.Errorf("must accept exactly two arguments, found %v", t.NumIn())
	case !isDecodable(t.In(0)):
		err = fmt.Errorf("must accept a struct or struct pointer as its first argument, found %v", t.In(0))
	case t.In(1) != _typeOfKit:
		err = fmt.Errorf("must accept a %v as its second argument, found %v", _typeOfKit, t.In(1))
	case t.NumOut() != 2:
		err = fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != _typeOfBinder:
		err = fmt.Errorf("must return a peer.Binder as its first result, found %v", t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid BuildPeerListUpdater %v: %v", t, err)
	}

	return &configSpec{inputType: t.In(0), factory: v}, nil
}

// Validated representation of a configuration function specified by the user.
type configSpec struct {
	// Type of object expected by the factory function
	inputType reflect.Type

	// Factory function to call
	factory reflect.Value

	// Example:
	//
	// 	factory = func(http.InboundConfig, ..) (transport.Inbound, error) { .. }
	// 	inputType = http.InboundConfig
}

// Decode the configuration for this type from the data map.
func (cs *configSpec) Decode(attrs attributeMap, opts ...mapdecode.Option) (*buildable, error) {
	inputConfig := reflect.New(cs.inputType)
	if err := attrs.Decode(inputConfig.Interface(), opts...); err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", cs.inputType, err)
	}
	return &buildable{factory: cs.factory, inputData: inputConfig.Elem()}, nil
}

// A fully configured object that can be built into an
// Inbound/Outbound/Transport.
type buildable struct {
	// Decoded configuration data. This is a value of the same type as the
	// factory function's input argument.
	inputData reflect.Value

	// A function that accepts Config as its first argument and returns a
	// result and an error.
	//
	// Build(...) will call this function and interpret the result.
	factory reflect.Value

	// Example:
	//
	// 	factory = func(*http.InboundConfig, _) .. { .. }
	// 	inputData = &http.InboundConfig{Address: ..}
}

// Build the object configured by this value. The arguments are passed to the
// build function with the underlying configuration as the first parameter.
//
// Arguments may be reflect.Value objects or any other type.
func (cv *buildable) Build(args ...interface{}) (interface{}, error) {
	// This function roughly translates to,
	//
	// 	return factory(inputData, args...)

	callArgs := make([]reflect.Value, len(args)+1)
	callArgs[0] = cv.inputData

	for i, v := range args {
		if value, ok := v.(reflect.Value); ok {
			callArgs[i+1] = value
		} else {
			callArgs[i+1] = reflect.ValueOf(v)
		}
	}

	result := cv.factory.Call(callArgs)
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
