// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcmiddlewarefx

import (
	"fmt"

	"go.uber.org/config"
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
)

// InboundTransportConfig describes the configuration
// shape for an ordered list of unary inbound transport middleware.
type InboundTransportConfig struct {
	Unary []string `yaml:"unary"`
}

// InboundTransportConfigParams defines the dependencies of this module.
type InboundTransportConfigParams struct {
	fx.In

	Provider config.Provider
}

// InboundTransportConfigResult defines the values produced by this module.
type InboundTransportConfigResult struct {
	fx.Out

	Config InboundTransportConfig
}

// NewInboundTransportConfig produces an UnaryInboundTransportConfig.
func NewInboundTransportConfig(p InboundTransportConfigParams) (InboundTransportConfigResult, error) {
	mc := InboundTransportConfig{}
	if err := p.Provider.Get(inboundTransportConfigurationKey).Populate(&mc); err != nil {
		return InboundTransportConfigResult{}, err
	}
	return InboundTransportConfigResult{
		Config: mc,
	}, nil
}

// UnaryInboundTransportParams defines the dependencies of this module.
type UnaryInboundTransportParams struct {
	fx.In

	Config          InboundTransportConfig
	Middleware      []yarpc.UnaryInboundTransportMiddleware   `group:"yarpcfx"`
	MiddlewareLists [][]yarpc.UnaryInboundTransportMiddleware `group:"yarpcfx"`
}

// UnaryInboundTransportResult defines the values produced by this module.
type UnaryInboundTransportResult struct {
	fx.Out

	// An ordered slice of middleware according to the given configuration.
	OrderedMiddleware []yarpc.UnaryInboundTransportMiddleware `name:"yarpcfx"`
}

// NewUnaryInboundTransport produces an ordered slice of unary inbound transport middleware.
func NewUnaryInboundTransport(
	p UnaryInboundTransportParams,
) (UnaryInboundTransportResult, error) {
	// Collect all of the middleware into a single slice.
	middleware := p.Middleware
	for _, ml := range p.MiddlewareLists {
		middleware = append(middleware, ml...)
	}

	// Compose a map of the middleware, and validate that there are not any name conflicts.
	middlewareMap := make(map[string]yarpc.UnaryInboundTransportMiddleware, len(middleware))
	for _, m := range middleware {
		name := m.Name()
		if _, ok := middlewareMap[name]; ok {
			return UnaryInboundTransportResult{}, fmt.Errorf("unary inbound transport middleware %q was registered more than once", name)
		}
		middlewareMap[name] = m
	}

	// Construct an ordered slice of middleware using the configured slice of names.
	ordered := make([]yarpc.UnaryInboundTransportMiddleware, len(p.Config.Unary))
	for i, name := range p.Config.Unary {
		m, ok := middlewareMap[name]
		if !ok {
			return UnaryInboundTransportResult{}, fmt.Errorf("failed to resolve unary inbound transport middleware: %q", name)
		}
		ordered[i] = m
	}

	return UnaryInboundTransportResult{
		OrderedMiddleware: ordered,
	}, nil
}
