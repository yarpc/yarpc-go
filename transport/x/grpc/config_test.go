// Copyright (c) 2017 Uber Technologies, Inc.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/yarpcconfig"
)

func TestNewTransportSpecOptions(t *testing.T) {
	transportSpec, err := newTransportSpec(
		BackoffStrategy(nil),
		withInboundUnaryInterceptor(nil),
	)
	require.NoError(t, err)
	require.Equal(t, 1, len(transportSpec.TransportOptions))
	require.Equal(t, 1, len(transportSpec.InboundOptions))
	require.Equal(t, 0, len(transportSpec.OutboundOptions))
}

func TestConfigBuildInboundOtherTransport(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildInbound(&InboundConfig{}, testTransport{}, nil)
	require.Equal(t, newTransportCastError(testTransport{}), err)
}

func TestConfigBuildInboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildInbound(&InboundConfig{}, NewTransport(), nil)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestConfigBuildUnaryOutboundOtherTransport(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildUnaryOutbound(&OutboundConfig{}, testTransport{}, nil)
	require.Equal(t, newTransportCastError(testTransport{}), err)
}

func TestConfigBuildUnaryOutboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildUnaryOutbound(&OutboundConfig{}, NewTransport(), nil)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestTransportSpecUnknownOption(t *testing.T) {
	assert.Panics(t, func() { TransportSpec(testOption{}) })
}

func TestTransportSpec(t *testing.T) {
	type attrs map[string]interface{}

	type wantInbound struct {
		Address string
	}

	type wantOutbound struct {
		Address string
	}

	type test struct {
		desc          string
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
			inboundCfg:  attrs{"address": ":54567"},
			wantInbound: &wantInbound{Address: ":54567"},
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
					transportName: attrs{"address": "localhost:54569"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:54569",
				},
			},
		},
		{
			desc: "outbound interpolation",
			outboundCfg: attrs{
				"myservice": attrs{
					transportName: attrs{"address": "${ADDR}"},
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
					transportName: attrs{"peer": "localhost:54569"},
				},
			},
		},
		{
			desc: "outbound bad peer list",
			outboundCfg: attrs{
				"myservice": attrs{
					transportName: attrs{
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
					transportName: attrs{"with": "derp"},
				},
			},
			wantErrors: []string{
				`failed to configure unary outbound for "myservice":`,
				`no recognized peer chooser preset "derp"`,
			},
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
			if tt.inboundCfg != nil {
				cfgData["inbounds"] = attrs{transportName: tt.inboundCfg}
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
					if !ok {
						require.True(t, ok, "expected *peer.Single, got %T", outbound.peerChooser)
					}
					require.NoError(t, single.Start())
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					peer, _, err := single.Choose(ctx, &transport.Request{})
					require.NoError(t, err)
					require.Equal(t, wantOutbound.Address, peer.Identifier())
				}
			}
		})
	}
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
