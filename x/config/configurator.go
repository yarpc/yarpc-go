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
	"io"
	"io/ioutil"
	"reflect"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/decode"

	"gopkg.in/yaml.v2"
)

// Configurator helps build Dispatchers using runtime configuration.
type Configurator struct {
	knownTransports map[string]*transportSpec
}

// New sets up a new empty Configurator. The returned Configurator does not
// know about any transports. Individual TransportSpecs must be registered
// against it using the RegisterTransport function.
func New() *Configurator {
	return &Configurator{knownTransports: make(map[string]*transportSpec)}
}

// TransportSpec specifies the configuration parameters for a transport. These
// specifications are registered against a Configurator to teach it how to
// parse the configuration for that transport and build instances of it.
type TransportSpec struct {
	// Name of the transport
	Name string

	// A function in the shape,
	//
	// 	func(C) (transport.Transport, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters accepted by this transport.
	BuildTransport interface{}

	// TODO(abg): Document how these values are actually used since it may be
	// non-obvious.

	// TODO(abg): Make error returns optional -- if the function doesn't
	// return an error value, we can just wrap it to always return nil there.

	// Everything below is optional

	// A function in the shape,
	//
	// 	func(C, transport.Transport) (transport.Inbound, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters for the inbound.
	//
	// This may be nil if this transport does not support inbounds.
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
	BuildUnaryOutbound  interface{}
	BuildOnewayOutbound interface{}

	// The following two are maps from preset name to functions in the shapes,
	//
	// 	func(C, transport.Transport) (transport.UnaryOutbound, error)
	// 	func(C, transport.Transport) (transport.OnewayOutbound, error)
	//
	// Where C is a struct or pointer to a struct defining the configuration
	// parameters for that outbound preset.
	//
	// Either value may be nil to indicate that the transport does not have
	// presets for that RPC type.
	UnaryOutboundPresets  map[string]interface{}
	OnewayOutboundPresets map[string]interface{}

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

// RegisterTransport registers a TransportSpec with the given Configurator.
func (c *Configurator) RegisterTransport(t TransportSpec) error {
	spec, err := newTransportSpec(&t)
	if err != nil {
		return err
	}

	// TODO include more information in error
	// TODO: Panic if a transport with the given name is already registered?
	c.knownTransports[t.Name] = spec
	return nil
}

// LoadYAML loads a YARPC configuration from YAML.
func (c *Configurator) LoadYAML(r io.Reader) (*Builder, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return c.Load(data)
}

// Load a YARPC configuration from the given data map.
func (c *Configurator) Load(data map[string]interface{}) (*Builder, error) {
	var cfg yarpcConfig
	if err := decode.Decode(&cfg, data); err != nil {
		return nil, err
	}

	// Set of transports we actually need
	needTransports := make(map[string]struct{})
	result := Builder{Name: cfg.Name}

	for _, inbound := range cfg.Inbounds {
		if inbound.Disabled {
			continue
		}

		spec, err := c.spec(inbound.Type)
		if err != nil {
			return nil, err
		}

		builder, err := spec.InboundBuilder(inbound.Attributes)
		if err != nil {
			return nil, err
		}

		needTransports[inbound.Type] = struct{}{}
		result.Inbounds = append(result.Inbounds, InboundConfig{TransportName: inbound.Type, Builder: builder})
	}

	for name, clientConfig := range cfg.Outbounds {
		ocfg := OutboundConfig{
			Name:    name,
			Service: clientConfig.Service,
		}

		if clientConfig.Implicit == nil {
			if clientConfig.Unary != nil {
				cfg := clientConfig.Unary
				spec, err := c.spec(cfg.Type)
				if err != nil {
					return nil, err
				}

				builder, err := spec.UnaryOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}

				needTransports[cfg.Type] = struct{}{}
				ocfg.Unary = &UnaryOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}

			if clientConfig.Oneway != nil {
				cfg := clientConfig.Oneway
				spec, err := c.spec(cfg.Type)
				if err != nil {
					return nil, err
				}

				builder, err := spec.OnewayOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}

				needTransports[cfg.Type] = struct{}{}
				ocfg.Oneway = &OnewayOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}
		} else {
			cfg := clientConfig.Implicit
			spec, err := c.spec(cfg.Type)
			if err != nil {
				return nil, err
			}

			if spec.SupportsUnaryOutbound() {
				builder, err := spec.UnaryOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}
				needTransports[cfg.Type] = struct{}{}
				ocfg.Unary = &UnaryOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}

			if spec.SupportsOnewayOutbound() {
				builder, err := spec.OnewayOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}
				needTransports[cfg.Type] = struct{}{}
				ocfg.Oneway = &OnewayOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}
		}
		result.Outbounds = append(result.Outbounds, ocfg)
	}

	// Transports with explicit configuration.
	for name, attrs := range cfg.Transports {
		// Skip because we don't actually need this.
		if _, need := needTransports[name]; !need {
			continue
		}
		delete(needTransports, name)

		spec, err := c.spec(name)
		if err != nil {
			return nil, err
		}

		builder, err := spec.TransportBuilder(attrs)
		if err != nil {
			return nil, fmt.Errorf("failed to decode configuration for transport %q: %v", name, err)
		}

		result.Transports = append(result.Transports, TransportConfig{
			Name:    name,
			Builder: builder,
		})
	}

	// All remaining transports
	for name := range needTransports {
		spec, err := c.spec(name)
		if err != nil {
			// TODO maybe mention which inbounds/outbounds needed this
			return nil, err
		}

		builder, err := spec.TransportBuilder(attributeMap{})
		if err != nil {
			return nil, fmt.Errorf("failed to decode configuration for transport %q: %v", name, err)
		}

		result.Transports = append(result.Transports, TransportConfig{
			Name:    name,
			Builder: builder,
		})
	}

	return &result, nil
}

// Helper to return the spec for the transport with the given name
func (c *Configurator) spec(name string) (*transportSpec, error) {
	spec, ok := c.knownTransports[name]
	if !ok {
		return nil, fmt.Errorf("unknown transport %q", name)
	}
	return spec, nil
}

var (
	_typeOfError          = reflect.TypeOf((*error)(nil)).Elem()
	_typeOfTransport      = reflect.TypeOf((*transport.Transport)(nil)).Elem()
	_typeOfInbound        = reflect.TypeOf((*transport.Inbound)(nil)).Elem()
	_typeOfUnaryOutbound  = reflect.TypeOf((*transport.UnaryOutbound)(nil)).Elem()
	_typeOfOnewayOutbound = reflect.TypeOf((*transport.OnewayOutbound)(nil)).Elem()
)

// Builds transport.Transport objects given a function,
//
// 	func(cfg MyConfigType) (transport.Transport, error)
//
// Where MyConfigType defines configuration parameters of the transport that
// are parseable from YAML or other markup formats.
//
// This must be used in two stages: Load() and then Build(). Having a separate
// Load() step ensures that everything has parsed successfully before we
// start building it, plus it gives us a view of exactly what configuration
// we're using to build everything.
type transportBuilder struct {
	cfgType reflect.Type
	cfg     reflect.Value
	build   reflect.Value // == func(cfg) (transport.Transport, error)
}

// Build a new transportBuilder from the given build function.
func newTransportBuilder(build interface{}) (*transportBuilder, error) {
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
		err = fmt.Errorf("must return a error as its second result, found %v", t.Out(1))
	}

	if err != nil {
		err = fmt.Errorf("invalid BuildTransport %v: %v", t, err)
	}
	return &transportBuilder{cfgType: t.In(0), build: v}, err
}

func (b transportBuilder) Load(attrs attributeMap) (TransportBuilder, error) {
	cfg := reflect.New(b.cfgType)
	if err := attrs.Decode(cfg.Interface()); err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", b.cfgType, err)
	}

	// Note: the receiver is not on the pointer so that these objects are
	// re-usable.
	b.cfg = cfg.Elem()
	return &b, nil
}

func (b *transportBuilder) BuildTransport() (transport.Transport, error) {
	result := b.build.Call([]reflect.Value{b.cfg})
	if err, _ := result[1].Interface().(error); err != nil {
		return nil, err
	}
	return result[0].Interface().(transport.Transport), nil
}

// Builder for types that depend on a transport.Transport as their first
// argument.
//
// This logic is shared between inbounds and different outbound types. It's
// very similar to transportBuilder except they all depend on a
// transport.Transport as their first argument.
//
// This implements functions to be a valid InboundBilder,
// UnaryOutboundBuilder, and OnewayOutboundBuilder but it's up to the logic in
// this package to ensure that this is not used incorrectly.
type builder struct {
	cfg        reflect.Value
	cfgType    reflect.Type
	resultType reflect.Type
	buildFunc  reflect.Value
	fieldNames map[string]struct{}
}

func newBuilder(build interface{}, outputType reflect.Type) (*builder, error) {
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
	case t.In(1) != _typeOfTransport:
		// TODO: We can make this smarter by making transport.Transport
		// optional and either the first or the second argument instead of
		// requiring it as the second argument.
		err = fmt.Errorf("must accept a transport.Transport as its second argument, found %v", t.In(1))
	case t.NumOut() != 2:
		err = fmt.Errorf("must return exactly two results, found %v", t.NumOut())
	case t.Out(0) != outputType:
		err = fmt.Errorf("must return a %v as its first result, found %v", outputType, t.Out(0))
	case t.Out(1) != _typeOfError:
		err = fmt.Errorf("must return a error as its second result, found %v", t.Out(1))
	}

	return &builder{
		cfgType:    t.In(0),
		fieldNames: fieldNames(t.In(0)),
		resultType: outputType,
		buildFunc:  v,
	}, err
}

func (b builder) Load(attrs attributeMap) (*builder, error) {
	cfg := reflect.New(b.cfgType)
	if err := attrs.Decode(cfg.Interface()); err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", b.cfgType, err)
	}

	// Note: the receiver is not on the pointer so that these objects are
	// re-usable.
	b.cfg = cfg.Elem()
	return &b, nil
}

func (b *builder) build(t transport.Transport) (reflect.Value, error) {
	out := b.buildFunc.Call([]reflect.Value{b.cfg, reflect.ValueOf(t)})
	if err, _ := out[1].Interface().(error); err != nil {
		return reflect.Zero(b.resultType), err
	}
	return out[0], nil
}

func (b *builder) BuildInbound(t transport.Transport) (transport.Inbound, error) {
	value, err := b.build(t)
	if err != nil {
		return nil, err
	}
	return value.Interface().(transport.Inbound), nil
}

func (b *builder) BuildUnaryOutbound(t transport.Transport) (transport.UnaryOutbound, error) {
	value, err := b.build(t)
	if err != nil {
		return nil, err
	}
	return value.Interface().(transport.UnaryOutbound), nil
}

func (b *builder) BuildOnewayOutbound(t transport.Transport) (transport.OnewayOutbound, error) {
	value, err := b.build(t)
	if err != nil {
		return nil, err
	}
	return value.Interface().(transport.OnewayOutbound), nil
}

func newInboundBuilder(build interface{}) (*builder, error) {
	b, err := newBuilder(build, _typeOfInbound)
	if err != nil {
		return nil, err
	}

	if _, hasType := b.fieldNames["Type"]; hasType {
		return nil, errors.New("inbound configurations must not have a Type field")
	}

	if _, hasDisabled := b.fieldNames["Disabled"]; hasDisabled {
		return nil, errors.New("inbound configurations must not have a Disabled field")
	}

	return b, nil
}

func newOutboundBuilder(build interface{}, resultType reflect.Type) (*builder, error) {
	b, err := newBuilder(build, resultType)
	if err != nil {
		return nil, err
	}

	if _, hasWith := b.fieldNames["With"]; hasWith {
		return nil, errors.New("outbound configurations must not have a With field")
	}

	return b, nil
}

// Internal representation of TransportSpec.
type transportSpec struct {
	Name string

	transportBuilder      *transportBuilder
	inboundBuilder        *builder
	unaryOutboundBuilder  *builder
	onewayOutboundBuilder *builder
	unaryOutboundPresets  map[string]*builder
	onewayOutboundPresets map[string]*builder
}

func newTransportSpec(spec *TransportSpec) (_ *transportSpec, err error) {
	out := transportSpec{Name: spec.Name}

	if spec.BuildTransport == nil {
		return nil, errors.New("BuildTransport is required")
	}

	out.transportBuilder, err = newTransportBuilder(spec.BuildTransport)
	if err != nil {
		return nil, err
	}

	if spec.BuildInbound != nil {
		out.inboundBuilder, err = newInboundBuilder(spec.BuildInbound)
		if err != nil {
			return nil, err
		}
	}

	if spec.BuildUnaryOutbound != nil {
		out.unaryOutboundBuilder, err = newOutboundBuilder(spec.BuildUnaryOutbound, _typeOfUnaryOutbound)
		if err != nil {
			return nil, err
		}
	}

	if spec.BuildOnewayOutbound != nil {
		out.onewayOutboundBuilder, err = newOutboundBuilder(spec.BuildOnewayOutbound, _typeOfOnewayOutbound)
		if err != nil {
			return nil, err
		}
	}

	out.unaryOutboundPresets, err = buildOutboundPresets(spec.UnaryOutboundPresets, _typeOfUnaryOutbound)
	if err != nil {
		return nil, err
	}

	out.onewayOutboundPresets, err = buildOutboundPresets(spec.OnewayOutboundPresets, _typeOfOnewayOutbound)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (s *transportSpec) TransportBuilder(attrs attributeMap) (TransportBuilder, error) {
	return s.transportBuilder.Load(attrs)
}

func (s *transportSpec) InboundBuilder(attrs attributeMap) (InboundBuilder, error) {
	if s.inboundBuilder == nil {
		return nil, fmt.Errorf("transport %q does not define an inbound", s.Name)
	}
	return s.inboundBuilder.Load(attrs)
}

func (s *transportSpec) UnaryOutboundBuilder(preset string, attrs attributeMap) (UnaryOutboundBuilder, error) {
	if s.unaryOutboundBuilder == nil {
		return nil, fmt.Errorf("transport %q does not define a unary outbound", s.Name)
	}

	b := s.unaryOutboundBuilder
	if preset != "" {
		var ok bool
		b, ok = s.unaryOutboundPresets[preset]
		if !ok {
			return nil, fmt.Errorf("unknown preset %q for unary outbound %q", preset, s.Name)
		}
	}

	b, err := b.Load(attrs)
	return b, err
}

func (s *transportSpec) OnewayOutboundBuilder(preset string, attrs attributeMap) (OnewayOutboundBuilder, error) {
	if s.onewayOutboundBuilder == nil {
		return nil, fmt.Errorf("transport %q does not define a oneway outbound", s.Name)
	}

	b := s.onewayOutboundBuilder
	if preset != "" {
		var ok bool
		b, ok = s.onewayOutboundPresets[preset]
		if !ok {
			return nil, fmt.Errorf("unknown preset %q for oneway outbound %q", preset, s.Name)
		}
	}

	b, err := b.Load(attrs)
	return b, err
}

func (s *transportSpec) SupportsUnaryOutbound() bool {
	return s.unaryOutboundBuilder != nil
}

func (s *transportSpec) SupportsOnewayOutbound() bool {
	return s.onewayOutboundBuilder != nil
}

func buildOutboundPresets(m map[string]interface{}, resultType reflect.Type) (map[string]*builder, error) {
	if len(m) == 0 {
		return nil, nil
	}
	out := make(map[string]*builder, len(m))
	for k, v := range m {
		var err error
		out[k], err = newOutboundBuilder(v, resultType)
		if err != nil {
			return nil, fmt.Errorf("invalid preset %q: %v", k, err)
		}
	}
	return out, nil
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
		fields[t.Field(i).Name] = struct{}{}
	}
	return fields
}

func isDecodable(t reflect.Type) bool {
	for ; t.Kind() == reflect.Ptr; t = t.Elem() {
	}

	// TODO(abg): Do we want to support top-level map types for configuration

	if t.Kind() == reflect.Struct {
		return true
	}
	return false
}
