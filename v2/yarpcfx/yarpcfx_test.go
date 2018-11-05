package yarpcfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
)

func TestNewClientProvider(t *testing.T) {
	foo := yarpc.Client{Caller: "foo-caller", Service: "foo-service"}
	bar := yarpc.Client{Caller: "bar-caller", Service: "bar-service"}

	res, err := NewClientProvider(ClientProviderParams{
		SingleClients: []yarpc.Client{foo},
		ClientLists:   [][]yarpc.Client{{bar}},
	})
	require.NoError(t, err)
	provider := res.Provider

	client, ok := provider.Client("foo-service")
	assert.True(t, ok)
	assert.Equal(t, client.Caller, "foo-caller")
	assert.Equal(t, client.Service, "foo-service")

	client, ok = provider.Client("bar-service")
	assert.True(t, ok)
	assert.Equal(t, client.Caller, "bar-caller")
	assert.Equal(t, client.Service, "bar-service")

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
