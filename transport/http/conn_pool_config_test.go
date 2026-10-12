// Copyright (c) 2026 Uber Technologies, Inc.
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

package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/yarpc/yarpcconfig"
)

func ptr[T any](v T) *T { return &v }

// TestDefaultPoolConfigMatchesGRPC pins the defaults to the values the gRPC
// transport ships (transport/grpc/options.go: defaultClientConnPool*), so the
// two transports scale the same way out of the box.
func TestDefaultPoolConfigMatchesGRPC(t *testing.T) {
	assert.Equal(t, connpool.Config{
		DynamicScalingEnabled:  true,
		MaxConcurrentStreams:   100,
		ScaleUpThreshold:       0.7,
		ScaleDownGap:           0.1,
		MinConnections:         1,
		MaxConnections:         50,
		IdleTimeout:            5 * time.Minute,
		ScalingMonitorInterval: 30 * time.Second,
	}, NewTransport().h2PoolConfig)
	assert.NoError(t, defaultH2PoolConfig.Validate())
}

func TestPoolTransportOptions(t *testing.T) {
	tr := NewTransport(
		WithDynamicConnectionScaling(false),
		MaxConcurrentStreams(250),
		ScaleUpThreshold(0.9),
		ScaleDownGap(0.2),
		MinConnections(2),
		MaxConnections(8),
		ConnIdleTimeout(time.Minute),
		ScalingMonitorInterval(45*time.Second),
	)
	assert.Equal(t, connpool.Config{
		DynamicScalingEnabled:  false,
		MaxConcurrentStreams:   250,
		ScaleUpThreshold:       0.9,
		ScaleDownGap:           0.2,
		MinConnections:         2,
		MaxConnections:         8,
		IdleTimeout:            time.Minute,
		ScalingMonitorInterval: 45 * time.Second,
	}, tr.h2PoolConfig)
}

func TestValidateClientConnectionPoolConfig(t *testing.T) {
	require.NoError(t, validateClientConnectionPoolConfig(ClientConnectionPoolConfig{}), "an empty block inherits everything")
	require.NoError(t, validateClientConnectionPoolConfig(ClientConnectionPoolConfig{
		DynamicScalingEnabled: ptr(false), MaxConcurrentStreams: 1, ScaleUpThreshold: 1,
		ScaleDownGap: 0.5, MinConnections: 3, MaxConnections: 3, IdleTimeout: 1, ScalingMonitorInterval: 1,
	}))

	tests := []struct {
		name    string
		cfg     ClientConnectionPoolConfig
		wantErr string
	}{
		{"negative min", ClientConnectionPoolConfig{MinConnections: -1}, "clientConnectionPool.minConnections must be non-negative, got -1"},
		{"negative max", ClientConnectionPoolConfig{MaxConnections: -1}, "clientConnectionPool.maxConnections must be non-negative, got -1"},
		{"max below min", ClientConnectionPoolConfig{MinConnections: 5, MaxConnections: 2}, "clientConnectionPool.maxConnections (2) must be >= minConnections (5)"},
		{"negative streams", ClientConnectionPoolConfig{MaxConcurrentStreams: -1}, "clientConnectionPool.maxConcurrentStreams must be non-negative, got -1"},
		{"threshold above one", ClientConnectionPoolConfig{ScaleUpThreshold: 1.5}, "clientConnectionPool.scaleUpThreshold must be in [0, 1], got 1.5"},
		{"gap of one", ClientConnectionPoolConfig{ScaleDownGap: 1}, "clientConnectionPool.scaleDownGap must be in [0, 1), got 1"},
		{"negative idle timeout", ClientConnectionPoolConfig{IdleTimeout: -time.Second}, "clientConnectionPool.idleTimeout must be non-negative, got -1s"},
		{"negative interval", ClientConnectionPoolConfig{ScalingMonitorInterval: -time.Second}, "clientConnectionPool.scalingMonitorInterval must be non-negative, got -1s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClientConnectionPoolConfig(tt.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestDialOptionsResolvedPoolConfig(t *testing.T) {
	base := defaultH2PoolConfig

	assert.Equal(t, base, newDialOptions(nil).resolvedPoolConfig(base), "no override: the transport config as-is")

	got := newDialOptions([]DialOption{OutboundConnectionPool(ClientConnectionPoolConfig{
		MinConnections: 20,
		MaxConnections: 30,
	})}).resolvedPoolConfig(base)
	want := base
	want.MinConnections, want.MaxConnections = 20, 30
	assert.Equal(t, want, got, "unset override fields inherit from the transport config")
}

func TestRetainPeerValidatesResolvedPoolConfig(t *testing.T) {
	t.Run("an override that conflicts with the transport config is rejected", func(t *testing.T) {
		tr := NewTransport() // MaxConnections defaults to 50
		require.NoError(t, tr.Start())
		defer func() { assert.NoError(t, tr.Stop()) }()

		// Valid on its own (no maxConnections set), invalid once merged.
		d := tr.NewDialer(OutboundConnectionPool(ClientConnectionPoolConfig{MinConnections: 99}))
		_, err := d.RetainPeer(testIdentifier{"127.0.0.1:1"}, idSubscriber{1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxConnections (50) must be >= minConnections (99)")
		assert.Empty(t, tr.peers, "a rejected peer must not be registered")
	})

	t.Run("an invalid transport-wide config is rejected when a peer is retained", func(t *testing.T) {
		tr := NewTransport(MinConnections(10), MaxConnections(2))
		require.NoError(t, tr.Start())
		defer func() { assert.NoError(t, tr.Stop()) }()

		_, err := tr.RetainPeer(testIdentifier{"127.0.0.1:1"}, idSubscriber{1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxConnections (2) must be >= minConnections (10)")
		assert.Empty(t, tr.peers)
	})

	t.Run("a valid override is applied to the retained peer", func(t *testing.T) {
		tr := NewTransport()
		require.NoError(t, tr.Start())
		defer func() { assert.NoError(t, tr.Stop()) }()

		d := tr.NewDialer(OutboundConnectionPool(ClientConnectionPoolConfig{MinConnections: 4})).WithConnectionIsolation()
		pi, err := d.RetainPeer(testIdentifier{"127.0.0.1:1"}, idSubscriber{1})
		require.NoError(t, err)
		startup := pi.(*httpPeer).poolConfig.Startup()
		assert.Equal(t, 4, startup.MinConnections)
		assert.Equal(t, 50, startup.MaxConnections, "unset fields inherit")
	})
}

func TestIsolationCopiesDialOptions(t *testing.T) {
	tr := NewTransport()
	base := tr.NewDialer(OutboundConnectionPool(ClientConnectionPoolConfig{MinConnections: 4}))
	isolated := base.WithConnectionIsolation()

	assert.NotSame(t, base.options, isolated.options, "an isolated dialer owns its options")
	assert.Equal(t, 4, isolated.options.connPoolOverride.MinConnections, "and keeps the override")
}

func TestLiveProviderPrecedence(t *testing.T) {
	global := func() ClientConnectionPoolConfig { return ClientConnectionPoolConfig{MaxConnections: 11} }
	outbound := func() ClientConnectionPoolConfig { return ClientConnectionPoolConfig{MaxConnections: 22} }

	resolve := func(t *testing.T, tr *Transport, opts ...DialOption) *httpPeer {
		t.Helper()
		require.NoError(t, tr.Start())
		t.Cleanup(func() { assert.NoError(t, tr.Stop()) })
		pi, err := tr.NewDialer(opts...).WithConnectionIsolation().RetainPeer(testIdentifier{"127.0.0.1:1"}, idSubscriber{1})
		require.NoError(t, err)
		return pi.(*httpPeer)
	}

	t.Run("no provider", func(t *testing.T) {
		p := resolve(t, NewTransport())
		assert.False(t, p.poolConfig.HasProvider())
		assert.Equal(t, 50, p.poolConfig.Get().MaxConnections)
	})

	t.Run("global provider applies to every peer", func(t *testing.T) {
		p := resolve(t, NewTransport(WithGlobalLiveConnectionPoolProvider(global)))
		assert.True(t, p.poolConfig.HasProvider())
		assert.Equal(t, 11, p.poolConfig.Get().MaxConnections)
	})

	t.Run("the outbound provider replaces the global one", func(t *testing.T) {
		p := resolve(t, NewTransport(WithGlobalLiveConnectionPoolProvider(global)), outboundLiveProvider(outbound))
		assert.Equal(t, 22, p.poolConfig.Get().MaxConnections)
	})

	t.Run("a live overlay sits on top of the startup override", func(t *testing.T) {
		p := resolve(t, NewTransport(WithGlobalLiveConnectionPoolProvider(global)),
			OutboundConnectionPool(ClientConnectionPoolConfig{MinConnections: 3}))
		got := p.poolConfig.Get()
		assert.Equal(t, 3, got.MinConnections, "startup override survives where the live hook has no opinion")
		assert.Equal(t, 11, got.MaxConnections)
	})

	t.Run("an invalid live overlay falls back to the startup config", func(t *testing.T) {
		bad := func() ClientConnectionPoolConfig { return ClientConnectionPoolConfig{MinConnections: 99} }
		p := resolve(t, NewTransport(WithGlobalLiveConnectionPoolProvider(bad)))
		assert.Equal(t, defaultH2PoolConfig, p.poolConfig.Get())
	})
}

// loadConfig builds the YAML in cfg with TransportSpec(opts...), the way a
// service's dispatcher would.
func loadConfig(t *testing.T, cfg map[string]interface{}, opts ...Option) (yarpc.Config, error) {
	t.Helper()
	configurator := yarpcconfig.New()
	require.NoError(t, configurator.RegisterTransport(TransportSpec(opts...)))
	require.NoError(t, configurator.RegisterPeerList(roundrobin.Spec()))
	return configurator.LoadConfig("foo", cfg)
}

type m = map[string]interface{}

func singleOutbound(extra m) m {
	http := m{"url": "http://127.0.0.1:4040/yarpc"}
	for k, v := range extra {
		http[k] = v
	}
	return m{"svc": m{"http": http}}
}

func outboundOf(t *testing.T, cfg yarpc.Config, name string) *Outbound {
	t.Helper()
	ob, ok := cfg.Outbounds[name].Unary.(*Outbound)
	require.True(t, ok, "expected *Outbound, got %T", cfg.Outbounds[name].Unary)
	return ob
}

func TestTransportConfigClientConnectionPool(t *testing.T) {
	t.Run("YAML values are applied over the defaults", func(t *testing.T) {
		cfg, err := loadConfig(t, m{
			"transports": m{"http": m{"clientConnectionPool": m{
				"dynamicScalingEnabled":  false,
				"maxConcurrentStreams":   250,
				"scaleUpThreshold":       0.9,
				"scaleDownGap":           0.2,
				"minConnections":         2,
				"maxConnections":         8,
				"idleTimeout":            "1m",
				"scalingMonitorInterval": "45s",
			}}},
			"outbounds": singleOutbound(nil),
		})
		require.NoError(t, err)
		assert.Equal(t, connpool.Config{
			DynamicScalingEnabled:  false,
			MaxConcurrentStreams:   250,
			ScaleUpThreshold:       0.9,
			ScaleDownGap:           0.2,
			MinConnections:         2,
			MaxConnections:         8,
			IdleTimeout:            time.Minute,
			ScalingMonitorInterval: 45 * time.Second,
		}, outboundOf(t, cfg, "svc").transport.h2PoolConfig)
	})

	t.Run("omitted fields keep programmatic options and defaults", func(t *testing.T) {
		cfg, err := loadConfig(t, m{
			"transports": m{"http": m{"clientConnectionPool": m{"minConnections": 2}}},
			"outbounds":  singleOutbound(nil),
		}, MaxConnections(9), WithDynamicConnectionScaling(false))
		require.NoError(t, err)
		got := outboundOf(t, cfg, "svc").transport.h2PoolConfig
		assert.Equal(t, 2, got.MinConnections, "YAML wins where set")
		assert.Equal(t, 9, got.MaxConnections, "the programmatic option survives where YAML is silent")
		assert.False(t, got.DynamicScalingEnabled, "an omitted dynamicScalingEnabled does not turn scaling back on")
		assert.EqualValues(t, 100, got.MaxConcurrentStreams, "and unset values stay at the default")
	})

	t.Run("dynamicScalingEnabled: true in YAML overrides the programmatic option", func(t *testing.T) {
		cfg, err := loadConfig(t, m{
			"transports": m{"http": m{"clientConnectionPool": m{"dynamicScalingEnabled": true}}},
			"outbounds":  singleOutbound(nil),
		}, WithDynamicConnectionScaling(false))
		require.NoError(t, err)
		assert.True(t, outboundOf(t, cfg, "svc").transport.h2PoolConfig.DynamicScalingEnabled)
	})

	invalid := []struct {
		name    string
		pool    m
		wantErr string
	}{
		{"threshold out of range", m{"scaleUpThreshold": 1.5}, "clientConnectionPool.scaleUpThreshold must be in [0, 1], got 1.5"},
		{"negative min", m{"minConnections": -1}, "clientConnectionPool.minConnections must be non-negative, got -1"},
		{"min above max", m{"minConnections": 60}, "clientConnectionPool.maxConnections (50) must be >= minConnections (60)"},
		{"gap swallows threshold", m{"scaleDownGap": 0.7}, "scaleUpThreshold (0.70) minus scaleDownGap (0.70) must be > 0"},
	}
	for _, tt := range invalid {
		t.Run("invalid: "+tt.name, func(t *testing.T) {
			_, err := loadConfig(t, m{
				"transports": m{"http": m{"clientConnectionPool": tt.pool}},
				"outbounds":  singleOutbound(nil),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestOutboundConfigClientConnectionPool(t *testing.T) {
	t.Run("YAML override applies to the outbound's peer and inherits the rest", func(t *testing.T) {
		cfg, err := loadConfig(t, m{
			"transports": m{"http": m{"clientConnectionPool": m{"maxConnections": 10}}},
			"outbounds": singleOutbound(m{"clientConnectionPool": m{
				"minConnections": 3,
				"idleTimeout":    "2m",
			}}),
		})
		require.NoError(t, err)

		ob := outboundOf(t, cfg, "svc")
		tr := ob.transport
		require.NoError(t, tr.Start())
		require.NoError(t, ob.Start())
		defer func() {
			assert.NoError(t, ob.Stop())
			assert.NoError(t, tr.Stop())
		}()

		require.Len(t, tr.peers, 1)
		for _, p := range tr.peers {
			got := p.poolConfig.Startup()
			assert.Equal(t, 3, got.MinConnections)
			assert.Equal(t, 10, got.MaxConnections, "inherited from the transport block")
			assert.Equal(t, 2*time.Minute, got.IdleTimeout)
			assert.Equal(t, 0.7, got.ScaleUpThreshold, "inherited default")
		}
		assert.Equal(t, 1, tr.h2PoolConfig.MinConnections, "the transport-wide config is untouched by the outbound override")
		assert.Equal(t, 10, tr.h2PoolConfig.MaxConnections)
	})

	t.Run("invalid override values are rejected", func(t *testing.T) {
		_, err := loadConfig(t, m{"outbounds": singleOutbound(m{"clientConnectionPool": m{"scaleDownGap": 1.5}})})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clientConnectionPool.scaleDownGap must be in [0, 1), got 1.5")
	})

	t.Run("an override that conflicts with the transport config is rejected at build time", func(t *testing.T) {
		_, err := loadConfig(t, m{"outbounds": singleOutbound(m{"clientConnectionPool": m{"minConnections": 99}})})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxConnections (50) must be >= minConnections (99)")
	})

	t.Run("outbounds to the same address share a peer unless isolation is requested", func(t *testing.T) {
		urlOnly := func(extra m) m {
			h := m{"url": "http://127.0.0.1:4040/yarpc"}
			for k, v := range extra {
				h[k] = v
			}
			return m{"http": h}
		}
		tests := []struct {
			desc      string
			outbounds m
			opts      []Option
			wantPeers int
		}{
			{
				desc:      "default is shared, as before",
				outbounds: m{"a": urlOnly(nil), "b": urlOnly(nil)},
				wantPeers: 1,
			},
			{
				desc:      "IsolateConnectionsPerOutbound isolates every outbound",
				outbounds: m{"a": urlOnly(nil), "b": urlOnly(nil)},
				opts:      []Option{IsolateConnectionsPerOutbound(true)},
				wantPeers: 2,
			},
			{
				desc: "an outbound with its own pool override is isolated",
				outbounds: m{
					"a": urlOnly(m{"clientConnectionPool": m{"minConnections": 2}}),
					"b": urlOnly(nil),
				},
				wantPeers: 2,
			},
		}
		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				cfg, err := loadConfig(t, m{"outbounds": tt.outbounds}, tt.opts...)
				require.NoError(t, err)

				a, b := outboundOf(t, cfg, "a"), outboundOf(t, cfg, "b")
				require.Same(t, a.transport, b.transport)
				tr := a.transport
				require.NoError(t, tr.Start())
				require.NoError(t, a.Start())
				require.NoError(t, b.Start())
				defer func() {
					assert.NoError(t, a.Stop())
					assert.NoError(t, b.Stop())
					assert.NoError(t, tr.Stop())
				}()

				assert.Len(t, tr.peers, tt.wantPeers)
			})
		}
	})

	t.Run("peer-list outbounds are isolated and use the override too", func(t *testing.T) {
		cfg, err := loadConfig(t, m{"outbounds": m{"svc": m{"http": m{
			"url":                  "http://x/yarpc",
			"clientConnectionPool": m{"minConnections": 6},
			"round-robin":          m{"peers": []string{"127.0.0.1:4040", "127.0.0.1:4041"}},
		}}}})
		require.NoError(t, err)

		ob := outboundOf(t, cfg, "svc")
		tr := ob.transport
		require.NoError(t, tr.Start())
		require.NoError(t, ob.Start())
		defer func() {
			assert.NoError(t, ob.Stop())
			assert.NoError(t, tr.Stop())
		}()

		require.Len(t, tr.peers, 2)
		for key, p := range tr.peers {
			assert.NotNil(t, key.connectionScope, "peers of a YAML outbound are scoped to it")
			assert.Equal(t, 6, p.poolConfig.Startup().MinConnections)
		}
	})
}

func TestOutboundLiveProviderFactory(t *testing.T) {
	type call struct {
		outbound string
		yaml     *ClientConnectionPoolConfig
	}
	var calls []call
	factory := func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider {
		calls = append(calls, call{outbound, yaml})
		if outbound != "with-hook" {
			return nil // keep the global hook
		}
		return func() ClientConnectionPoolConfig { return ClientConnectionPoolConfig{MaxConnections: 22} }
	}
	global := func() ClientConnectionPoolConfig { return ClientConnectionPoolConfig{MaxConnections: 11} }

	cfg, err := loadConfig(t, m{"outbounds": m{
		"with-hook": m{"http": m{
			"url":                  "http://127.0.0.1:4040/yarpc",
			"clientConnectionPool": m{"minConnections": 2},
		}},
		"without-hook": m{"http": m{"url": "http://127.0.0.1:4041/yarpc"}},
	}}, WithOutboundLiveConnectionPoolProvider(factory), WithGlobalLiveConnectionPoolProvider(global))
	require.NoError(t, err)

	// yarpcconfig builds a unary and a oneway Outbound for each http entry,
	// and each build calls the factory.
	require.Len(t, calls, 4, "the factory is called once per built outbound")
	byName := map[string]*ClientConnectionPoolConfig{}
	counts := map[string]int{}
	for _, c := range calls {
		byName[c.outbound] = c.yaml
		counts[c.outbound]++
	}
	assert.Equal(t, map[string]int{"with-hook": 2, "without-hook": 2}, counts)
	require.Contains(t, byName, "with-hook", "the factory receives Kit.OutboundKey()")
	require.NotNil(t, byName["with-hook"])
	assert.Equal(t, 2, byName["with-hook"].MinConnections, "and the outbound's own YAML block")
	assert.Nil(t, byName["without-hook"], "nil when the outbound has no block")

	a, b := outboundOf(t, cfg, "with-hook"), outboundOf(t, cfg, "without-hook")
	tr := a.transport
	require.NoError(t, tr.Start())
	require.NoError(t, a.Start())
	require.NoError(t, b.Start())
	defer func() {
		assert.NoError(t, a.Stop())
		assert.NoError(t, b.Stop())
		assert.NoError(t, tr.Stop())
	}()

	maxByAddr := map[string]int{}
	for _, p := range tr.peers {
		maxByAddr[p.addr] = p.poolConfig.Get().MaxConnections
	}
	assert.Equal(t, 22, maxByAddr["127.0.0.1:4040"], "the outbound hook replaces the global one")
	assert.Equal(t, 11, maxByAddr["127.0.0.1:4041"], "a nil result keeps the global hook")
}

func TestTransportSpecAcceptsDialOptions(t *testing.T) {
	cfg, err := loadConfig(t, m{"outbounds": singleOutbound(nil)},
		OutboundConnectionPool(ClientConnectionPoolConfig{MinConnections: 7, MaxConnections: 9}))
	require.NoError(t, err)

	ob := outboundOf(t, cfg, "svc")
	tr := ob.transport
	require.NoError(t, tr.Start())
	require.NoError(t, ob.Start())
	defer func() {
		assert.NoError(t, ob.Stop())
		assert.NoError(t, tr.Stop())
	}()
	for _, p := range tr.peers {
		assert.Equal(t, 7, p.poolConfig.Startup().MinConnections)
		assert.Equal(t, 9, p.poolConfig.Startup().MaxConnections)
	}
}
