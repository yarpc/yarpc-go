package config

import (
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

// YARPC TODO
type YARPC struct {
	Name       string
	Inbounds   []InboundConfig
	Outbounds  []OutboundConfig
	Transports []TransportConfig
}

// Build TODO
func (c *YARPC) Build() (yarpc.Config, error) {
	cfg := yarpc.Config{Name: c.Name, Outbounds: make(yarpc.Outbounds)}

	transports := make(map[string]transport.Transport)
	for _, tcfg := range c.Transports {
		t, err := tcfg.Builder.BuildTransport()
		if err != nil {
			return cfg, fmt.Errorf("failed to build transport %q: %v", tcfg.Name, err)
		}
		transports[tcfg.Name] = t
	}

	for _, icfg := range c.Inbounds {
		tname := icfg.TransportName
		// TODO: error if transport not found in map
		inbound, err := icfg.Builder.BuildInbound(transports[tname])
		if err != nil {
			return cfg, fmt.Errorf("failed to build inbound %q: %v", tname, err)
		}
		cfg.Inbounds = append(cfg.Inbounds, inbound)
	}

	for _, ocfg := range c.Outbounds {
		outbounds := transport.Outbounds{ServiceName: ocfg.Service}
		if ocfg.Oneway != nil {
			tname := ocfg.Oneway.TransportName
			// TODO: error if transport not found in map
			oneway, err := ocfg.Oneway.Builder.BuildOnewayOutbound(transports[tname])
			if err != nil {
				return cfg, fmt.Errorf("failed to build oneway outbound %q: %v", ocfg.Name, err)
			}
			outbounds.Oneway = oneway
		}
		if ocfg.Unary != nil {
			tname := ocfg.Unary.TransportName
			// TODO: error if transport not found in map
			unary, err := ocfg.Unary.Builder.BuildUnaryOutbound(transports[tname])
			if err != nil {
				return cfg, fmt.Errorf("failed to build unary outbound %q: %v", ocfg.Name, err)
			}
			outbounds.Unary = unary
		}
		cfg.Outbounds[ocfg.Name] = outbounds
	}

	return cfg, nil
}

// TransportConfig TODO
type TransportConfig struct {
	Name    string
	Builder TransportBuilder
}

// InboundConfig TODO
type InboundConfig struct {
	TransportName string
	Builder       InboundBuilder
}

// OutboundConfig TODO
type OutboundConfig struct {
	Name    string
	Service string
	Unary   *UnaryOutboundConfig
	Oneway  *OnewayOutboundConfig
}

// UnaryOutboundConfig TODO
type UnaryOutboundConfig struct {
	TransportName string
	Builder       UnaryOutboundBuilder
}

// OnewayOutboundConfig TODO
type OnewayOutboundConfig struct {
	TransportName string
	Builder       OnewayOutboundBuilder
}

// TransportBuilder TODO
type TransportBuilder interface {
	BuildTransport() (transport.Transport, error)
}

// InboundBuilder TODO
type InboundBuilder interface {
	BuildInbound(transport.Transport) (transport.Inbound, error)
}

// UnaryOutboundBuilder TODO
type UnaryOutboundBuilder interface {
	BuildUnaryOutbound(transport.Transport) (transport.UnaryOutbound, error)
}

// OnewayOutboundBuilder TODO
type OnewayOutboundBuilder interface {
	BuildOnewayOutbound(transport.Transport) (transport.OnewayOutbound, error)
}
