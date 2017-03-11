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
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
)

type configuredClient struct {
	Service string
	Unary   *configuredOutbound
	Oneway  *configuredOutbound
}

type configuredInbound struct {
	Transport string
	Value     *configuredValue
}

type configuredOutbound struct {
	Transport string
	Value     *configuredValue
}

type builder struct {
	Name string

	// Transports that we actually need and their specs. We need a transport
	// only if we have at least one inbound or outbound using it.
	needTransports map[string]*compiledTransportSpec

	transports map[string]*configuredValue
	inbounds   []configuredInbound
	clients    map[string]*configuredClient
}

func newBuilder(name string) *builder {
	return &builder{
		Name:           name,
		needTransports: make(map[string]*compiledTransportSpec),
		transports:     make(map[string]*configuredValue),
		clients:        make(map[string]*configuredClient),
	}
}

func (b *builder) Build() (yarpc.Config, error) {
	transports := make(map[string]transport.Transport)

	for name, spec := range b.needTransports {
		cv, ok := b.transports[name]

		var err error
		if !ok {
			// No configuration provided for the transport. Use an empty map.
			cv, err = spec.Transport.Decode(attributeMap{})
			if err != nil {
				return yarpc.Config{}, err
			}
		}

		transports[name], err = buildTransport(cv)
		if err != nil {
			return yarpc.Config{}, err
		}
	}

	cfg := yarpc.Config{Name: b.Name}
	for _, i := range b.inbounds {
		ib, err := buildInbound(i.Value, transports[i.Transport])
		if err != nil {
			return yarpc.Config{}, err
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

		if o := c.Unary; o != nil {
			ob.Unary, err = buildUnaryOutbound(o.Value, transports[o.Transport])
			if err != nil {
				return yarpc.Config{}, err
			}
		}
		if o := c.Oneway; o != nil {
			ob.Oneway, err = buildOnewayOutbound(o.Value, transports[o.Transport])
			if err != nil {
				return yarpc.Config{}, err
			}
		}

		outbounds[ccname] = ob
	}
	if len(outbounds) > 0 {
		cfg.Outbounds = outbounds
	}

	return cfg, nil
}

// buildTransport builds a Transport from the given value. This will panic if
// the output type is not a Transport.
func buildTransport(cv *configuredValue) (transport.Transport, error) {
	result, err := cv.Build()
	if err != nil {
		return nil, err
	}
	return result.(transport.Transport), nil
}

// buildInbound builds an Inbound from the given value. This will panic if the
// output type for this is not transport.Inbound.
func buildInbound(cv *configuredValue, t transport.Transport) (transport.Inbound, error) {
	result, err := cv.Build(t)
	if err != nil {
		return nil, err
	}
	return result.(transport.Inbound), nil
}

// buildUnaryOutbound builds an UnaryOutbound from the given value. This will panic
// if the output type for this is not transport.UnaryOutbound.
func buildUnaryOutbound(cv *configuredValue, t transport.Transport) (transport.UnaryOutbound, error) {
	result, err := cv.Build(t)
	if err != nil {
		return nil, err
	}
	return result.(transport.UnaryOutbound), nil
}

// buildOnewayOutbound builds an OnewayOutbound from the given value. This will
// panic if the output type for this is not transport.OnewayOutbound.
func buildOnewayOutbound(cv *configuredValue, t transport.Transport) (transport.OnewayOutbound, error) {
	result, err := cv.Build(t)
	if err != nil {
		return nil, err
	}
	return result.(transport.OnewayOutbound), nil
}

func (b *builder) AddTransportConfig(spec *compiledTransportSpec, attrs attributeMap) error {
	cv, err := spec.Transport.Decode(attrs)
	if err != nil {
		return fmt.Errorf("failed to decode transport configuration: %v", err)
	}

	b.transports[spec.Name] = cv
	return nil
}

func (b *builder) AddInboundConfig(spec *compiledTransportSpec, attrs attributeMap) error {
	if spec.Inbound == nil {
		return fmt.Errorf("transport %q does not support inbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.Inbound.Decode(attrs)
	if err != nil {
		return fmt.Errorf("failed to decode inbound configuration: %v", err)
	}

	b.inbounds = append(b.inbounds, configuredInbound{
		Transport: spec.Name,
		Value:     cv,
	})
	return nil
}

func (b *builder) AddImplicitOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs attributeMap,
) error {
	var errs []error
	supportsOutbound := false

	if spec.SupportsUnaryOutbound() {
		supportsOutbound = true
		if err := b.AddUnaryOutbound(spec, outboundKey, service, attrs); err != nil {
			errs = append(errs, err)
		}
	}

	if spec.SupportsOnewayOutbound() {
		supportsOutbound = true
		if err := b.AddOnewayOutbound(spec, outboundKey, service, attrs); err != nil {
			errs = append(errs, err)
		}
	}

	if !supportsOutbound {
		return fmt.Errorf("transport %q does not support outbound requests", spec.Name)
	}

	return errors.MultiError(errs)
}

func (b *builder) AddUnaryOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs attributeMap,
) error {
	if spec.UnaryOutbound == nil {
		return fmt.Errorf("transport %q does not support unary outbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.UnaryOutbound.Decode(attrs)
	if err != nil {
		return fmt.Errorf("failed to decode unary outbound configuration: %v", err)
	}

	cc, ok := b.clients[outboundKey]
	if !ok {
		cc = &configuredClient{Service: service}
		b.clients[outboundKey] = cc
	}

	cc.Unary = &configuredOutbound{Transport: spec.Name, Value: cv}
	return nil
}

func (b *builder) AddOnewayOutbound(
	spec *compiledTransportSpec, outboundKey, service string, attrs attributeMap,
) error {
	if spec.OnewayOutbound == nil {
		return fmt.Errorf("transport %q does not support oneway outbound requests", spec.Name)
	}

	b.needTransport(spec)
	cv, err := spec.OnewayOutbound.Decode(attrs)
	if err != nil {
		return fmt.Errorf("failed to decode oneway outbound configuration: %v", err)
	}

	cc, ok := b.clients[outboundKey]
	if !ok {
		cc = &configuredClient{Service: service}
		b.clients[outboundKey] = cc
	}

	cc.Oneway = &configuredOutbound{Transport: spec.Name, Value: cv}
	return nil
}

func (b *builder) needTransport(spec *compiledTransportSpec) {
	b.needTransports[spec.Name] = spec
}
