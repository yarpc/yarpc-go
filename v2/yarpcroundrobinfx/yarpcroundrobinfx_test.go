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
	p, err := yarpcdialer.NewProvider(yarpctest.NewFakeDialer("http"))
	require.NoError(t, err)
	return p
}

func TestNewConfig(t *testing.T) {
	cfg := strings.NewReader("yarpc: {choosers: {roundrobin: {bar: {dialer: http, capacity: 100}}}}")
	provider, err := config.NewYAML(config.Source(cfg))
	require.NoError(t, err)

	res, err := NewConfig(ConfigParams{
		Provider: provider,
	})
	require.NoError(t, err)
	assert.Equal(t,
		Config{
			Choosers: map[string]RoundRobinConfig{
				"bar": {Dialer: "http", Capacity: 100},
			},
		},
		res.Config)
}

func TestNewList(t *testing.T) {
	t.Run("unknown dialer", func(t *testing.T) {
		_, err := NewList(ListParams{
			Config: Config{
				Choosers: map[string]RoundRobinConfig{
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
				Choosers: map[string]RoundRobinConfig{
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
