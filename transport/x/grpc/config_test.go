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
	"testing"

	"go.uber.org/yarpc/x/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransportSpecOptions(t *testing.T) {
	transportSpec, err := newTransportSpec(
		WithInboundTracer(nil),
		WithOutboundTracer(nil),
		WithOutboundTracer(nil),
	)
	require.NoError(t, err)
	require.Equal(t, 1, len(transportSpec.InboundOptions))
	require.Equal(t, 2, len(transportSpec.OutboundOptions))
}

func TestConfigBuildInboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildInbound(&InboundConfig{}, nil, nil)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
}

func TestConfigBuildUnaryOutboundRequiredAddress(t *testing.T) {
	transportSpec := &transportSpec{}
	_, err := transportSpec.buildUnaryOutbound(&OutboundConfig{}, nil, nil)
	require.Equal(t, newRequiredFieldMissingError("address"), err)
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
			inboundCfg:  attrs{"address": ":34567"},
			wantInbound: &wantInbound{Address: ":34567"},
		},
		{
			desc:        "inbound interpolation",
			inboundCfg:  attrs{"address": "${HOST:}:${PORT}"},
			env:         map[string]string{"HOST": "127.0.0.1", "PORT": "34568"},
			wantInbound: &wantInbound{Address: "127.0.0.1:34568"},
		},
		{
			desc: "simple outbound",
			outboundCfg: attrs{
				"myservice": attrs{
					transportName: attrs{"address": "localhost:4040"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "localhost:4040",
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
			env: map[string]string{"ADDR": "127.0.0.1:80"},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					Address: "127.0.0.1:80",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			env := make(map[string]string)
			for k, v := range tt.env {
				env[k] = v
			}

			configurator := config.New(config.InterpolationResolver(mapResolver(env)))
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
				assert.Equal(t, wantOutbound.Address, outbound.address)
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
