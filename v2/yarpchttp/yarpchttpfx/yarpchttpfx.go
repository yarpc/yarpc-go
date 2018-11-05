package yarpchttpfx

import (
	"context"
	"net/url"

	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/config"
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/zap"
)

const (
	_name                     = "yarpchttpfx"
	_inboundConfigurationKey  = "yarpc.http.inbounds"
	_outboundConfigurationKey = "yarpc.http.outbounds"
)

// Module produces yarpchttp clients and starts yarpchttp inbounds.
var Module = fx.Options(
	fx.Provide(NewInboundConfig),
	fx.Provide(NewOutboundsConfig),
	fx.Provide(NewClients),
	fx.Invoke(StartInbounds),
)

// InboundConfig is the configuration for starting yarpchttp inbounds.
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
	inbound := yarpchttp.Inbound{
		Addr:   p.Config.Address,
		Router: p.Router,
		Logger: p.Logger,
		Tracer: p.Tracer,
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
	Clients map[string]OutboundConfig `yaml:",inline"`
}

// OutboundConfig is the configuration for constructing a specific outbound.
type OutboundConfig struct {
	Address string `yaml:"address"`
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
	ic := OutboundsConfig{}
	if err := p.Provider.Get(_outboundConfigurationKey).Populate(&ic); err != nil {
		return OutboundsConfigResult{}, err
	}
	return OutboundsConfigResult{
		Config: ic,
	}, nil
}

// ClientParams defines the dependencies of this module.
type ClientParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    OutboundsConfig
	Logger    *zap.Logger        `optional:"true"`
	Tracer    opentracing.Tracer `optional:"true"`
}

// ClientResult defines the values produced by this module.
type ClientResult struct {
	fx.Out

	Clients []yarpc.Client `group:"yarpcfx"`
}

// NewClients produces yarpc.Clients.
func NewClients(p ClientParams) (ClientResult, error) {
	var clients []yarpc.Client
	for name, o := range p.Config.Clients {
		url, err := url.Parse(o.Address)
		if err != nil {
			return ClientResult{}, err
		}
		dialer := &yarpchttp.Dialer{
			Tracer: p.Tracer,
			Logger: p.Logger,
		}
		outbound := &yarpchttp.Outbound{
			Dialer: dialer,
			URL:    url,
			Tracer: p.Tracer,
		}
		p.Lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return dialer.Start(ctx)
			},
			OnStop: func(ctx context.Context) error {
				return dialer.Stop(ctx)
			},
		})
		clients = append(
			clients,
			yarpc.Client{
				Caller:  "foo", // TODO(amckinney): Derive from servicefx.
				Service: name,
				Unary:   outbound,
			},
		)
	}
	return ClientResult{
		Clients: clients,
	}, nil
}
