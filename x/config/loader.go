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

	"go.uber.org/yarpc/internal/decode"

	"gopkg.in/yaml.v2"
)

// Loader TODO
type Loader struct {
	knownTransports map[string]transportSpec
}

// NewLoader TODO
func NewLoader() *Loader {
	return &Loader{knownTransports: make(map[string]transportSpec)}
}

// TransportSpec TODO
type TransportSpec struct {
	// Name of the transport
	Name                string
	TransportConfigType TransportBuilder

	// Everything below is optional

	InboundConfigType        InboundBuilder
	UnaryOutboundConfigType  UnaryOutboundBuilder
	OnewayOutboundConfigType OnewayOutboundBuilder
	UnaryOutboundPresets     map[string]UnaryOutboundBuilder
	OnewayOutboundPresets    map[string]OnewayOutboundBuilder
}

// RegisterTransport TODO
func (l *Loader) RegisterTransport(t TransportSpec) error {
	getStruct := func(t reflect.Type) reflect.Type {
		switch t.Kind() {
		case reflect.Struct:
			return t
		case reflect.Ptr:
			if t.Elem().Kind() == reflect.Struct {
				return t.Elem()
			}
		}
		return nil
	}

	spec := transportSpec{name: t.Name}

	// TODO include more information in error

	if t.TransportConfigType == nil {
		return errors.New("a transport configuration type is required")
	}

	spec.transportConfigType = getStruct(reflect.TypeOf(t.TransportConfigType))
	if spec.transportConfigType == nil {
		return errors.New("transport configurations can only be defined on structs")
	}

	if t.InboundConfigType != nil {
		spec.inboundConfigType = getStruct(reflect.TypeOf(t.InboundConfigType))
		if spec.inboundConfigType == nil {
			return errors.New("inbound configurations can only be defined on structs")
		}

		if _, ok := spec.inboundConfigType.FieldByName("Type"); ok {
			return fmt.Errorf("inbound configurations cannot have a Type field")
		}

		if _, ok := spec.inboundConfigType.FieldByName("Disabled"); ok {
			return fmt.Errorf("inbound configurations cannot have a Disabled field")
		}
	}

	if t.UnaryOutboundConfigType != nil {
		spec.unaryOutboundConfigType = getStruct(reflect.TypeOf(t.UnaryOutboundConfigType))
		if spec.unaryOutboundConfigType == nil {
			return fmt.Errorf("unary outbound configurations can only be defined on structs")
		}

		// We should be checking the config: tags too
		if _, ok := spec.unaryOutboundConfigType.FieldByName("With"); ok {
			return fmt.Errorf("outbound configurations cannot have a With field")
		}
	}

	if t.OnewayOutboundConfigType != nil {
		spec.onewayOutboundConfigType = getStruct(reflect.TypeOf(t.OnewayOutboundConfigType))
		if spec.onewayOutboundConfigType == nil {
			return fmt.Errorf("oneway outbound configurations can only be defined on structs")
		}

		if _, ok := spec.onewayOutboundConfigType.FieldByName("With"); ok {
			return fmt.Errorf("outbound configurations cannot have a With field")
		}
	}

	spec.unaryOutboundPresets = make(map[string]reflect.Type, len(t.UnaryOutboundPresets))
	for name, preset := range t.UnaryOutboundPresets {
		spec.unaryOutboundPresets[name] = getStruct(reflect.TypeOf(preset))
		if spec.unaryOutboundPresets[name] == nil {
			return fmt.Errorf("outbound presets can only be defined on structs")
		}
	}

	spec.onewayOutboundPresets = make(map[string]reflect.Type, len(t.OnewayOutboundPresets))
	for name, preset := range t.OnewayOutboundPresets {
		spec.onewayOutboundPresets[name] = getStruct(reflect.TypeOf(preset))
		if spec.onewayOutboundPresets[name] == nil {
			return fmt.Errorf("outbound presets can only be defined on structs")
		}
	}

	// TODO: Panic if a transport with the given name is already registered?
	l.knownTransports[t.Name] = spec
	return nil
}

// LoadYAML loads a YARPC configuration from YAML.
func (l *Loader) LoadYAML(r io.Reader) (*Builder, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return l.Load(data)
}

// Load a YARPC configuration from the given data map.
func (l *Loader) Load(data map[string]interface{}) (*Builder, error) {
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

		spec, err := l.spec(inbound.Type)
		if err != nil {
			return nil, err
		}

		builder, err := spec.inboundBuilder(inbound.Attributes)
		if err != nil {
			return nil, err
		}

		needTransports[inbound.Type] = struct{}{}
		result.Inbounds = append(result.Inbounds, InboundConfig{
			TransportName: inbound.Type,
			Builder:       builder,
		})
	}

	for name, clientConfig := range cfg.Outbounds {
		ocfg := OutboundConfig{
			Name:    name,
			Service: clientConfig.Service,
		}

		if clientConfig.Implicit == nil {
			if clientConfig.Unary != nil {
				cfg := clientConfig.Unary
				spec, err := l.spec(cfg.Type)
				if err != nil {
					return nil, err
				}

				builder, err := spec.unaryOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}

				needTransports[cfg.Type] = struct{}{}
				ocfg.Unary = &UnaryOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}

			if clientConfig.Oneway != nil {
				cfg := clientConfig.Oneway
				spec, err := l.spec(cfg.Type)
				if err != nil {
					return nil, err
				}

				builder, err := spec.onewayOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}

				needTransports[cfg.Type] = struct{}{}
				ocfg.Oneway = &OnewayOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}
		} else {
			cfg := clientConfig.Implicit
			spec, err := l.spec(cfg.Type)
			if err != nil {
				return nil, err
			}

			if spec.supportsUnaryOutbound() {
				builder, err := spec.unaryOutboundBuilder(cfg.Preset, cfg.Attributes)
				if err != nil {
					return nil, err
				}
				needTransports[cfg.Type] = struct{}{}
				ocfg.Unary = &UnaryOutboundConfig{TransportName: cfg.Type, Builder: builder}
			}

			if spec.supportsOnewayOutbound() {
				builder, err := spec.onewayOutboundBuilder(cfg.Preset, cfg.Attributes)
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

		spec, err := l.spec(name)
		if err != nil {
			return nil, err
		}

		builder, err := spec.transportBuilder(attrs)
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
		spec, err := l.spec(name)
		if err != nil {
			// TODO maybe mention which inbounds/outbounds needed this
			return nil, err
		}

		builder, err := spec.transportBuilder(attributeMap{})
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
func (l Loader) spec(name string) (transportSpec, error) {
	spec, ok := l.knownTransports[name]
	if !ok {
		return transportSpec{}, fmt.Errorf("unknown transport %q", name)
	}
	return spec, nil
}

// Internal representation of TransportSpec
type transportSpec struct {
	name                     string
	transportConfigType      reflect.Type
	inboundConfigType        reflect.Type
	unaryOutboundConfigType  reflect.Type
	onewayOutboundConfigType reflect.Type
	unaryOutboundPresets     map[string]reflect.Type
	onewayOutboundPresets    map[string]reflect.Type
}

func (s *transportSpec) supportsUnaryOutbound() bool {
	return s.unaryOutboundConfigType != nil
}

func (s *transportSpec) supportsOnewayOutbound() bool {
	return s.onewayOutboundConfigType != nil
}

func (s *transportSpec) transportBuilder(attrs attributeMap) (TransportBuilder, error) {
	result := reflect.New(s.transportConfigType).Interface()
	if err := attrs.Decode(result); err != nil {
		return nil, fmt.Errorf("failed to decode configuration for transport %q: %v", s.name, err)
	}
	return result.(TransportBuilder), nil
}

func (s *transportSpec) inboundBuilder(attrs attributeMap) (InboundBuilder, error) {
	if s.inboundConfigType == nil {
		return nil, fmt.Errorf("transport %q does not define an inbound", s.name)
	}

	result := reflect.New(s.inboundConfigType).Interface()
	if err := attrs.Decode(result); err != nil {
		return nil, fmt.Errorf("failed to decode configuration for inbound %q: %v", s.name, err)
	}
	return result.(InboundBuilder), nil
}

func (s *transportSpec) unaryOutboundBuilder(preset string, attrs attributeMap) (UnaryOutboundBuilder, error) {
	typ := s.unaryOutboundConfigType
	if typ == nil {
		return nil, fmt.Errorf("transport %q does not support unary outbounds", s.name)
	}

	if preset != "" {
		var ok bool
		typ, ok = s.unaryOutboundPresets[preset]
		if !ok {
			return nil, fmt.Errorf("unknown preset %q for unary outbound %q", preset, s.name)
		}
	}

	result := reflect.New(typ).Interface()
	if err := attrs.Decode(result); err != nil {
		return nil, fmt.Errorf("failed to decode configuration for unary outbound %q: %v", s.name, err)
	}
	return result.(UnaryOutboundBuilder), nil
}

func (s *transportSpec) onewayOutboundBuilder(preset string, attrs attributeMap) (OnewayOutboundBuilder, error) {
	typ := s.onewayOutboundConfigType
	if typ == nil {
		return nil, fmt.Errorf("transport %q does not support oneway outbounds", s.name)
	}

	if preset != "" {
		var ok bool
		typ, ok = s.onewayOutboundPresets[preset]
		if !ok {
			return nil, fmt.Errorf("unknown preset %q for oneway outbound %q", preset, s.name)
		}
	}

	result := reflect.New(typ).Interface()
	if err := attrs.Decode(result); err != nil {
		return nil, fmt.Errorf("failed to decode configuration for oneway outbound %q: %v", s.name, err)
	}
	return result.(OnewayOutboundBuilder), nil
}
