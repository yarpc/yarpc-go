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
	_inboundConfigurationKey  = "yarpchttp.inbounds"
	_outboundConfigurationKey = "yarpchttp.outbounds"
)

// Module produces yarpchttp clients and starts yarpchttp inbounds.
var Module = fx.Options(
	fx.Provide(NewClients),
	fx.Invoke(StartInbounds),
)

// InboundConfig is the configuration for starting yarpchttp inbounds.
type InboundConfig struct {
	Address string `yaml:"address"`
}

// StartInboundsParams defines the dependencies of this module.
type StartInboundsParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Router    yarpc.Router
	Provider  config.Provider
	Logger    *zap.Logger        `optional:"true"`
	Tracer    opentracing.Tracer `optional:"true"`
}

// StartInbounds constructs and starts yarpchttp inbounds.
func StartInbounds(p StartInboundsParams) error {
	ic := InboundConfig{}
	if err := p.Provider.Get(_inboundConfigurationKey).Populate(&ic); err != nil {
		return err
	}
	inbound := yarpchttp.Inbound{
		Addr:   ic.Address,
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

// Outbounds is the configuration for constructing a set of yarpchttp
// outbounds.
type Outbounds struct {
	Clients map[string]OutboundConfig `yaml:",inline"`
}

// OutboundConfig is the configuration for constructing a specific
// yarpchttp outbound.
type OutboundConfig struct {
	Address string `yaml:"address"`
}

// ClientParams defines the dependencies of this module.
type ClientParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Provider  config.Provider
	Logger    *zap.Logger        `optional:"true"`
	Tracer    opentracing.Tracer `optional:"true"`
}

// ClientResult defines the values produced by this module.
type ClientResult struct {
	fx.Out

	Clients []yarpc.Client `group:"yarpcfx"`
}

// NewClients produces yarpchttp yarpc.Clients.
func NewClients(p ClientParams) (ClientResult, error) {
	oc := Outbounds{}
	if err := p.Provider.Get(_outboundConfigurationKey).Populate(&oc); err != nil {
		return ClientResult{}, err
	}
	var clients []yarpc.Client
	for name, o := range oc.Clients {
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
