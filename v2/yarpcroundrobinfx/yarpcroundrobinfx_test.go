package yarpcroundrobinfx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/config"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcdialer"
	"go.uber.org/yarpc/v2/yarpctest"
)

func newDialerProvider(t *testing.T) yarpc.DialerProvider {
	p := yarpcdialer.NewProvider()
	http := yarpctest.NewFakeDialer("http")
	require.NoError(t, p.Register("http", http))
	return p
}

func TestNewConfig(t *testing.T) {
	cfg := strings.NewReader("yarpc: {peers: {roundrobin: {bar: {dialer: http, capacity: 100}}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewConfig(ConfigParams{
		Provider: provider,
	})
	require.NoError(t, err)
	assert.Equal(t,
		Config{
			Clients: map[string]RoundRobinConfig{
				"bar": {Dialer: "http", Capacity: 100},
			},
		},
		res.Config)
}

func TestNewList(t *testing.T) {
	t.Run("unknown dialer", func(t *testing.T) {
		_, err := NewList(ListParams{
			Config: Config{
				Clients: map[string]RoundRobinConfig{
					"bar": {Dialer: "dne", Capacity: 100},
				},
			},
			Provider: newDialerProvider(t),
		})
		assert.EqualError(t, err, `failed to resolve dialer "dne"`)
	})

	t.Run("successfully create chooser and list", func(t *testing.T) {
		res, err := NewList(ListParams{
			Config: Config{
				Clients: map[string]RoundRobinConfig{
					"bar": {Dialer: "http", Capacity: 100},
				},
			},
			Provider: newDialerProvider(t),
		})
		require.NoError(t, err)

		require.Len(t, res.Choosers, 1)
		assert.Equal(t, "bar", res.Choosers[0].Name())
		require.Len(t, res.Lists, 1)
		assert.Equal(t, "bar", res.Lists[0].Name())
	})
}
