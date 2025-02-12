// Copyright (c) 2025 Uber Technologies, Inc.
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
	"fmt"
	"reflect"

	"github.com/uber-go/mapdecode"
	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/internal/config"
)

type buildableOutbounds struct {
	Service string
	Unary   *buildableOutbound
	Oneway  *buildableOutbound
	Stream  *buildableOutbound
}

type buildableInbound struct {
	Transport string
	Value     *buildable
}

type buildableOutbound struct {
	TransportSpec *compiledTransportSpec
	Value         *buildable
}

type builder struct {
	Name string
	kit  *Kit

	// Transports that we actually need and their specs. We need a transport
	// only if we have at least one inbound or outbound using it.
	needTransports map[string]*compiledTransportSpec

	transports map[string]*buildable
	inbounds   []buildableInbound
	clients    map[string]*buildableOutbounds
}

func newBuilder(name string, kit *Kit) *builder {
	return &builder{
		Name:           name,
		kit:            kit,
		needTransports: make(map[string]*compiledTransportSpec),
		transports:     make(map[string]*buildable),
		clients:        make(map[string]*buildableOutbounds),
	}
}

func (b *builder) Build() (yarpc.Config, error) {
	var (
		transports = make(map[string]transport.Transport)
		cfg        = yarpc.Config{Name: b.Name}
		errs       error
	)

	for name, spec := range b.needTransports {
		cv, ok := b.transports[name]

		var err error
		if !ok {
			// No configuration provided for the transport. Use an empty map.
			cv, err = spec.Transport.Decode(config.AttributeMap{}, config.InterpolateWith(b.kit.resolver))
			if err != nil {
				return yarpc.Config{}, err
			}
		}

		transports[name], err = buildTransport(cv, b.kit)
		if err != nil {
			return yarpc.Config{}, err
		}
	}

	for _, i := range b.inbounds {
		ib, err := buildInbound(i.Value, transports[i.Transport], b.kit)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		cfg.Inbounds = append(cfg.Inbounds, ib)
	}

	outbounds := make(yarpc.Outbounds, len(b.clients))
	for ccname, c := range b.clients {
		var err error

		var ob transport.Outbounds
		if c.Service != ccname {
			ob.ServiceName = c.Service
		}

		kit := b.kit.withOutboundName(c.Service)
		if o := c.Unary; o != nil {
			ob.Unary, err = buildUnaryOutbound(o, transports[o.TransportSpec.Name], kit)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf(`failed to configure unary outbound for %q: %v`, ccname, err))
				continue
			}
		}
		if o := c.Oneway; o != nil {
			ob.Oneway, err = buildOnewayOutbound(o, transports[o.TransportSpec.Name], kit)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf(`failed to configure oneway outbound for %q: %v`, ccname, err))
				continue
			}
		}
		if o := c.Stream; o != nil {
			ob.Stream, err = buildStreamOutbound(o, transports[o.TransportSpec.Name], kit)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf(`failed to configure stream outbound for %q: %v`, ccname, err))
				continue
			}
		}

		outbounds[ccname] = ob
	}
	if len(outbounds) > 0 {
		cfg.Outbounds = outbounds
	}

	return cfg, errs
}

// buildTransport builds a Transport from the given value. This will panic if
// the output type is not a Transport.
func buildTransport(cv *buildable, k *Kit) (transport.Transport, error) {
	result, err := cv.Build(k)
	if err != nil {
		return nil, err
	}
	return result.(transport.Transport), nil
}

// buildInbound builds an Inbound from the given value. This will panic if the
// output type for this is not transport.Inbound.
func buildInbound(cv *buildable, t transport.Transport, k *Kit) (transport.Inbound, error) {
	result, err := cv.Build(t, k)
	if err != nil {
		return nil, err
	}
	return result.(transport.Inbound), nil
}

// buildUnaryOutbound builds an UnaryOutbound from the given value. This will panic
// if the output type for this is not transport.UnaryOutbound.
func buildUnaryOutbound(o *buildableOutbound, t transport.Transport, k *Kit) (transport.UnaryOutbound, error) {
	result, err := o.Value.Build(t, k.withTransportSpec(o.TransportSpec))
	if err != nil {
		return nil, err
	}
	return result.(transport.UnaryOutbound), nil
}

// buildOnewayOutbound builds an OnewayOutbound from the given value. This will
// panic if the output type for this is not transport.OnewayOutbound.
func buildOnewayOutbound(o *buildableOutbound, t transport.Transport, k *Kit) (transport.OnewayOutbound, error) {
	result, err := o.Value.Build(t, k.withTransportSpec(o.TransportSpec))
	if err != nil {
		return nil, err
	}
	return result.(transport.OnewayOutbound), nil
}

// buildStreamOutbound builds an StreamOutbound from the given value. This will
// panic if the output type for this is not transport.StreamOutbound.
func buildStreamOutbound(o *buildableOutbound, t transport.Transport, k *Kit) (transport.StreamOutbound, error) {
	result, err := o.Value.Build(t, k.withTransportSpec(o.TransportSpec))
	if err != nil {
		return nil, err
	}
	return result.(transport.StreamOutbound), nil
}

func (b *builder) AddTransportConfig(spec *compiledTransportSpec, attrs config.AttributeMap) error {
	cv, err := spec.Transport.Decode(attrs, config.InterpolateWith(b.kit.resolver))
	if err != nil {
		return fmt.Errorf("failed to decode transport configuration: %v", err)
	}

	b.transports[spec.Name] = cv
	return nil
}

func (b *builder) AddInboundConfig(spec *compiledTransportSpec, attrs config.AttributeMap) error {
	if spec.Inbound == nil {
		return fmt.Errorf("transport %q does not support inbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.Inbound.Decode(attrs, config.InterpolateWith(b.kit.resolver), mapdecode.DecodeHook(tlsModeDecodeHook))
	if err != nil {
		return fmt.Errorf("failed to decode inbound configuration: %v", err)
	}

	b.inbounds = append(b.inbounds, buildableInbound{
		Transport: spec.Name,
		Value:     cv,
	})
	return nil
}

func (b *builder) AddImplicitOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs config.AttributeMap,
) error {
	var errs error
	supportsOutbound := false

	if spec.SupportsUnaryOutbound() {
		supportsOutbound = true
		if err := b.AddUnaryOutbound(spec, outboundKey, service, attrs); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if spec.SupportsOnewayOutbound() {
		supportsOutbound = true
		if err := b.AddOnewayOutbound(spec, outboundKey, service, attrs); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	if spec.SupportsStreamOutbound() {
		supportsOutbound = true
		if err := b.AddStreamOutbound(spec, outboundKey, service, attrs); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if !supportsOutbound {
		return fmt.Errorf("transport %q does not support outbound requests", spec.Name)
	}

	return errs
}

func (b *builder) AddUnaryOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs config.AttributeMap,
) error {
	if spec.UnaryOutbound == nil {
		return fmt.Errorf("transport %q does not support unary outbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.UnaryOutbound.Decode(attrs, config.InterpolateWith(b.kit.resolver), mapdecode.DecodeHook(tlsModeDecodeHook))
	if err != nil {
		return fmt.Errorf("failed to decode unary outbound configuration: %v", err)
	}

	cc, ok := b.clients[outboundKey]
	if !ok {
		cc = &buildableOutbounds{Service: service}
		b.clients[outboundKey] = cc
	}

	cc.Unary = &buildableOutbound{TransportSpec: spec, Value: cv}
	return nil
}

func (b *builder) AddOnewayOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs config.AttributeMap,
) error {
	if spec.OnewayOutbound == nil {
		return fmt.Errorf("transport %q does not support oneway outbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.OnewayOutbound.Decode(attrs, config.InterpolateWith(b.kit.resolver), mapdecode.DecodeHook(tlsModeDecodeHook))
	if err != nil {
		return fmt.Errorf("failed to decode oneway outbound configuration: %v", err)
	}

	cc, ok := b.clients[outboundKey]
	if !ok {
		cc = &buildableOutbounds{Service: service}
		b.clients[outboundKey] = cc
	}

	cc.Oneway = &buildableOutbound{TransportSpec: spec, Value: cv}
	return nil
}

func (b *builder) AddStreamOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs config.AttributeMap,
) error {
	if spec.StreamOutbound == nil {
		return fmt.Errorf("transport %q does not support stream outbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.StreamOutbound.Decode(attrs, config.InterpolateWith(b.kit.resolver), mapdecode.DecodeHook(tlsModeDecodeHook))
	if err != nil {
		return fmt.Errorf("failed to decode stream outbound configuration: %v", err)
	}

	cc, ok := b.clients[outboundKey]
	if !ok {
		cc = &buildableOutbounds{Service: service}
		b.clients[outboundKey] = cc
	}

	cc.Stream = &buildableOutbound{TransportSpec: spec, Value: cv}
	return nil
}

func (b *builder) needTransport(spec *compiledTransportSpec) {
	b.needTransports[spec.Name] = spec
}

func tlsModeDecodeHook(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	var mode yarpctls.Mode
	if from.Kind() != reflect.String || to != reflect.TypeOf(mode) {
		return data, nil
	}

	err := mode.UnmarshalText([]byte(data.String()))
	return reflect.ValueOf(mode), err
}
