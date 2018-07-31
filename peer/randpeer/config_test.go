package randpeer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

type attrs map[string]interface{}

func TestConfig(t *testing.T) {
	cfg := yarpcconfig.New()
	cfg.RegisterPeerList(Spec())
	cfg.RegisterTransport(yarpctest.FakeTransportSpec())
	config, err := cfg.LoadConfig("our-service", attrs{
		"outbounds": attrs{
			"their-service": attrs{
				"fake-transport": attrs{
					"random": attrs{
						"peers": []string{
							"1.1.1.1:1111",
							"2.2.2.2:2222",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, config.Outbounds)
	require.NotNil(t, config.Outbounds["their-service"])
	require.NotNil(t, config.Outbounds["their-service"].Unary)
}
