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

const outboundTransportMiddlewareConfigurationKey = "yarpc.middleware.outbounds.transport"

// Module produces ordered slices of middleware according to
// the middleware configuration.
var Module = fx.Provide(
	NewOutboundTransportMiddlewareConfig,
	NewUnaryOutboundTransportMiddleware,
)

// OutboundTransportMiddlewareConfig describes the configuration
// shape for an ordered list of unary outbound transport middleware.
type OutboundTransportMiddlewareConfig struct {
	Unary []string `yaml:"unary"`
}

// OutboundTransportMiddlewareConfigParams defines the dependencies of this module.
type OutboundTransportMiddlewareConfigParams struct {
	fx.In

	Provider config.Provider
}

// OutboundTransportMiddlewareConfigResult defines the values produced by this module.
type OutboundTransportMiddlewareConfigResult struct {
	fx.Out

	Config OutboundTransportMiddlewareConfig
}

// NewOutboundTransportMiddlewareConfig produces an UnaryOutboundTransportMiddlewareConfig.
func NewOutboundTransportMiddlewareConfig(p OutboundTransportMiddlewareConfigParams) (OutboundTransportMiddlewareConfigResult, error) {
	mc := OutboundTransportMiddlewareConfig{}
	if err := p.Provider.Get(outboundTransportMiddlewareConfigurationKey).Populate(&mc); err != nil {
		return OutboundTransportMiddlewareConfigResult{}, err
	}
	return OutboundTransportMiddlewareConfigResult{
		Config: mc,
	}, nil
}

// UnaryOutboundTransportMiddlewareParams defines the dependencies of this module.
type UnaryOutboundTransportMiddlewareParams struct {
	fx.In

	Config          OutboundTransportMiddlewareConfig
	Middleware      []yarpc.UnaryOutboundTransportMiddleware   `group:"yarpcfx"`
	MiddlewareLists [][]yarpc.UnaryOutboundTransportMiddleware `group:"yarpcfx"`
}

// UnaryOutboundTransportMiddlewareResult defines the values produced by this module.
type UnaryOutboundTransportMiddlewareResult struct {
	fx.Out

	Middleware []yarpc.UnaryOutboundTransportMiddleware `name:"yarpcfx"`
}

// NewUnaryOutboundTransportMiddleware produceds an ordered slice of unary outbound transport middleware.
func NewUnaryOutboundTransportMiddleware(
	p UnaryOutboundTransportMiddlewareParams,
) (UnaryOutboundTransportMiddlewareResult, error) {
	// Collect all of the middleware into a single slice.
	middleware := p.Middleware
	for _, ml := range p.MiddlewareLists {
		middleware = append(middleware, ml...)
	}

	// Compose a map of the middleware, and validate that there are not any name conflicts.
	middlewareMap := make(map[string]yarpc.UnaryOutboundTransportMiddleware, len(middleware))
	for _, m := range middleware {
		name := m.Name()
		if _, ok := middlewareMap[name]; ok {
			return UnaryOutboundTransportMiddlewareResult{}, fmt.Errorf("unary outbound transport middleware %q was registered more than once", name)
		}
		middlewareMap[name] = m
	}

	// Construct an ordered slice of middleware using the configured slice of names.
	ordered := make([]yarpc.UnaryOutboundTransportMiddleware, len(p.Config.Unary))
	for i, name := range p.Config.Unary {
		m, ok := middlewareMap[name]
		if !ok {
			return UnaryOutboundTransportMiddlewareResult{}, fmt.Errorf("failed to resolve unary outbound transport middleware: %q", name)
		}
		ordered[i] = m
	}

	return UnaryOutboundTransportMiddlewareResult{
		Middleware: ordered,
	}, nil
}
