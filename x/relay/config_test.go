package relay

import (
	"testing"

	"strings"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name   string
		Config Configuration

		// give{Unary,Oneway}Outbounds specifies a series of mock unary/oneway
		// outbounds to create in the dispatcher.
		giveUnaryOutbounds  []string
		giveOnewayOutbounds []string

		wantPanic bool
	}{
		{
			name: "single service unary",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
				},
			},
			giveUnaryOutbounds: []string{"myout"},
			wantPanic:          false,
		},
		{
			name: "missing service unary outbound",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "single service oneway",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "oneway",
					},
				},
			},
			giveOnewayOutbounds: []string{"myout"},
			wantPanic:           false,
		},
		{
			name: "missing service oneway outbound",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "oneway",
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "invalid service rpctype",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "totally-real-type",
					},
				},
			},
			giveUnaryOutbounds: []string{"myout"},
			wantPanic:          true,
		},
		{
			name: "multiple services",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myservice2": {
						OutboundKey: "myout2",
						RPCType:     "oneway",
					},
					"myservice3": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myservice4": {
						OutboundKey: "myout4",
						RPCType:     "unary",
					},
				},
			},
			giveUnaryOutbounds:  []string{"myout", "myout4"},
			giveOnewayOutbounds: []string{"myout2"},
			wantPanic:           false,
		},
		{
			name: "single shard unary",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
				},
			},
			giveUnaryOutbounds: []string{"myout"},
			wantPanic:          false,
		},
		{
			name: "missing shard unary outbound",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "single shard oneway",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "oneway",
					},
				},
			},
			giveOnewayOutbounds: []string{"myout"},
			wantPanic:           false,
		},
		{
			name: "missing shard oneway outbound",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "oneway",
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "invalid shard rpctype",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "totally-real-type",
					},
				},
			},
			giveUnaryOutbounds: []string{"myout"},
			wantPanic:          true,
		},
		{
			name: "multiple shards",
			Config: Configuration{
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myshard2": {
						OutboundKey: "myout2",
						RPCType:     "oneway",
					},
					"myshard3": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myshard4": {
						OutboundKey: "myout4",
						RPCType:     "unary",
					},
				},
			},
			giveUnaryOutbounds:  []string{"myout", "myout4"},
			giveOnewayOutbounds: []string{"myout2"},
			wantPanic:           false,
		},
		{
			name: "multiple shards and services",
			Config: Configuration{
				ServiceProxies: map[string]RouteConfig{
					"myservice": {
						OutboundKey: "myserviceout",
						RPCType:     "unary",
					},
					"myservice2": {
						OutboundKey: "myserviceout2",
						RPCType:     "oneway",
					},
					"myservice3": {
						OutboundKey: "myserviceout",
						RPCType:     "unary",
					},
					"myservice4": {
						OutboundKey: "myserviceout4",
						RPCType:     "unary",
					},
				},
				ShardProxies: map[string]RouteConfig{
					"myshard": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myshard2": {
						OutboundKey: "myout2",
						RPCType:     "oneway",
					},
					"myshard3": {
						OutboundKey: "myout",
						RPCType:     "unary",
					},
					"myshard4": {
						OutboundKey: "myout4",
						RPCType:     "unary",
					},
				},
			},
			giveUnaryOutbounds:  []string{"myout", "myout4", "myserviceout", "myserviceout4"},
			giveOnewayOutbounds: []string{"myout2", "myserviceout2"},
			wantPanic:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			dispatcher := getDispatcherWithOutbounds(
				tc.giveUnaryOutbounds,
				tc.giveOnewayOutbounds,
				mockCtrl,
			)

			if tc.wantPanic {
				assert.Panics(t, func() {
					tc.Config.GenerateServiceHandlers(dispatcher, zap.NewNop())
					tc.Config.GenerateShardKeyHandlers(dispatcher, zap.NewNop())
				})
				return
			}

			serviceHandlers := tc.Config.GenerateServiceHandlers(dispatcher, zap.NewNop())
			serviceHandlerMap := make(map[string]ServiceHandler)
			for _, handler := range serviceHandlers {
				serviceHandlerMap[handler.Service] = handler
			}
			assert.Len(t, serviceHandlers, len(tc.Config.ServiceProxies))
			for service, cfg := range tc.Config.ServiceProxies {
				h, ok := serviceHandlerMap[strings.ToLower(service)]
				require.True(t, ok, "no handler was created for %s", service)
				require.NotNil(t, h, "handler for %s was nil", service)

				switch strings.ToLower(cfg.RPCType) {
				case unary:
					assert.Equal(t, h.HandlerSpec.Type(), transport.Unary, "invalid handler type for %s", service)
				case oneway:
					assert.Equal(t, h.HandlerSpec.Type(), transport.Oneway, "invalid handler type for %s", service)
				default:
					assert.Fail(t, "invalid handler type (want(%s), got(%s)) for %s", cfg.RPCType, string(h.HandlerSpec.Type()), service)
				}
			}

			shardHandlers := tc.Config.GenerateShardKeyHandlers(dispatcher, zap.NewNop())
			shardHandlerMap := make(map[string]ShardKeyHandler)
			for _, handler := range shardHandlers {
				shardHandlerMap[handler.ShardKey] = handler
			}
			assert.Len(t, shardHandlers, len(tc.Config.ShardProxies))
			for shard, cfg := range tc.Config.ShardProxies {
				h, ok := shardHandlerMap[strings.ToLower(shard)]
				require.True(t, ok, "no handler was created for %s", shard)
				require.NotNil(t, h, "handler for %s was nil", shard)

				switch strings.ToLower(cfg.RPCType) {
				case unary:
					assert.Equal(t, h.HandlerSpec.Type(), transport.Unary, "invalid handler type for %s", shard)
				case oneway:
					assert.Equal(t, h.HandlerSpec.Type(), transport.Oneway, "invalid handler type for %s", shard)
				default:
					assert.Fail(t, "invalid handler type (want(%s), got(%s)) for %s", cfg.RPCType, string(h.HandlerSpec.Type()), shard)
				}
			}
		})
	}
}

func getDispatcherWithOutbounds(unaryOuts, onewayOuts []string, mockCtrl *gomock.Controller) *yarpc.Dispatcher {
	outbounds := make(yarpc.Outbounds, len(unaryOuts)+len(onewayOuts))
	for _, outKey := range unaryOuts {
		out := transporttest.NewMockUnaryOutbound(mockCtrl)
		out.EXPECT().Transports().AnyTimes().Return([]transport.Transport{})
		outbounds[outKey] = transport.Outbounds{
			Unary: out,
		}
	}
	for _, outKey := range onewayOuts {
		out := transporttest.NewMockOnewayOutbound(mockCtrl)
		out.EXPECT().Transports().AnyTimes().Return([]transport.Transport{})
		outbounds[outKey] = transport.Outbounds{
			Oneway: out,
		}
	}

	cfg := yarpc.Config{
		Name:      "service",
		Outbounds: outbounds,
		Metrics: yarpc.MetricsConfig{
			Tally: tally.NoopScope,
		},
	}

	return yarpc.NewDispatcher(cfg)
}
