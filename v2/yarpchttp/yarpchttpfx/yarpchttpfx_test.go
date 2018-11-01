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

func TestStartInbounds(t *testing.T) {
	cfg := strings.NewReader("yarpchttp: {inbounds: {address: http://127.0.0.1:0}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	assert.NoError(t, StartInbounds(StartInboundsParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Router:    yarpctest.NewFakeRouter(nil),
		Provider:  provider,
	}))
}

func TestNewClients(t *testing.T) {
	cfg := strings.NewReader("yarpchttp: {outbounds: {bar: {address: http://127.0.0.1:0}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewClients(ClientParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Provider:  provider,
	})
	require.NoError(t, err)

	assert.Len(t, res.Clients, 1)
	assert.Equal(t, res.Clients[0].Caller, "foo")
}
