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
	"os"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/interpolate"

	"go.uber.org/multierr"
	"gopkg.in/yaml.v2"
)

// Configurator helps build Dispatchers using runtime configuration.
//
// An empty Configurator does not know about any transports. Inform it about
// the different transports and their configuration parameters using the
// RegisterTransport function.
type Configurator struct {
	knownTransports       map[string]*compiledTransportSpec
	knownPeerLists        map[string]*compiledPeerListSpec
	knownPeerListUpdaters map[string]*compiledPeerListUpdaterSpec
	resolver              interpolate.VariableResolver
}

// New sets up a new empty Configurator. The returned Configurator does not
// know about any Transports, peer lists, or peer list updaters.
// Individual TransportSpecs, PeerListSpecs, and PeerListUpdaterSpecs must be registered
// against it using the RegisterTransport, RegisterPeerList, and RegisterPeerListUpdater
// functions.
func New(opts ...Option) *Configurator {
	c := &Configurator{
		knownTransports:       make(map[string]*compiledTransportSpec),
		knownPeerLists:        make(map[string]*compiledPeerListSpec),
		knownPeerListUpdaters: make(map[string]*compiledPeerListUpdaterSpec),
		resolver:              os.LookupEnv,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// RegisterTransport registers a TransportSpec with the given Configurator. An
// error is returned if the TransportSpec was invalid.
//
// If a transport with the same name was already registered, it will be
// overwritten.
//
// Use MustRegisterTransport if you want to panic in case of registration
// failure.
func (c *Configurator) RegisterTransport(t TransportSpec) error {
	if t.Name == "" {
		return errors.New("name is required")
	}

	spec, err := compileTransportSpec(&t)
	if err != nil {
		return fmt.Errorf("invalid TransportSpec for %q: %v", t.Name, err)
	}

	// TODO: Panic if a transport with the given name is already registered?
	c.knownTransports[t.Name] = spec
	return nil
}

// MustRegisterTransport registers the given TransportSpec with the
// Configurator. This function panics if the TransportSpec was invalid.
func (c *Configurator) MustRegisterTransport(t TransportSpec) {
	if err := c.RegisterTransport(t); err != nil {
		panic(err)
	}
}

// RegisterPeerList registers a PeerListSpec with the given Configurator. Returns
// an error if the PeerListSpec is invalid.
//
// If a chooser with the same name already exists, it will be replaced.
//
// Use MustRegisterPeerList to panic in the case of registration failure.
func (c *Configurator) RegisterPeerList(s PeerListSpec) error {
	if s.Name == "" {
		return errors.New("name is required")
	}

	spec, err := compilePeerListSpec(&s)
	if err != nil {
		return fmt.Errorf("invalid PeerListSpec for %q: %v", s.Name, err)
	}

	c.knownPeerLists[s.Name] = spec
	return nil
}

// MustRegisterPeerList registers the given PeerListSpec with the Configurator.
// This function panics if the PeerListSpec is invalid.
func (c *Configurator) MustRegisterPeerList(s PeerListSpec) {
	if err := c.RegisterPeerList(s); err != nil {
		panic(err)
	}
}

// RegisterPeerListUpdater registers a PeerListUpdaterSpec with the given
// Configurator.
// Returns an error if the PeerListUpdaterSpec is invalid.
//
// A binder enables custom peer list bindings, like DNS with SRV + A records or
// a task list file watcher.
//
// If a binder with the same name already exists, it will be replaced.
//
// Use MustRegisterPeerListUpdater to panic if the registration fails.
func (c *Configurator) RegisterPeerListUpdater(s PeerListUpdaterSpec) error {
	if s.Name == "" {
		return errors.New("name is required")
	}

	spec, err := compilePeerListUpdaterSpec(&s)
	if err != nil {
		return fmt.Errorf("invalid PeerListUpdaterSpec for %q: %v", s.Name, err)
	}

	c.knownPeerListUpdaters[s.Name] = spec
	return nil
}

// MustRegisterPeerListUpdater registers the given PeerListUpdaterSpec with the
// Configurator.
// This function panics if the PeerListUpdaterSpec is invalid.
func (c *Configurator) MustRegisterPeerListUpdater(s PeerListUpdaterSpec) {
	if err := c.RegisterPeerListUpdater(s); err != nil {
		panic(err)
	}
}

// LoadConfigFromYAML loads a yarpc.Config from YAML. Use LoadConfig if you
// have your own map[string]interface{} or map[interface{}]interface{} to
// provide.
func (c *Configurator) LoadConfigFromYAML(serviceName string, r io.Reader) (yarpc.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return yarpc.Config{}, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return yarpc.Config{}, err
	}
	return c.LoadConfig(serviceName, data)
}

// LoadConfig loads a yarpc.Config from a map[string]interface{} or
// map[interface{}]interface{}.
//
// See the module documentation for the shape the map[string]interface{} is
// expected to conform to.
func (c *Configurator) LoadConfig(serviceName string, data interface{}) (yarpc.Config, error) {
	var cfg yarpcConfig
	if err := decodeInto(&cfg, data); err != nil {
		return yarpc.Config{}, err
	}
	return c.load(serviceName, &cfg)
}

// NewDispatcherFromYAML builds a Dispatcher from the given YAML
// configuration.
func (c *Configurator) NewDispatcherFromYAML(serviceName string, r io.Reader) (*yarpc.Dispatcher, error) {
	cfg, err := c.LoadConfigFromYAML(serviceName, r)
	if err != nil {
		return nil, err
	}
	return yarpc.NewDispatcher(cfg), nil
}

// NewDispatcher builds a new Dispatcher from the given configuration data.
func (c *Configurator) NewDispatcher(serviceName string, data interface{}) (*yarpc.Dispatcher, error) {
	cfg, err := c.LoadConfig(serviceName, data)
	if err != nil {
		return nil, err
	}
	return yarpc.NewDispatcher(cfg), nil
}

func (c *Configurator) load(serviceName string, cfg *yarpcConfig) (_ yarpc.Config, err error) {
	b := newBuilder(serviceName, &Kit{name: serviceName, c: c}, c.resolver)

	for _, inbound := range cfg.Inbounds {
		if e := c.loadInboundInto(b, inbound); e != nil {
			err = multierr.Append(err, e)
		}
	}

	for name, outboundConfig := range cfg.Outbounds {
		if e := c.loadOutboundInto(b, name, outboundConfig); e != nil {
			err = multierr.Append(err, e)
		}
	}

	for name, attrs := range cfg.Transports {
		if e := c.loadTransportInto(b, name, attrs); e != nil {
			err = multierr.Append(err, e)
		}
	}

	if err != nil {
		return yarpc.Config{}, err
	}

	return b.Build()
}

func (c *Configurator) loadInboundInto(b *builder, i inbound) error {
	if i.Disabled {
		return nil
	}

	spec, err := c.spec(i.Type)
	if err != nil {
		// TODO: Maybe we should keep track of the inbound name so that if
		// it differs from the transport name, we can mention that in the
		// error message.
		return fmt.Errorf("failed to load inbound: %v", err)
	}

	return b.AddInboundConfig(spec, i.Attributes)
}

func (c *Configurator) loadOutboundInto(b *builder, name string, cfg outbounds) error {
	// This matches the signature of builder.AddImplicitOutbound,
	// AddUnaryOutbound and AddOnewayOutbound
	type adder func(*compiledTransportSpec, string, string, attributeMap) error

	loadUsing := func(o *outbound, adder adder) error {
		spec, err := c.spec(o.Type)
		if err != nil {
			return fmt.Errorf("failed to load configuration for outbound %q: %v", name, err)
		}

		if err := adder(spec, name, cfg.Service, o.Attributes); err != nil {
			return fmt.Errorf("failed to add outbound %q: %v", name, err)
		}

		return nil
	}

	if implicit := cfg.Implicit; implicit != nil {
		return loadUsing(implicit, b.AddImplicitOutbound)
	}

	if unary := cfg.Unary; unary != nil {
		if err := loadUsing(unary, b.AddUnaryOutbound); err != nil {
			return err
		}
	}

	if oneway := cfg.Oneway; oneway != nil {
		if err := loadUsing(oneway, b.AddOnewayOutbound); err != nil {
			return err
		}
	}

	return nil
}

func (c *Configurator) loadTransportInto(b *builder, name string, attrs attributeMap) error {
	spec, err := c.spec(name)
	if err != nil {
		return fmt.Errorf("failed to load configuration for transport %q: %v", name, err)
	}

	return b.AddTransportConfig(spec, attrs)
}

// Returns the compiled spec for the transport with the given name or an error
func (c *Configurator) spec(name string) (*compiledTransportSpec, error) {
	spec, ok := c.knownTransports[name]
	if !ok {
		return nil, fmt.Errorf("unknown transport %q", name)
	}
	return spec, nil
}
