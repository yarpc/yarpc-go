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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestNewClientProvider(t *testing.T) {
	t.Run("duplicate registration error", func(t *testing.T) {
		foo := yarpc.Client{Name: "foo"}
		_, err := NewClientProvider(ClientProviderParams{
			Clients: []yarpc.Client{foo, foo},
		})
		assert.EqualError(t, err, `client "foo" was registered more than once`)
	})
	t.Run("multiple clients", func(t *testing.T) {
		foo := yarpc.Client{Name: "foo", Caller: "foo-caller", Service: "foo-service"}
		bar := yarpc.Client{Name: "bar", Caller: "bar-caller", Service: "bar-service"}

		res, err := NewClientProvider(ClientProviderParams{
			Clients:     []yarpc.Client{foo},
			ClientLists: [][]yarpc.Client{{bar}},
		})
		require.NoError(t, err)
		provider := res.Provider

		client, ok := provider.Client("foo")
		assert.True(t, ok)
		assert.Equal(t, client.Caller, "foo-caller")
		assert.Equal(t, client.Service, "foo-service")

		client, ok = provider.Client("bar")
		assert.True(t, ok)
		assert.Equal(t, client.Caller, "bar-caller")
		assert.Equal(t, client.Service, "bar-service")

		_, ok = provider.Client("unknown")
		assert.False(t, ok)
	})
}

func TestClientHasMiddleware(t *testing.T) {
	var gotCallOder []string

	var middleware = yarpc.UnaryOutboundTransportMiddlewareFunc(
		func(ctx context.Context, _ *yarpc.Request, _ *yarpc.Buffer, o yarpc.UnaryOutbound) (*yarpc.Response, *yarpc.Buffer, error) {
			gotCallOder = append(gotCallOder, "middleware")
			return o.Call(ctx, nil, nil)
		})

	out := yarpctest.OutboundCallable(func(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
		gotCallOder = append(gotCallOder, "outbound")
		return nil, nil, nil
	})
	trans := &yarpctest.FakeTransport{}

	client := yarpc.Client{
		Name:  "client",
		Unary: trans.NewOutbound(nil, yarpctest.OutboundCallOverride(out)),
	}

	result, err := NewClientProvider(ClientProviderParams{
		UnaryOutboundTransportMiddleware: []yarpc.UnaryOutboundTransportMiddleware{middleware},
		Clients: []yarpc.Client{client},
	})

	require.NoError(t, err)

	client, ok := result.Provider.Client("client")
	require.True(t, ok, "could not find client")

	_, _, _ = client.Unary.Call(context.Background(), nil, nil)

	wantCallOrder := []string{"middleware", "outbound"}
	assert.Equal(t, wantCallOrder, gotCallOder)
}

func TestNewDialerProvider(t *testing.T) {
	t.Run("duplicate registration error", func(t *testing.T) {
		foo := yarpctest.NewFakeDialer("foo")
		_, err := NewDialerProvider(DialerProviderParams{
			Dialers: []yarpc.Dialer{foo, foo},
		})
		assert.EqualError(t, err, `dialer "foo" was registered more than once`)
	})
	t.Run("multiple dialers", func(t *testing.T) {
		foo := yarpctest.NewFakeDialer("foo")
		bar := yarpctest.NewFakeDialer("bar")

		res, err := NewDialerProvider(DialerProviderParams{
			Dialers:     []yarpc.Dialer{foo},
			DialerLists: [][]yarpc.Dialer{{bar}},
		})
		require.NoError(t, err)
		provider := res.Provider

		_, ok := provider.Dialer("foo")
		assert.True(t, ok)

		_, ok = provider.Dialer("bar")
		assert.True(t, ok)

		_, ok = provider.Dialer("unknown")
		assert.False(t, ok)
	})
}

func TestNewChooserProvider(t *testing.T) {
	t.Run("duplicate registration error", func(t *testing.T) {
		foo := yarpctest.NewFakePeerChooser("foo")
		_, err := NewChooserProvider(ChooserProviderParams{
			Choosers: []yarpc.Chooser{foo, foo},
		})
		assert.EqualError(t, err, `chooser "foo" was registered more than once`)
	})
	t.Run("multiple choosers", func(t *testing.T) {
		foo := yarpctest.NewFakePeerChooser("foo")
		bar := yarpctest.NewFakePeerChooser("bar")

		res, err := NewChooserProvider(ChooserProviderParams{
			Choosers:     []yarpc.Chooser{foo},
			ChooserLists: [][]yarpc.Chooser{{bar}},
		})
		require.NoError(t, err)
		provider := res.Provider

		_, ok := provider.Chooser("foo")
		assert.True(t, ok)

		_, ok = provider.Chooser("bar")
		assert.True(t, ok)

		_, ok = provider.Chooser("unknown")
		assert.False(t, ok)
	})
}

func TestNewListProvider(t *testing.T) {
	t.Run("duplicate registration error", func(t *testing.T) {
		foo := yarpctest.NewFakePeerList("foo")
		_, err := NewListProvider(ListProviderParams{
			Lists: []yarpc.List{foo, foo},
		})
		assert.EqualError(t, err, `list "foo" was registered more than once`)
	})
	t.Run("multiple lists", func(t *testing.T) {
		foo := yarpctest.NewFakePeerList("foo")
		bar := yarpctest.NewFakePeerList("bar")

		res, err := NewListProvider(ListProviderParams{
			Lists:     []yarpc.List{foo},
			ListLists: [][]yarpc.List{{bar}},
		})
		require.NoError(t, err)
		provider := res.Provider

		_, ok := provider.List("foo")
		assert.True(t, ok)

		_, ok = provider.List("bar")
		assert.True(t, ok)

		_, ok = provider.List("unknown")
		assert.False(t, ok)
	})
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
