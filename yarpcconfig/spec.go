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

package yarpcconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/uber-go/mapdecode"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/config"
)

// TransportSpec specifies the configuration parameters for a transport. These
// specifications are registered against a Configurator to teach it how to
// parse the configuration for that transport and build instances of it.
//
// Every TransportSpec MUST have a BuildTransport function. The spec may
// provide BuildInbound, BuildUnaryOutbound, and BuildOnewayOutbound functions
// if the Transport supports that functionality. For example, if a transport
// only supports incoming and outgoing Oneway requests, its spec will provide a
// BuildTransport, BuildInbound, and BuildOnewayOutbound function.
//
// The signature of BuildTransport must have the shape:
//
//  func(C, *config.Kit) (transport.Transport, error)
//
// Where C is a struct defining the configuration parameters for the transport,
// the kit carries information and tools from the configurator to this and
// other builders.
//
// The remaining Build* functions must have a similar signature, but also
// receive the transport instance.
//
// 	func(C, transport.Transport, *config.Kit) (X, error)
//
// Where X is one of, transport.Inbound, transport.UnaryOutbound, or
// transport.OnewayOutbound.
//
// For example,
//
// 	func(*OutboundConfig, transport.Transport) (transport.UnaryOutbound, error)
//
// Is a function to build a unary outbound from its outbound configuration and
// the corresponding transport.
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

	// TODO(abg): Make error returns optional

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
	BuildStreamOutbound interface{}

	// Named presets.
	//
	// These may be used by specifying a `with` key in the outbound
	// configuration.
	PeerChooserPresets []PeerChooserPreset

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

// PeerChooserPreset defines a named preset for a peer chooser. Peer chooser
// presets may be used by specifying a `with` key in the outbound
// configuration.
//
// 	http:
// 	  with: mypreset
type PeerChooserPreset struct {
	Name string

	// A function in the shape,
	//
	//  func(peer.Transport, *config.Kit) (peer.Chooser, error)
	//
	// Where the first argument is the transport object for which this preset
	// is being built.
	BuildPeerChooser interface{}

	// NOTE(abg): BuildChooser /could/ be a well-defined func type rather
	// than an interface{}. We've kept it as an interface{} so that we have
	// the freedom to add more information to the functions in the future.
}

// PeerChooserSpec specifies the configuration parameters for an outbound peer
// chooser. Peer choosers dictate how peers are selected for an outbound. These
// specifications are registered against a Configurator to teach it how to parse
// the configuration for that peer chooser and build instances of it.
//
// For example, we could implement and register a peer chooser spec that selects
// peers based on advanced configuration or sharding information.
//
// 	myoutbound:
// 	  tchannel:
// 	    mysharder:
//        shard1: 1.1.1.1:1234
//        ...
type PeerChooserSpec struct {
	Name string

	// A function in the shape,
	//
	//  func(C, p peer.Transport, *config.Kit) (peer.Chooser, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters needed to build this peer chooser.
	//
	// BuildPeerChooser is required.
	BuildPeerChooser interface{}
}

// PeerListSpec specifies the configuration parameters for an outbound peer
// list. Peer lists dictate the peer selection strategy and receive updates of
// new and removed peers from peer updaters. These specifications are
// registered against a Configurator to teach it how to parse the
// configuration for that peer list and build instances of it.
//
// For example, we could implement and register a peer list spec that selects
// peers at random and a peer list updater which pushes updates to it by
// polling a specific DNS A record.
//
// 	myoutbound:
// 	  random:
// 	    dns:
// 	      name: myservice.example.com
type PeerListSpec struct {
	Name string

	// A function in the shape,
	//
	//  func(C, peer.Transport, *config.Kit) (peer.ChooserList, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters needed to build this peer list. Parameters on the struct
	// should not conflict with peer list updater names as they share the
	// namespace with these fields.
	//
	// BuildPeerList is required.
	BuildPeerList interface{}
}

// PeerListUpdaterSpec specifies the configuration parameters for an outbound
// peer list updater. Peer list updaters inform peer lists about peers as they
// are added or removed. These specifications are registered against a
// Configurator to teach it how to parse the configuration for that peer list
// updater and build instances of it.
//
// For example, we could implement a peer list updater which monitors a
// specific file on the system for a list of peers and pushes updates to any
// peer list.
//
// 	myoutbound:
// 	  round-robin:
// 	    peers-file:
// 	      format: json
// 	      path: /etc/hosts.json
type PeerListUpdaterSpec struct {
	// Name of the peer selection strategy.
	Name string

	// A function in the shape,
	//
	//  func(C, *config.Kit) (peer.Binder, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters accepted by this peer chooser.
	//
	// The returned peer binder will receive the peer list specified alongside
	// the peer updater; it should return a peer updater that feeds updates to
	// that peer list once started.
	//
	// BuildPeerListUpdater is required.
	BuildPeerListUpdater interface{}
}

var (
	_typeOfError           = reflect.TypeOf((*error)(nil)).Elem()
	_typeOfTransport       = reflect.TypeOf((*transport.Transport)(nil)).Elem()
	_typeOfInbound         = reflect.TypeOf((*transport.Inbound)(nil)).Elem()
	_typeOfUnaryOutbound   = reflect.TypeOf((*transport.UnaryOutbound)(nil)).Elem()
	_typeOfOnewayOutbound  = reflect.TypeOf((*transport.OnewayOutbound)(nil)).Elem()
	_typeOfStreamOutbound  = reflect.TypeOf((*transport.StreamOutbound)(nil)).Elem()
	_typeOfPeerTransport   = reflect.TypeOf((*peer.Transport)(nil)).Elem()
	_typeOfPeerChooserList = reflect.TypeOf((*peer.ChooserList)(nil)).Elem()
	_typeOfPeerChooser     = reflect.TypeOf((*peer.Chooser)(nil)).Elem()
	_typeOfBinder          = reflect.TypeOf((*peer.Binder)(nil)).Elem()
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
	StreamOutbound *configSpec

	PeerChooserPresets map[string]*compiledPeerChooserPreset
}

func (s *compiledTransportSpec) SupportsUnaryOutbound() bool {
	return s.UnaryOutbound != nil
}

func (s *compiledTransportSpec) SupportsOnewayOutbound() bool {
	return s.OnewayOutbound != nil
}

func (s *compiledTransportSpec) SupportsStreamOutbound() bool {
	return s.StreamOutbound != nil
}

func compileTransportSpec(spec *TransportSpec) (*compiledTransportSpec, error) {
	out := compiledTransportSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("field Name is required")
	}

	switch strings.ToLower(spec.Name) {
	case "unary", "oneway", "stream":
		return nil, fmt.Errorf("transport name cannot be %q: %q is a reserved name", spec.Name, spec.Name)
	}

	if spec.BuildTransport == nil {
		return nil, errors.New("field BuildTransport is required")
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
	if spec.BuildStreamOutbound != nil {
		out.StreamOutbound = appendError(compileStreamOutboundConfig(spec.BuildStreamOutbound))
	}

	if len(spec.PeerChooserPresets) == 0 {
		return &out, err
	}

	presets := make(map[string]*compiledPeerChooserPreset, len(spec.PeerChooserPresets))
	out.PeerChooserPresets = presets
	for _, p := range spec.PeerChooserPresets {
		if _, ok := presets[p.Name]; ok {
			err = multierr.Append(err, fmt.Errorf(
				"found multiple peer lists with the name %q under transport %q",
				p.Name, spec.Name))
			continue
		}

		cp, e := compilePeerChooserPreset(p)
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

func compileStreamOutboundConfig(build interface{}) (*configSpec, error) {
	v := reflect.ValueOf(build)
	t := v.Type()

	if err := validateConfigFunc(t, _typeOfStreamOutbound); err != nil {
		return nil, fmt.Errorf("invalid BuildStreamOutbound: %v", err)
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

type compiledPeerChooserPreset struct {
	name    string
	factory reflect.Value
}

// Build builds the peer.Chooser from the compiled peer chooser preset.
func (c *compiledPeerChooserPreset) Build(t peer.Transport, k *Kit) (peer.Chooser, error) {
	results := c.factory.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(k)})
	chooser, _ := results[0].Interface().(peer.Chooser)
	err, _ := results[1].Interface().(error)
	return chooser, err
}

func compilePeerChooserPreset(preset PeerChooserPreset) (*compiledPeerChooserPreset, error) {
	if preset.Name == "" {
		return nil, errors.New("field Name is required")
	}

	if preset.BuildPeerChooser == nil {
		return nil, errors.New("field BuildPeerChooser is required")
	}

	v := reflect.ValueOf(preset.BuildPeerChooser)
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
		return nil, fmt.Errorf("invalid BuildPeerChooser %v: %v", t, err)
	}

	return &compiledPeerChooserPreset{name: preset.Name, factory: v}, nil
}

// Compiled internal representation of a user-specified PeerChooserSpec.
type compiledPeerChooserSpec struct {
	Name        string
	PeerChooser *configSpec
}

func compilePeerChooserSpec(spec *PeerChooserSpec) (*compiledPeerChooserSpec, error) {
	out := compiledPeerChooserSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("field Name is required")
	}

	if spec.BuildPeerChooser == nil {
		return nil, errors.New("field BuildPeerChooser is required")
	}

	buildPeerChooser, err := compilePeerChooserConfig(spec.BuildPeerChooser)
	if err != nil {
		return nil, err
	}
	out.PeerChooser = buildPeerChooser

	return &out, nil
}

func compilePeerChooserConfig(build interface{}) (*configSpec, error) {
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
	case t.Out(0) != _typeOfPeerChooser:
		err = fmt.Errorf("must return a peer.Chooser as its first result, found %v", t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return an error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		return nil, fmt.Errorf("invalid BuildPeerChooser %v: %v", t, err)
	}

	return &configSpec{inputType: t.In(0), factory: v}, nil
}

// Compiled internal representation of a user-specified PeerListSpec.
type compiledPeerListSpec struct {
	Name     string
	PeerList *configSpec
}

func compilePeerListSpec(spec *PeerListSpec) (*compiledPeerListSpec, error) {
	out := compiledPeerListSpec{Name: spec.Name}

	if spec.Name == "" {
		return nil, errors.New("field Name is required")
	}

	if spec.BuildPeerList == nil {
		return nil, errors.New("field BuildPeerList is required")
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
	case t.Out(0) != _typeOfPeerChooserList:
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
		return nil, errors.New("field Name is required")
	}

	if spec.BuildPeerListUpdater == nil {
		return nil, errors.New("field BuildPeerListUpdater is required")
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
func (cs *configSpec) Decode(attrs config.AttributeMap, opts ...mapdecode.Option) (*buildable, error) {
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
