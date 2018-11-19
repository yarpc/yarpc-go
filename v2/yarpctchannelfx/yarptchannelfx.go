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

package yarpctchannelfx

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctchannel"
	"go.uber.org/zap"
)

const (
	_name                     = "yarpctchannelfx"
	_inboundConfigurationKey  = "yarpc.tchannel.inbounds"
	_outboundConfigurationKey = "yarpc.tchannel.outbounds"
)

// Module produces yarpctchannel clients and starts yarpctchannel inbounds.
var Module = fx.Options(
	fx.Provide(NewInboundConfig),
	fx.Provide(NewOutboundsConfig),
	fx.Provide(NewClients),
	fx.Provide(NewDialer),
	fx.Invoke(StartInbounds),
)

// InboundConfig is the configuration for starting yarpctchannel inbounds.
type InboundConfig struct {
	Address string `yaml:"address"`
}

// InboundConfigParams defines the dependencies of this module.
type InboundConfigParams struct {
	fx.In

	Provider config.Provider
}

// InboundConfigResult defines the values produced by this module.
type InboundConfigResult struct {
	fx.Out

	Config InboundConfig
}

// NewInboundConfig produces an InboundConfig.
func NewInboundConfig(p InboundConfigParams) (InboundConfigResult, error) {
	ic := InboundConfig{}
	if err := p.Provider.Get(_inboundConfigurationKey).Populate(&ic); err != nil {
		return InboundConfigResult{}, err
	}
	return InboundConfigResult{
		Config: ic,
	}, nil
}

// StartInboundsParams defines the dependencies of this module.
type StartInboundsParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Router    yarpc.Router
	Config    InboundConfig
	Logger    *zap.Logger        `optional:"true"`
	Tracer    opentracing.Tracer `optional:"true"`
}

// StartInbounds constructs and starts inbounds.
func StartInbounds(p StartInboundsParams) error {
	inbound := yarpctchannel.Inbound{
		Service: "foo", //TODO(apeatsbond): grab from servicefx
		Addr:    p.Config.Address,
		Router:  p.Router,
		Tracer:  p.Tracer,
		Logger:  p.Logger,
	}
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return inbound.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return inbound.Stop(ctx)
		},
	})
	return nil
}

// OutboundsConfig is the configuration for constructing a set of outbounds.
type OutboundsConfig struct {
	Outbounds map[string]OutboundConfig `yaml:",inline"`
}

// OutboundConfig is the configuration for constructing a specific outbound.
type OutboundConfig struct {
	// Specifies the outbound's service name.
	//
	// If not set, defaults to the configured outbound key.
	Service string `yaml:"name"`

	// Specifies the address to use for this Outbound.
	Address string `yaml:"address"`

	// Specifies the peer list chooser to use for this Outbound.
	//
	// If set, an address does not need to be configured.
	Chooser string `yaml:"chooser"`
}

// OutboundsConfigParams defines the dependencies of this module.
type OutboundsConfigParams struct {
	fx.In

	Provider config.Provider
}

// OutboundsConfigResult defines the values produced by this module.
type OutboundsConfigResult struct {
	fx.Out

	Config OutboundsConfig
}

// NewOutboundsConfig produces an OutboundsConfig.
func NewOutboundsConfig(p OutboundsConfigParams) (OutboundsConfigResult, error) {
	oc := OutboundsConfig{}
	if err := p.Provider.Get(_outboundConfigurationKey).Populate(&oc); err != nil {
		return OutboundsConfigResult{}, err
	}
	return OutboundsConfigResult{
		Config: oc,
	}, nil
}

// ClientParams defines the dependencies of this module.
type ClientParams struct {
	fx.In

	Lifecycle       fx.Lifecycle
	Config          OutboundsConfig
	Dialer          *yarpctchannel.Dialer
	ChooserProvider yarpc.ChooserProvider
}

// ClientResult defines the values produced by this module.
type ClientResult struct {
	fx.Out

	Clients []yarpc.Client `group:"yarpcfx"`
}

// NewClients produces yarpc.Clients.
func NewClients(p ClientParams) (ClientResult, error) {
	var clients []yarpc.Client
	for name, o := range p.Config.Outbounds {
		var chooser yarpc.Chooser

		if o.Chooser != "" {
			var ok bool
			chooser, ok = p.ChooserProvider.Chooser(o.Chooser)
			if !ok {
				return ClientResult{}, fmt.Errorf("failed to resolve outbound peer list chooser: %q", o.Chooser)
			}
		}

		outbound := &yarpctchannel.Outbound{
			Chooser: chooser,
			Addr:    o.Address,
			Dialer:  p.Dialer,
		}

		// If the outbound's service is configured, use it.
		// Otherwise, default to the outbound key.
		service := o.Service
		if service == "" {
			service = name
		}
		clients = append(
			clients,
			yarpc.Client{
				Name:    name,
				Caller:  "foo", // TODO(apeatsbond): grab from servicefx.
				Service: service,
				Unary:   outbound,
			},
		)
	}
	return ClientResult{
		Clients: clients,
	}, nil
}

// DialerParams defines the dependencies of this module.
type DialerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger        `optional:"true"`
	Tracer    opentracing.Tracer `optional:"true"`
}

// DialerResult defines the values produced by this module.
type DialerResult struct {
	fx.Out

	TChannelDialer *yarpctchannel.Dialer
	Dialer         yarpc.Dialer `group:"yarpcfx"`
}

// NewDialer produces a yarpc.Dialer.
func NewDialer(p DialerParams) (DialerResult, error) {
	dialer := &yarpctchannel.Dialer{
		Caller: "foo", //TODO(apeatsbond): grab from servicefx
		Tracer: p.Tracer,
		Logger: p.Logger,
	}
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return dialer.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return dialer.Stop(ctx)
		},
	})
	return DialerResult{
		TChannelDialer: dialer,
		Dialer:         dialer,
	}, nil
}
