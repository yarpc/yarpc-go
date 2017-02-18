package config

import (
	"fmt"
	"reflect"

	"go.uber.org/yarpc/internal/decode"

	"gopkg.in/yaml.v2"
)

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

// Loader TODO
type Loader struct {
	knownTransports map[string]transportSpec
}

// NewLoader TODO
func NewLoader() *Loader {
	return &Loader{knownTransports: make(map[string]transportSpec)}
}

func (l Loader) spec(name string) (transportSpec, error) {
	spec, ok := l.knownTransports[name]
	if !ok {
		return transportSpec{}, fmt.Errorf("unknown transport %q", name)
	}
	return spec, nil
}

// LoadYAML loads a YARPC configuration from YAML.
func (l *Loader) LoadYAML(b []byte) (*YARPC, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return l.Load(data)
}

// Load a YARPC configuration from the given data map.
func (l *Loader) Load(data map[string]interface{}) (*YARPC, error) {
	var cfg yarpcConfig
	if err := decode.Decode(&cfg, data); err != nil {
		return nil, err
	}

	// Set of transports we actually need
	needTransports := make(map[string]struct{})
	result := YARPC{Name: cfg.Name}

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
