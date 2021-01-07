// Copyright (c) 2021 Uber Technologies, Inc.
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

package direct

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpctest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

	t.Run("release error", func(t *testing.T) {
		const addr = "foohost:barport"

		core, observedLogs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)
		giveErr := errors.New("transport retain error")

		trans := yarpctest.NewFakeTransport(
			yarpctest.ReleaseErrors(giveErr, []string{addr}))

		chooser, err := New(Configuration{}, trans, Logger(logger))
		require.NoError(t, err)

		_, onFinish, err := chooser.Choose(context.Background(), &transport.Request{ShardKey: addr})
		require.NoError(t, err)

		onFinish(nil)

		logs := observedLogs.TakeAll()
		require.Len(t, logs, 1, "unexpected number of logs")

		logCtx := logs[0].Context[0]
		assert.Equal(t, "error", logCtx.Key)

		err, ok := logCtx.Interface.(error)
		require.True(t, ok)
		assert.EqualError(t, err, giveErr.Error())
	})

	t.Run("choose", func(t *testing.T) {
		const addr = "foohost:barport"

		chooser, err := New(Configuration{}, yarpctest.NewFakeTransport())
		require.NoError(t, err)

		p, onFinish, err := chooser.Choose(context.Background(), &transport.Request{ShardKey: addr})
		require.NoError(t, err)

		require.NotNil(t, p)
		assert.Equal(t, addr, p.Identifier())

		require.NotNil(t, onFinish)
		onFinish(nil)
	})
}
