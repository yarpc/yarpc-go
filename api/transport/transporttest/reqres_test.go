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

package transporttest

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

func TestRequestMatcher(t *testing.T) {
	resMatcher := NewRequestMatcher(t, &transport.Request{
		ID:              "id",
		Host:            "host-name",
		Environment:     "testing",
		Caller:          "caller",
		Service:         "service-name",
		Transport:       "transport",
		Encoding:        "encoding",
		Procedure:       "procedure",
		Headers:         transport.NewHeaders().With("foo", "bar"),
		ShardKey:        "shardkey",
		RoutingKey:      "routingkey",
		RoutingDelegate: "routingdelegate",
		Body:            ioutil.NopCloser(bytes.NewReader([]byte("my body"))),
	})

	t.Run("non-request", func(t *testing.T) {
		require.Panics(t, func() {
			resMatcher.Matches(&transport.Response{})
		})
	})

	t.Run("ID mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID: "wrong id",
		}))
	})

	t.Run("Host mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:   "id",
			Host: "wrong host-name",
		}))
	})

	t.Run("Environment mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "wrong env",
		}))
	})

	t.Run("Caller mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "wrong-caller",
		}))
	})

	t.Run("Service mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "wrong service-name",
		}))
	})

	t.Run("Transport mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "wrong transport",
		}))
	})

	t.Run("Encoding mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "transport",
			Encoding:    "wrong encoding",
		}))
	})

	t.Run("Procedure mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "transport",
			Encoding:    "encoding",
			Procedure:   "wrong procedure",
		}))
	})

	t.Run("Headers mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "transport",
			Encoding:    "encoding",
			Procedure:   "procedure",
			Headers:     transport.NewHeaders().With("foo", "wrong"),
		}))
	})

	t.Run("ShardKey mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "transport",
			Encoding:    "encoding",
			Procedure:   "procedure",
			Headers:     transport.NewHeaders().With("foo", "bar"),
			ShardKey:    "wrong shardkey",
		}))
	})

	t.Run("RoutingKey mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:          "id",
			Host:        "host-name",
			Environment: "testing",
			Caller:      "caller",
			Service:     "service-name",
			Transport:   "transport",
			Encoding:    "encoding",
			Procedure:   "procedure",
			Headers:     transport.NewHeaders().With("foo", "bar"),
			ShardKey:    "shardkey",
			RoutingKey:  "wrong routingkey",
		}))
	})

	t.Run("RoutingDelegate mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:              "id",
			Host:            "host-name",
			Environment:     "testing",
			Caller:          "caller",
			Service:         "service-name",
			Transport:       "transport",
			Encoding:        "encoding",
			Procedure:       "procedure",
			Headers:         transport.NewHeaders().With("foo", "bar"),
			ShardKey:        "shardkey",
			RoutingKey:      "routingkey",
			RoutingDelegate: "wrong routingdelegate",
		}))
	})

	t.Run("Body mismatch", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:              "id",
			Host:            "host-name",
			Environment:     "testing",
			Caller:          "caller",
			Service:         "service-name",
			Transport:       "transport",
			Encoding:        "encoding",
			Procedure:       "procedure",
			Headers:         transport.NewHeaders().With("foo", "bar"),
			ShardKey:        "shardkey",
			RoutingKey:      "routingkey",
			RoutingDelegate: "routingdelegate",
			Body:            ioutil.NopCloser(bytes.NewReader([]byte("wrong body"))),
		}))
	})

	t.Run("match", func(t *testing.T) {
		require.False(t, resMatcher.Matches(&transport.Request{
			ID:              "id",
			Host:            "host-name",
			Environment:     "testing",
			Caller:          "caller",
			Service:         "service-name",
			Transport:       "transport",
			Encoding:        "encoding",
			Procedure:       "procedure",
			Headers:         transport.NewHeaders().With("foo", "bar"),
			ShardKey:        "shardkey",
			RoutingKey:      "routingkey",
			RoutingDelegate: "routingdelegate",
			Body:            ioutil.NopCloser(bytes.NewReader([]byte("body"))),
		}))
	})
}
