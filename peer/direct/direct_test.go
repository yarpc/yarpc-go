package direct

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpctest"
)

func TestDirect(t *testing.T) {
	t.Run("nil transport", func(t *testing.T) {
		_, err := New(Configuration{}, nil)
		assert.Error(t, err)
	})

	t.Run("chooser interface", func(t *testing.T) {
		chooser, err := New(Configuration{}, yarpctest.NewFakeTransport())
		require.NoError(t, err)

		assert.NoError(t, chooser.Start())
		assert.True(t, chooser.IsRunning())
		assert.NoError(t, chooser.Stop())
	})

	t.Run("missing shard key", func(t *testing.T) {
		chooser, err := New(Configuration{}, yarpctest.NewFakeTransport())
		require.NoError(t, err)
		_, _, err = chooser.Choose(context.Background(), &transport.Request{})
		assert.Error(t, err)
	})

	t.Run("retain error", func(t *testing.T) {
		const addr = "foohost:barport"
		giveErr := errors.New("transport retain error")

		trans := yarpctest.NewFakeTransport(
			yarpctest.RetainErrors(giveErr, []string{addr}))

		chooser, err := New(Configuration{}, trans)
		require.NoError(t, err)

		_, _, err = chooser.Choose(context.Background(), &transport.Request{ShardKey: addr})
		assert.EqualError(t, err, giveErr.Error())
	})

	t.Run("choose sucess", func(t *testing.T) {
		const addr = "foohost:barport"

		chooser, err := New(Configuration{}, yarpctest.NewFakeTransport())
		require.NoError(t, err)

		p, onFinish, err := chooser.Choose(context.Background(), &transport.Request{ShardKey: addr})
		require.NoError(t, err)

		require.NotNil(t, onFinish)
		onFinish(nil)

		require.NotNil(t, p)
		assert.Equal(t, addr, p.Identifier())
	})
}
