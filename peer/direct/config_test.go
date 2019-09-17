package direct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

type attrs map[string]interface{}

func TestConfig(t *testing.T) {
	const theirService = "their-service"

	cfg := yarpcconfig.New()
	assert.NotPanics(t, func() {
		cfg.MustRegisterPeerChooser(Spec())
		cfg.MustRegisterTransport(yarpctest.FakeTransportSpec())
	})

	config, err := cfg.LoadConfig("our-service", attrs{
		"outbounds": attrs{
			theirService: attrs{
				"fake-transport": attrs{
					name: attrs{},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, config.Outbounds)
	require.NotNil(t, config.Outbounds[theirService])
	require.NotNil(t, config.Outbounds[theirService].Unary)
}
