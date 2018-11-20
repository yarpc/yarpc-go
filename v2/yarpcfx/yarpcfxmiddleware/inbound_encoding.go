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

package yarpcfxmiddleware

import (
	"fmt"

	"go.uber.org/config"
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
)

// InboundEncodingConfig describes the configuration
// shape for an ordered list of unary inbound encoding middleware.
type InboundEncodingConfig struct {
	Unary []string `yaml:"unary"`
}

// InboundEncodingConfigParams defines the dependencies of this module.
type InboundEncodingConfigParams struct {
	fx.In

	Provider config.Provider
}

// InboundEncodingConfigResult defines the values produced by this module.
type InboundEncodingConfigResult struct {
	fx.Out

	Config InboundEncodingConfig
}

// newInboundEncodingConfig produces an UnaryInboundEncodingConfig.
func newInboundEncodingConfig(p InboundEncodingConfigParams) (InboundEncodingConfigResult, error) {
	mc := InboundEncodingConfig{}
	if err := p.Provider.Get(inboundEncodingConfigurationKey).Populate(&mc); err != nil {
		return InboundEncodingConfigResult{}, err
	}
	return InboundEncodingConfigResult{
		Config: mc,
	}, nil
}

// UnaryInboundEncodingParams defines the dependencies of this module.
type UnaryInboundEncodingParams struct {
	fx.In

	Config          InboundEncodingConfig
	Middleware      []yarpc.UnaryInboundEncodingMiddleware   `group:"yarpcfx"`
	MiddlewareLists [][]yarpc.UnaryInboundEncodingMiddleware `group:"yarpcfx"`
}

// UnaryInboundEncodingResult defines the values produced by this module.
type UnaryInboundEncodingResult struct {
	fx.Out

	// An ordered slice of middleware according to the given configuration.
	OrderedMiddleware []yarpc.UnaryInboundEncodingMiddleware `name:"yarpcfx"`
}

// newUnaryInboundEncoding produces an ordered slice of unary inbound encoding middleware.
func newUnaryInboundEncoding(
	p UnaryInboundEncodingParams,
) (UnaryInboundEncodingResult, error) {
	// Collect all of the middleware into a single slice.
	middleware := p.Middleware
	for _, ml := range p.MiddlewareLists {
		middleware = append(middleware, ml...)
	}

	// Compose a map of the middleware, and validate that there are not any name conflicts.
	middlewareMap := make(map[string]yarpc.UnaryInboundEncodingMiddleware, len(middleware))
	for _, m := range middleware {
		name := m.Name()
		if _, ok := middlewareMap[name]; ok {
			return UnaryInboundEncodingResult{}, fmt.Errorf("unary inbound encoding middleware %q was registered more than once", name)
		}
		middlewareMap[name] = m
	}

	// Construct an ordered slice of middleware using the configured slice of names.
	ordered := make([]yarpc.UnaryInboundEncodingMiddleware, len(p.Config.Unary))
	for i, name := range p.Config.Unary {
		m, ok := middlewareMap[name]
		if !ok {
			return UnaryInboundEncodingResult{}, fmt.Errorf("failed to resolve unary inbound encoding middleware: %q", name)
		}
		ordered[i] = m
	}

	return UnaryInboundEncodingResult{
		OrderedMiddleware: ordered,
	}, nil
}
