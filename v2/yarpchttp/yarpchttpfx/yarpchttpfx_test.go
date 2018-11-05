package yarpchttpfx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/config"
	"go.uber.org/fx/fxtest"
	"go.uber.org/yarpc/v2/yarpctest"
)

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
	cfg := strings.NewReader("yarpc: {http: {outbounds: {bar: {address: http://127.0.0.1:0, service: baz}}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewOutboundsConfig(OutboundsConfigParams{
		Provider: provider,
	})
	require.NoError(t, err)
	assert.Equal(t,
		OutboundsConfig{
			Clients: map[string]OutboundConfig{
				"bar": {Address: "http://127.0.0.1:0", Service: "baz"},
			},
		},
		res.Config,
	)
}

func TestNewClients(t *testing.T) {
	res, err := NewClients(ClientParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Config: OutboundsConfig{
			Clients: map[string]OutboundConfig{
				"bar": {Address: "http://127.0.0.1:0"},
			},
		},
	})
	require.NoError(t, err)
	assert.Len(t, res.Clients, 1)
	assert.Equal(t, res.Clients[0].Caller, "foo")
}
