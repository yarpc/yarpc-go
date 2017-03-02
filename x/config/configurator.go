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

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/decode"
	errs "go.uber.org/yarpc/internal/errors"

	"gopkg.in/yaml.v2"
)

// Configurator helps build Dispatchers using runtime configuration.
type Configurator struct {
	knownTransports map[string]*compiledTransportSpec
}

// New sets up a new empty Configurator. The returned Configurator does not
// know about any transports. Individual TransportSpecs must be registered
// against it using the RegisterTransport function.
func New() *Configurator {
	return &Configurator{knownTransports: make(map[string]*compiledTransportSpec)}
}

// RegisterTransport registers a TransportSpec with the given Configurator.
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

// MustRegisterTransport is the same as RegisterTransport except it panics in
// case of failure.
func (c *Configurator) MustRegisterTransport(t TransportSpec) {
	if err := c.RegisterTransport(t); err != nil {
		panic(err)
	}
}

// LoadConfigFromYAML loads a YARPC configuration from YAML.
func (c *Configurator) LoadConfigFromYAML(r io.Reader) (yarpc.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return yarpc.Config{}, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return yarpc.Config{}, err
	}
	return c.LoadConfig(data)
}

// LoadConfig a YARPC configuration from the given data.
func (c *Configurator) LoadConfig(data interface{}) (yarpc.Config, error) {
	var cfg yarpcConfig
	if err := decode.Decode(&cfg, data); err != nil {
		return yarpc.Config{}, err
	}
	return c.load(&cfg)
}

// NewDispatcherFromYAML builds a Dispatcher from the given YAML
// configuration.
func (c *Configurator) NewDispatcherFromYAML(r io.Reader) (*yarpc.Dispatcher, error) {
	cfg, err := c.LoadConfigFromYAML(r)
	if err != nil {
		return nil, err
	}
	return yarpc.NewDispatcher(cfg), nil
}

// NewDispatcher builds a new Dispatcher from the given data.
func (c *Configurator) NewDispatcher(data interface{}) (*yarpc.Dispatcher, error) {
	cfg, err := c.LoadConfig(data)
	if err != nil {
		return nil, err
	}
	return yarpc.NewDispatcher(cfg), nil
}

func (c *Configurator) load(cfg *yarpcConfig) (yarpc.Config, error) {
	b := newBuilder(cfg.Name)

	var errors []error

	for _, inbound := range cfg.Inbounds {
		if inbound.Disabled {
			continue
		}

		spec, err := c.spec(inbound.Type)
		if err != nil {
			// TODO: Maybe we should keep track of the inbound name so that if
			// it differs from the transport name, we can mention that in the
			// error message.
			errors = append(errors, fmt.Errorf("failed to load inbound: %v", err))
			continue
		}

		if err := b.AddInboundConfig(spec, inbound.Attributes); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	for name, clientConfig := range cfg.Outbounds {
		if implicit := clientConfig.Implicit; implicit != nil {
			spec, err := c.spec(implicit.Type)
			if err != nil {
				errors = append(errors, fmt.Errorf(
					"failed to load configuration for outbound %q: %v", name, err))
				continue
			}

			if err := b.AddImplicitOutbound(spec, name, clientConfig.Service, implicit.Attributes); err != nil {
				errors = append(errors, err)
				continue
			}

			continue
		}

		if unary := clientConfig.Unary; unary != nil {
			spec, err := c.spec(unary.Type)
			if err != nil {
				errors = append(errors, fmt.Errorf(
					"failed to load configuration for unary outbound %q: %v", name, err))
				continue
			}

			if err := b.AddUnaryOutbound(spec, name, clientConfig.Service, unary.Attributes); err != nil {
				errors = append(errors, err)
				continue
			}
		}

		if oneway := clientConfig.Oneway; oneway != nil {
			spec, err := c.spec(oneway.Type)
			if err != nil {
				errors = append(errors, fmt.Errorf(
					"failed to load configuration for oneway outbound %q: %v", name, err))
				continue
			}

			if err := b.AddOnewayOutbound(spec, name, clientConfig.Service, oneway.Attributes); err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	for name, attrs := range cfg.Transports {
		spec, err := c.spec(name)
		if err != nil {
			errors = append(errors, fmt.Errorf(
				"failed to load configuration for transport %q: %v", name, err))
			continue
		}

		if err := b.AddTransportConfig(spec, attrs); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	if len(errors) > 0 {
		return yarpc.Config{}, errs.MultiError(errors)
	}

	return b.Build()
}

// Returns the compiled spec for the transport with the given name or an error
func (c *Configurator) spec(name string) (*compiledTransportSpec, error) {
	spec, ok := c.knownTransports[name]
	if !ok {
		return nil, fmt.Errorf("unknown transport %q", name)
	}
	return spec, nil
}
