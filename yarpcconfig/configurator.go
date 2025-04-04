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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/config"
	"go.uber.org/yarpc/internal/interpolate"
	"gopkg.in/yaml.v2"
)

// Configurator helps build Dispatchers using runtime configuration.
//
// A new Configurator does not know about any transports, peer lists, or peer
// list updaters. Inform it about them by using the RegisterTransport,
// RegisterPeerList, and RegisterPeerListUpdater functions, or their Must*
// variants.
type Configurator struct {
	knownTransports       map[string]*compiledTransportSpec
	knownPeerChoosers     map[string]*compiledPeerChooserSpec
	knownPeerLists        map[string]*compiledPeerListSpec
	knownPeerListUpdaters map[string]*compiledPeerListUpdaterSpec
	knownCompressors      map[string]transport.Compressor
	resolver              interpolate.VariableResolver
}

// New sets up a new empty Configurator. The returned Configurator does not
// know about any Transports, peer lists, or peer list updaters.
func New(opts ...Option) *Configurator {
	c := &Configurator{
		knownTransports:       make(map[string]*compiledTransportSpec),
		knownPeerChoosers:     make(map[string]*compiledPeerChooserSpec),
		knownPeerLists:        make(map[string]*compiledPeerListSpec),
		knownPeerListUpdaters: make(map[string]*compiledPeerListUpdaterSpec),
		knownCompressors:      make(map[string]transport.Compressor),
		resolver:              os.LookupEnv,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// RegisterTransport registers a TransportSpec with the Configurator, teaching
// it how to load configuration and build inbounds and outbounds for that
// transport.
//
// An error is returned if the TransportSpec is invalid. Use
// MustRegisterTransport if you want to panic in case of registration failure.
//
// If a transport with the same name already exists, it will be replaced.
//
// See TransportSpec for details on how to integrate your own transport with
// the system.
func (c *Configurator) RegisterTransport(t TransportSpec) error {
	if t.Name == "" {
		return errors.New("name is required")
	}

	spec, err := compileTransportSpec(&t)
	if err != nil {
		return fmt.Errorf("invalid TransportSpec for %q: %v", t.Name, err)
	}

	c.knownTransports[t.Name] = spec
	return nil
}

// MustRegisterTransport registers the given TransportSpec with the
// Configurator. This function panics if the TransportSpec is invalid.
func (c *Configurator) MustRegisterTransport(t TransportSpec) {
	if err := c.RegisterTransport(t); err != nil {
		panic(err)
	}
}

// RegisterPeerChooser registers a PeerChooserSpec with the given Configurator,
// teaching it how to build peer choosers of this kind from configuration.
//
// An error is returned if the PeerChooserSpec is invalid. Use
// MustRegisterPeerChooser to panic in the case of registration failure.
//
// If a peer chooser with the same name already exists, it will be replaced.
//
// If a peer list is registered with the same name, it will be ignored.
//
// See PeerChooserSpec for details on how to integrate your own peer chooser
// with the system.
func (c *Configurator) RegisterPeerChooser(s PeerChooserSpec) error {
	if s.Name == "" {
		return errors.New("name is required")
	}

	spec, err := compilePeerChooserSpec(&s)
	if err != nil {
		return fmt.Errorf("invalid PeerChooserSpec for %q: %v", s.Name, err)
	}

	c.knownPeerChoosers[s.Name] = spec
	return nil
}

// MustRegisterPeerChooser registers the given PeerChooserSpec with the
// Configurator.
// This function panics if the PeerChooserSpec is invalid.
func (c *Configurator) MustRegisterPeerChooser(s PeerChooserSpec) {
	if err := c.RegisterPeerChooser(s); err != nil {
		panic(err)
	}
}

// RegisterPeerList registers a PeerListSpec with the given Configurator,
// teaching it how to build peer lists of this kind from configuration.
//
// An error is returned if the PeerListSpec is invalid. Use
// MustRegisterPeerList to panic in the case of registration failure.
//
// If a peer list with the same name already exists, it will be replaced.
//
// If a peer chooser is registered with the same name, this list will be
// ignored.
//
// See PeerListSpec for details on how to integrate your own peer list with
// the system.
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
// Configurator, teaching it how to build peer list updaters of this kind from
// configuration.
//
// Returns an error if the PeerListUpdaterSpec is invalid.  Use
// MustRegisterPeerListUpdater to panic if the registration fails.
//
// If a peer list updater with the same name already exists, it will be
// replaced.
//
// See PeerListUpdaterSpec for details on how to integrate your own peer list
// updater with the system.
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

// MustRegisterPeerListUpdater registers the given PeerListUpdaterSpec with
// the Configurator. This function panics if the PeerListUpdaterSpec is
// invalid.
func (c *Configurator) MustRegisterPeerListUpdater(s PeerListUpdaterSpec) {
	if err := c.RegisterPeerListUpdater(s); err != nil {
		panic(err)
	}
}

// RegisterCompressor registers the given Compressor for the configurator, so
// any transport can use the given compression strategy.
func (c *Configurator) RegisterCompressor(z transport.Compressor) error {
	if c.knownCompressors[z.Name()] != nil {
		return fmt.Errorf("compressor already registered on configurator for name %q", z.Name())
	}
	c.knownCompressors[z.Name()] = z
	return nil
}

// MustRegisterCompressor registers the given compressor or panics.
func (c *Configurator) MustRegisterCompressor(z transport.Compressor) {
	if err := c.RegisterCompressor(z); err != nil {
		panic(err)
	}
}

// LoadConfigFromYAML loads a yarpc.Config from YAML data. Use LoadConfig if
// you have already parsed a map[string]interface{} or
// map[interface{}]interface{}.
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
	if err := config.DecodeInto(&cfg, data); err != nil {
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

// Kit creates a dependency kit for the configurator, suitable for passing to
// spec builder functions.
func (c *Configurator) Kit(serviceName string) *Kit {
	return &Kit{
		name:     serviceName,
		c:        c,
		resolver: c.resolver,
	}
}

func (c *Configurator) load(serviceName string, cfg *yarpcConfig) (_ yarpc.Config, err error) {
	b := newBuilder(serviceName, c.Kit(serviceName))

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

	if e := c.validateLogging(cfg.Logging); e != nil {
		err = multierr.Append(err, e)
	}

	if err != nil {
		return yarpc.Config{}, err
	}

	yc, err := b.Build()
	if err != nil {
		return yc, err
	}

	cfg.Logging.fill(&yc)
	cfg.Metrics.fill(&yc)
	return yc, nil
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
	// AddUnaryOutbound, AddOnewayOutbound and AddStreamOutbound
	type adder func(*compiledTransportSpec, string, string, config.AttributeMap) error

	loadUsing := func(o *outbound, adder adder) error {
		var useHTTP2 bool
		o.Attributes.Get("useHTTP2", &useHTTP2)

		var spec *compiledTransportSpec
		var err error
		if useHTTP2 {
			spec, err = c.spec("http2")
			if err != nil {
				return fmt.Errorf("failed to load configuration for outbound %q: %v", name, err)
			}
		} else {
			spec, err = c.spec(o.Type)
			if err != nil {
				return fmt.Errorf("failed to load configuration for outbound %q: %v", name, err)
			}
		}

		// spec, err := c.spec(o.Type)
		// if err != nil {
		// 	return fmt.Errorf("failed to load configuration for outbound %q: %v", name, err)
		// }

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

	if stream := cfg.Stream; stream != nil {
		if err := loadUsing(stream, b.AddStreamOutbound); err != nil {
			return err
		}
	}

	return nil
}

func (c *Configurator) loadTransportInto(b *builder, name string, attrs config.AttributeMap) error {
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

// validateLogging validates if the given logging is valid or not.
func (c *Configurator) validateLogging(l logging) error {
	if (l.Levels.ApplicationError != nil || l.Levels.Failure != nil) && (l.Levels.ServerError != nil || l.Levels.ClientError != nil) {
		return fmt.Errorf("invalid logging configuration, failure/applicationError configuration can not be used with serverError/clientError")
	}

	if (l.Levels.Outbound.ApplicationError != nil || l.Levels.Outbound.Failure != nil) && (l.Levels.Outbound.ServerError != nil || l.Levels.Outbound.ClientError != nil) {
		return fmt.Errorf("invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError")
	}

	if (l.Levels.Inbound.ApplicationError != nil || l.Levels.Inbound.Failure != nil) && (l.Levels.Inbound.ServerError != nil || l.Levels.Inbound.ClientError != nil) {
		return fmt.Errorf("invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError")
	}

	return nil
}
