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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
		Clients:     []yarpc.Client{foo},
		ClientLists: [][]yarpc.Client{{bar}},
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
		Procedures:     []yarpc.TransportProcedure{single},
		ProcedureLists: [][]yarpc.TransportProcedure{list},
	})
	require.NoError(t, err)
	assert.Len(t, res.Router.Procedures(), 3)
}
