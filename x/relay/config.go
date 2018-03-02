package relay

import (
	"fmt"
	"strings"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

const (
	unary  = "unary"
	oneway = "oneway"

	_true  = "true"
	_false = "false"
)

// Configuration specifies the type of proxies that can be defined on the
// Frontcar
type Configuration struct {
	// ServiceProxies are proxies from a `Service` to an `Outbound`
	ServiceProxies map[string]RouteConfig `yaml:"service-proxies"`

	// ShardProxies are proxies from a `ShardKey` to an `Outbound`
	ShardProxies map[string]RouteConfig `yaml:"shardkey-proxies"`

	// Default is the default route config.
	Default RouteConfig `yaml:"default"`
}

// GenerateServiceHandlers generates a map of service to HandlerSpec for requests.
func (c Configuration) GenerateServiceHandlers(d *yarpc.Dispatcher, logger *zap.Logger) []ServiceHandler {
	handlers := make([]ServiceHandler, 0, len(c.ServiceProxies))
	for service, cfg := range c.ServiceProxies {
		if cfg.OverrideServiceName != _true { // If not explicitly true, don't override the service name.
			cfg.OverrideServiceName = _false
		}
		handlers = append(handlers, ServiceHandler{
			Service:     service,
			HandlerSpec: cfg.generateHandlerSpec(d, logger),
			Signature:   fmt.Sprintf("Proxy(Service=%q, Outbound=%q)", service, cfg.OutboundKey),
		})
	}
	return handlers
}

// GenerateShardKeyHandlers generates a map of shardkey to HandlerSpec for requests.
func (c Configuration) GenerateShardKeyHandlers(d *yarpc.Dispatcher, logger *zap.Logger) []ShardKeyHandler {
	handlers := make([]ShardKeyHandler, 0, len(c.ShardProxies))
	for shard, cfg := range c.ShardProxies {
		if cfg.OverrideServiceName != _false { // if not explicitly false, override the service name.
			cfg.OverrideServiceName = _true
		}
		handlers = append(handlers, ShardKeyHandler{
			ShardKey:    shard,
			HandlerSpec: cfg.generateHandlerSpec(d, logger),
			Signature:   fmt.Sprintf("Proxy(ShardKey=%q, Outbound=%q)", shard, cfg.OutboundKey),
		})
	}
	return handlers
}

// GenerateDefaultHandler generates a procedure for the default handler, the
// default handler is optional, so it will return an ok boolean to determine if
// a default was set.
func (c Configuration) GenerateDefaultHandler(d *yarpc.Dispatcher, logger *zap.Logger) (_ transport.Procedure, ok bool) {
	if (RouteConfig{}) == c.Default {
		return transport.Procedure{}, false
	}
	return transport.Procedure{
		Name:        "*", // `*` means that we are a proxy
		Service:     "*", // `*` means that we are a proxy
		HandlerSpec: c.Default.generateHandlerSpec(d, logger),
		Signature:   fmt.Sprintf("DefaultProxy(Outbound=%q)", c.Default.OutboundKey),
	}, true
}

// RouteConfig defines how a proxy is configured
type RouteConfig struct {
	// OutboundKey specifies an outbound defined in the rpc dispatcher
	// where we want to proxy traffic.
	OutboundKey string `yaml:"outbound"`

	// RPCType specifies the request type that will be going through this proxy.
	// (unary/oneway)
	RPCType string `yaml:"rpctype"`

	// OverrideServiceName specifies the service name of the request will be
	// overridden with the service name from the outbound.
	// This can be set to one of "true" or "false".  If this is not set it will
	// be set to whatever the default for the proxy type is (Service proxies are
	// set to "false", Shard proxies set it to "true").
	OverrideServiceName string `yaml:"overrideServiceName"`
}

func (r RouteConfig) generateHandlerSpec(d *yarpc.Dispatcher, logger *zap.Logger) transport.HandlerSpec {
	overrideServiceName := r.OverrideServiceName == _true
	cc := d.ClientConfig(r.OutboundKey)
	switch strings.ToLower(r.RPCType) {
	case unary:
		var h transport.UnaryHandler
		if overrideServiceName {
			h = UnaryProxyHandler(cc.GetUnaryOutbound(), WithServiceName(cc.Service()), WithLogger(logger))
		} else {
			h = UnaryProxyHandler(cc.GetUnaryOutbound(), WithLogger(logger))
		}
		return transport.NewUnaryHandlerSpec(
			middleware.ApplyUnaryInbound(
				h,
				d.InboundMiddleware().Unary,
			),
		)
	case oneway:
		var h transport.OnewayHandler
		if overrideServiceName {
			h = OnewayProxyHandler(cc.GetOnewayOutbound(), WithServiceName(cc.Service()), WithLogger(logger))
		} else {
			h = OnewayProxyHandler(cc.GetOnewayOutbound(), WithLogger(logger))
		}
		return transport.NewOnewayHandlerSpec(
			middleware.ApplyOnewayInbound(
				h,
				d.InboundMiddleware().Oneway,
			),
		)
	default:
		panic("Unsupported transport type for proxies " + r.RPCType)
	}
}
