package yarpcfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
)

func TestNewClientProvider(t *testing.T) {
	foo := yarpc.Client{Caller: "foo-caller", Service: "foo-service"}

	res, err := NewClientProvider(ClientProviderParams{
		Clients: []yarpc.Client{foo},
	})
	require.NoError(t, err)
	provider := res.Provider

	client, ok := provider.Client("foo-service")
	assert.True(t, ok)
	assert.Equal(t, client.Caller, "foo-caller")
	assert.Equal(t, client.Service, "foo-service")

	_, ok = provider.Client("unknown")
	assert.False(t, ok)
}

func TestNewRouter(t *testing.T) {
	single := yarpc.TransportProcedure{Name: "Hello::HelloWorld"}
	list := []yarpc.TransportProcedure{
		{Name: "Hello::FirstList"},
		{Name: "Hello::SecondList"},
	}

	res, err := NewRouter(RouterParams{
		SingleProcedures: []yarpc.TransportProcedure{single},
		ProcedureLists:   [][]yarpc.TransportProcedure{list},
	})
	require.NoError(t, err)
	assert.Len(t, res.Router.Procedures(), 3)
}
