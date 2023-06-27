// Copyright (c) 2022 Uber Technologies, Inc.
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

package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/yarpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

func TestNewTransportSpecOptions(t *testing.T) {
	transportSpec, err := newTransportSpec(
		BackoffStrategy(nil),
	)
	require.NoError(t, err)
	require.Equal(t, 1, len(transportSpec.TransportOptions))
	require.Equal(t, 0, len(transportSpec.InboundOptions))
	require.Equal(t, 0, len(transportSpec.OutboundOptions))
}

func TestConfigBuildInboundOtherTransport(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildInbound(&InboundConfig{}, testTransport{}, _kit)
	require.Equal(t, newTransportCastError(testTransport{}), err)
}

func TestConfigBuildInboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildInbound(&InboundConfig{}, NewTransport(), _kit)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestConfigBuildUnaryOutboundOtherTransport(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildUnaryOutbound(&OutboundConfig{}, testTransport{}, _kit)
	require.Equal(t, newTransportCastError(testTransport{}), err)
}

func TestConfigBuildUnaryOutboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildUnaryOutbound(&OutboundConfig{}, NewTransport(), _kit)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestConfigBuildStreamOutboundOtherTransport(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildStreamOutbound(&OutboundConfig{}, testTransport{}, _kit)
	require.Equal(t, newTransportCastError(testTransport{}), err)
}

func TestConfigBuildStreamOutboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildStreamOutbound(&OutboundConfig{}, NewTransport(), _kit)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestTransportSpecUnknownOption(t *testing.T) {
	assert.Panics(t, func() { TransportSpec(testOption{}) })
}

type fakeOutboundTLSConfigProvider struct {
	returnErr         error
	expectedSpiffeIDs []string
}

func (f fakeOutboundTLSConfigProvider) ClientTLSConfig(spiffeIDs []string) (*tls.Config, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	if !reflect.DeepEqual(f.expectedSpiffeIDs, spiffeIDs) {
		return nil, errors.New("spiffe IDs do not match")
	}
	return &tls.Config{}, nil
}

func TestTransportSpec(t *testing.T) {
	type attrs map[string]interface{}

	type wantInbound struct {
		Address                 string
		ServerMaxRecvMsgSize    int
		ServerMaxSendMsgSize    int
		ServerMaxHeaderListSize uint32
		ClientMaxRecvMsgSize    int
		ClientMaxSendMsgSize    int
		ClientMaxHeaderListSize uint32
		TLS                     bool
		TLSMode                 yarpctls.Mode
	}

	type wantOutbound struct {
		Address                 string
		TLS                     bool
		Compressor              string
		WantCustomContextDialer bool
		Keepalive               *keepalive.ClientParameters
		TLSConfig               bool
	}

	type test struct {
		desc string
		// must specify inboundCfg if transportCfg specified
		transportCfg  attrs
		inboundCfg    attrs
		outboundCfg   attrs
		env           map[string]string
		opts          []Option
		wantInbound   *wantInbound
		wantOutbounds map[string]wantOutbound
		wantErrors    []string
	}

	tests := []test{
		{
			desc:        "simple inbound",
			inboundCfg:  attrs{"address": ":54567", "tls": attrs{"mode": "enforced"}},
			wantInbound: &wantInbound{Address: ":54567", TLSMode: yarpctls.Enforced},
		},
		{
			desc:        "inbound interpolation",
			inboundCfg:  attrs{"address": "${HOST:}:${PORT}"},
			env:         map[string]string{"HOST": "127.0.0.1", "PORT": "54568"},
			wantInbound: &wantInbound{Address: "127.0.0.1:54568"},
		},
		{
			desc:       "bad inbound address",
			inboundCfg: attrs{"address": "derp"},
			wantErrors: []string{"address derp"},
		},
		{
			desc: "simple outbound",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"address": "localhost:54569"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54569",
				},
			},
		},
		{
			desc: "simple outbound with compressor",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address":    "localhost:54569",
						"compressor": "gzip",
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address:    "localhost:54569",
					Compressor: "gzip",
				},
			},
		},
		{
			desc: "outbound interpolation",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"address": "${ADDR}"},
				},
			},
			env: map[string]string{"ADDR": "127.0.0.1:54570"},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "127.0.0.1:54570",
				},
			},
		},
		{
			desc: "simple outbound with peer",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"peer": "localhost:54569"},
				},
			},
		},
		{
			desc: "outbound bad peer list",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"least-pending": []string{
							"127.0.0.1:8080",
							"127.0.0.1:8081",
							"127.0.0.1:8082",
						},
					},
				},
			},
			wantErrors: []string{
				`failed to configure unary outbound for "myservice"`,
				`failed to read attribute "least-pending"`,
			},
		},
		{
			desc: "unknown preset",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"with": "derp"},
				},
			},
			wantErrors: []string{
				`failed to configure unary outbound for "myservice":`,
				`no recognized peer chooser preset "derp"`,
			},
		},
		{
			desc: "inbound and transport with message size options",
			transportCfg: attrs{
				"serverMaxRecvMsgSize":    "1024",
				"serverMaxSendMsgSize":    "2048",
				"serverMaxHeaderListSize": "32768",
				"clientMaxRecvMsgSize":    "4096",
				"clientMaxSendMsgSize":    "8192",
				"clientMaxHeaderListSize": "16384",
			},
			inboundCfg: attrs{"address": ":54571"},
			wantInbound: &wantInbound{
				Address:                 ":54571",
				ServerMaxRecvMsgSize:    1024,
				ServerMaxSendMsgSize:    2048,
				ServerMaxHeaderListSize: 32768,
				ClientMaxRecvMsgSize:    4096,
				ClientMaxSendMsgSize:    8192,
				ClientMaxHeaderListSize: 16384,
			},
		},
		{
			desc: "TLS enabled on an inbound",
			inboundCfg: attrs{
				"address": "localhost:54569",
				"tls": attrs{
					"enabled":  true,
					"certFile": "testdata/cert",
					"keyFile":  "testdata/key",
				},
			},
			wantInbound: &wantInbound{
				Address: "127.0.0.1:54569",
				TLS:     true,
			},
		},
		{
			desc: "TLS enabled on an inbound with invalid config",
			inboundCfg: attrs{
				"address": "localhost:54713",
				"tls": attrs{
					"enabled": true,
				},
			},
			wantErrors: []string{`both certFile and keyFile`},
		},
		{
			desc: "TLS enabled on an outbound",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"tls": attrs{
							"enabled": true,
						},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54816",
					TLS:     true,
				},
			},
		},
		{
			desc: "simple outbound with custom dialer option",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"address": "localhost:54569"},
				},
			},
			opts: []Option{ContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "TCP", addr)
			})},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address:                 "localhost:54569",
					WantCustomContextDialer: true,
				},
			},
		},
		{
			desc: "simple outbound with keepalive params",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{"address": "localhost:54569"},
				},
			},
			opts: []Option{KeepaliveParams(keepalive.ClientParameters{
				Timeout: time.Second * 10,
				Time:    time.Second * 30,
			})},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54569",
					Keepalive: &keepalive.ClientParameters{
						Timeout: time.Second * 10,
						Time:    time.Second * 30,
					},
				},
			},
		},
		{
			desc: "Outbound with keepalive from attrs",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"grpc-keepalive": attrs{
							"enabled":               "true",
							"time":                  "30s",
							"timeout":               "20s",
							"permit-without-stream": "true",
						},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54816",
					Keepalive: &keepalive.ClientParameters{
						Timeout:             time.Second * 20,
						Time:                time.Second * 30,
						PermitWithoutStream: true,
					},
				},
			},
		},
		{
			desc: "Outbound with keepalive defaults",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"grpc-keepalive": attrs{
							"enabled": "true",
						},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54816",
					Keepalive: &keepalive.ClientParameters{
						Timeout: time.Second * 20,
						Time:    time.Second * 10,
					},
				},
			},
		},
		{
			desc: "invalid keepalive time",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"grpc-keepalive": attrs{
							"enabled": "true",
							"time":    "10foo",
							"timeout": "10",
						},
					},
				},
			},
			wantErrors: []string{
				`could not parse gRPC keepalive time: time: unknown unit`,
			},
		},
		{
			desc: "invalid keepalive timeout",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"grpc-keepalive": attrs{
							"enabled": "true",
							"time":    "10s",
							"timeout": "10foo",
						},
					},
				},
			},
			wantErrors: []string{
				`could not parse gRPC keepalive timeout: time: unknown unit`,
			},
		},
		{
			desc: "keepalive from attrs disabled",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54816",
						"grpc-keepalive": attrs{
							"enabled": "false",
							"time":    "10",
							"timeout": "10",
						},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54816",
				},
			},
		},
		{
			desc: "simple TLS outbound",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54569",
						"tls": attrs{
							"mode":       yarpctls.Enforced,
							"spiffe-ids": []string{"spiffe-test-1"},
						},
					},
				},
			},
			opts: []Option{OutboundTLSConfigProvider(&fakeOutboundTLSConfigProvider{
				expectedSpiffeIDs: []string{"spiffe-test-1"},
			})},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address:   "localhost:54569",
					TLSConfig: true,
				},
			},
		},
		{
			desc: "TLS outbound without spiffe id",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54569",
						"tls": attrs{
							"mode": yarpctls.Enforced,
						},
					},
				},
			},
			opts: []Option{OutboundTLSConfigProvider(&fakeOutboundTLSConfigProvider{})},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address:   "localhost:54569",
					TLSConfig: true,
				},
			},
		},
		{
			desc: "fail TLS outbound with invalid tls mode",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54569",
						"tls": attrs{
							"mode": yarpctls.Permissive,
						},
					},
				},
			},
			opts:       []Option{OutboundTLSConfigProvider(&fakeOutboundTLSConfigProvider{})},
			wantErrors: []string{"outbound does not support permissive TLS mode"},
		},
		{
			desc: "fail TLS outbound when tls config provider returns error",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54569",
						"tls": attrs{
							"mode":       yarpctls.Enforced,
							"spiffe-ids": []string{"test-spiffe"},
						},
					},
				},
			},
			opts:       []Option{OutboundTLSConfigProvider(&fakeOutboundTLSConfigProvider{returnErr: errors.New("test error")})},
			wantErrors: []string{"test error"},
		},
		{
			desc: "fail TLS outbound without outbound tls config provider",
			outboundCfg: attrs{
				"myservice": attrs{
					TransportName: attrs{
						"address": "localhost:54569",
						"tls": attrs{
							"mode":       yarpctls.Enforced,
							"spiffe-ids": []string{"test-spiffe"},
						},
					},
				},
			},
			wantErrors: []string{"outbound TLS enforced but outbound TLS config provider is nil"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			env := make(map[string]string)
			for k, v := range tt.env {
				env[k] = v
			}

			configurator := yarpcconfig.New(yarpcconfig.InterpolationResolver(mapResolver(env)))
			err := configurator.RegisterTransport(TransportSpec(tt.opts...))
			require.NoError(t, err)

			cfgData := make(attrs)
			if tt.transportCfg != nil {
				cfgData["transports"] = attrs{TransportName: tt.transportCfg}
			}
			if tt.inboundCfg != nil {
				cfgData["inbounds"] = attrs{TransportName: tt.inboundCfg}
			}
			if tt.outboundCfg != nil {
				cfgData["outbounds"] = tt.outboundCfg
			}
			cfg, err := configurator.LoadConfig("foo", cfgData)
			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}
			require.NoError(t, err)

			if tt.wantInbound != nil {
				require.Len(t, cfg.Inbounds, 1)
				inbound, ok := cfg.Inbounds[0].(*Inbound)
				require.True(t, ok, "expected *Inbound, got %T", cfg.Inbounds[0])
				assert.Contains(t, inbound.listener.Addr().String(), tt.wantInbound.Address)
				assert.Equal(t, "foo", inbound.t.options.serviceName)

				if tt.wantInbound.ServerMaxRecvMsgSize > 0 {
					assert.Equal(t, tt.wantInbound.ServerMaxRecvMsgSize, inbound.t.options.serverMaxRecvMsgSize)
				} else {
					assert.Equal(t, 1024*1024*64, inbound.t.options.serverMaxRecvMsgSize)
				}
				if tt.wantInbound.ServerMaxSendMsgSize > 0 {
					assert.Equal(t, tt.wantInbound.ServerMaxSendMsgSize, inbound.t.options.serverMaxSendMsgSize)
				} else {
					assert.Equal(t, defaultServerMaxSendMsgSize, inbound.t.options.serverMaxSendMsgSize)
				}
				if tt.wantInbound.ClientMaxRecvMsgSize > 0 {
					assert.Equal(t, tt.wantInbound.ClientMaxRecvMsgSize, inbound.t.options.clientMaxRecvMsgSize)
				} else {
					assert.Equal(t, 1024*1024*64, inbound.t.options.clientMaxRecvMsgSize)
				}
				if tt.wantInbound.ClientMaxSendMsgSize > 0 {
					assert.Equal(t, tt.wantInbound.ClientMaxSendMsgSize, inbound.t.options.clientMaxSendMsgSize)
				} else {
					assert.Equal(t, defaultClientMaxSendMsgSize, inbound.t.options.clientMaxSendMsgSize)
				}
				if tt.wantInbound.ClientMaxHeaderListSize > 0 {
					require.NotNil(t, inbound.t.options.clientMaxHeaderListSize)
					assert.Equal(t, tt.wantInbound.ClientMaxHeaderListSize, *inbound.t.options.clientMaxHeaderListSize)
				} else {
					assert.Nil(t, inbound.t.options.clientMaxHeaderListSize)
				}
				if tt.wantInbound.ServerMaxHeaderListSize > 0 {
					require.NotNil(t, inbound.t.options.serverMaxHeaderListSize)
					assert.Equal(t, tt.wantInbound.ServerMaxHeaderListSize, *inbound.t.options.serverMaxHeaderListSize)
				} else {
					assert.Nil(t, inbound.t.options.serverMaxHeaderListSize)
				}
				assert.Equal(t, tt.wantInbound.TLS, inbound.options.creds != nil)
				assert.Equal(t, tt.wantInbound.TLSMode, inbound.options.tlsMode)
			} else {
				assert.Len(t, cfg.Inbounds, 0)
			}
			for svc, wantOutbound := range tt.wantOutbounds {
				ob, ok := cfg.Outbounds[svc]
				require.True(t, ok, "no outbounds for %s", svc)
				outbound, ok := ob.Unary.(*Outbound)
				require.True(t, ok, "expected *Outbound, got %T", ob)
				if wantOutbound.Address != "" {
					single, ok := outbound.peerChooser.(*peer.Single)
					require.True(t, ok, "expected *peer.Single, got %T", outbound.peerChooser)
					require.NoError(t, single.Start())
					defer single.Stop()
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					peer, _, err := single.Choose(ctx, &transport.Request{})
					require.NoError(t, err)
					require.Equal(t, wantOutbound.Address, peer.Identifier())
					dialer, ok := single.Transport().(*Dialer)
					require.True(t, ok, "expected *Dialer, got %T", single.Transport())
					assert.Equal(t, wantOutbound.TLS, dialer.options.creds != nil)
					assert.Equal(t, wantOutbound.TLSConfig, dialer.options.tlsConfig != nil)
					assert.Equal(t, svc, dialer.options.destServiceName)
					if wantOutbound.WantCustomContextDialer {
						assert.NotNil(t, dialer.options.contextDialer, "expected custom context dialer")
					}

					if wantOutbound.Keepalive != nil {
						require.NotNil(t, dialer.options.keepaliveParams, "expected keepalive parameters")
						assert.Equal(t, wantOutbound.Keepalive, dialer.options.keepaliveParams)
					} else {
						require.Nil(t, dialer.options.keepaliveParams, "unexpected keepalive paramters")
					}
				}
			}
		})
	}
}

func TestContextDialerOptionUsage(t *testing.T) {
	type attrs map[string]interface{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()
	server := grpc.NewServer()
	defer server.Stop()
	go func() {
		require.NoError(t, server.Serve(lis))
	}()

	dialContextInvoked := 0
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		dialContextInvoked++
		return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
	configurator := yarpcconfig.New()
	require.NoError(t, configurator.RegisterTransport(TransportSpec(ContextDialer(dialer))))
	cfgData := attrs{
		"outbounds": attrs{
			"myservice": attrs{
				TransportName: attrs{"address": lis.Addr().String()},
			},
		},
	}
	cfg, err := configurator.LoadConfig("myservice", cfgData)
	require.NoError(t, err)
	outbound, ok := cfg.Outbounds["myservice"].Unary.(*Outbound)
	require.True(t, ok, "expected a gRPC outbound")
	require.NoError(t, outbound.Start())
	defer outbound.Stop()

	peer, _, err := outbound.peerChooser.Choose(ctx, &transport.Request{})
	require.NoError(t, err)
	grpcPeer, ok := peer.(*grpcPeer)
	require.True(t, ok, "expected a gRPC peer")

	for {
		state := grpcPeer.clientConn.GetState()
		if state == connectivity.Ready {
			break
		}
		grpcPeer.clientConn.WaitForStateChange(ctx, state)
	}
	require.Equal(t, connectivity.Ready, grpcPeer.clientConn.GetState(), "expected gRPC connection in Ready state")
	require.Equal(t, 1, dialContextInvoked, "counter should increment by one from dialer invocation")
}

func mapResolver(m map[string]string) func(string) (string, bool) {
	return func(k string) (v string, ok bool) {
		if m != nil {
			v, ok = m[k]
		}
		return
	}
}

type testOption struct{}

func (testOption) grpcOption() {}

type testTransport struct{}

func (testTransport) Start() error    { return nil }
func (testTransport) Stop() error     { return nil }
func (testTransport) IsRunning() bool { return false }
