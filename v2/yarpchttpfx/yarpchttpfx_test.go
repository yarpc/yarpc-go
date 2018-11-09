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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package yarpchttpfx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/config"
	"go.uber.org/fx/fxtest"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcchooser"
	"go.uber.org/yarpc/v2/yarpcdialer"
	"go.uber.org/yarpc/v2/yarpctest"
)

func newDialerProvider(t *testing.T) yarpc.DialerProvider {
	p, err := yarpcdialer.NewProvider(yarpctest.NewFakeDialer("http"))
	require.NoError(t, err)
	return p
}

func newChooserProvider(t *testing.T) yarpc.ChooserProvider {
	p, err := yarpcchooser.NewProvider(yarpctest.NewFakePeerChooser("roundrobin"))
	require.NoError(t, err)
	return p
}

func TestNewInboundConfig(t *testing.T) {
	cfg := strings.NewReader("yarpc: {http: {inbounds: {address: http://127.0.0.1:0}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewInboundConfig(InboundConfigParams{
		Provider: provider,
	})
	require.NoError(t, err)
	assert.Equal(t, InboundConfig{Address: "http://127.0.0.1:0"}, res.Config)
}

func TestStartInbounds(t *testing.T) {
	assert.NoError(t, StartInbounds(StartInboundsParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Router:    yarpctest.NewFakeRouter(nil),
		Config:    InboundConfig{Address: "http://127.0.0.1:0"},
	}))
}

func TestNewOutboundsConfig(t *testing.T) {
	cfg := strings.NewReader("yarpc: {http: {outbounds: {bar: {address: http://127.0.0.1:0}}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewOutboundsConfig(OutboundsConfigParams{
		Provider: provider,
	})
	require.NoError(t, err)
	assert.Equal(t,
		OutboundsConfig{
			Outbounds: map[string]OutboundConfig{
				"bar": {Address: "http://127.0.0.1:0"},
			},
		},
		res.Config,
	)
}

func TestNewClients(t *testing.T) {
	tests := []struct {
		desc        string
		giveCfg     OutboundConfig
		wantCaller  string
		wantName    string
		wantService string
		wantErr     string
	}{
		{
			desc:        "policy successfully configured",
			giveCfg:     OutboundConfig{Policy: "roundrobin"},
			wantCaller:  "foo",
			wantName:    "bar",
			wantService: "bar",
		},
		{
			desc:    "policy does not exist",
			giveCfg: OutboundConfig{Policy: "dne"},
			wantErr: `failed to resolve outbound peer list policy: "dne"`,
		},
		{
			desc:        "address successfully configured",
			giveCfg:     OutboundConfig{Address: "http://127.0.0.1:0"},
			wantCaller:  "foo",
			wantName:    "bar",
			wantService: "bar",
		},
		{
			desc:    "address failed to parse",
			giveCfg: OutboundConfig{Address: "127:0"},
			wantErr: "parse 127:0: first path segment in URL cannot contain colon",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := NewClients(ClientParams{
				Lifecycle: fxtest.NewLifecycle(t),
				Config: OutboundsConfig{
					Outbounds: map[string]OutboundConfig{
						"bar": tt.giveCfg,
					},
				},
				DialerProvider:  newDialerProvider(t),
				ChooserProvider: newChooserProvider(t),
			})
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, res.Clients, 1)

			client := res.Clients[0]
			assert.Equal(t, client.Caller, tt.wantCaller)
			assert.Equal(t, client.Name, tt.wantName)
			assert.Equal(t, client.Service, tt.wantService)
		})
	}
}
