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

package connpool

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func boolPtr(b bool) *bool { return &b }

func TestConfigApply(t *testing.T) {
	base := baseConfig()

	t.Run("nil override is a no-op", func(t *testing.T) {
		assert.Equal(t, base, base.Apply(nil))
	})

	t.Run("zero override is a no-op", func(t *testing.T) {
		assert.Equal(t, base, base.Apply(&Override{}))
	})

	t.Run("every set field replaces the base", func(t *testing.T) {
		got := base.Apply(&Override{
			DynamicScalingEnabled:  boolPtr(false),
			MaxConcurrentStreams:   7,
			ScaleUpThreshold:       0.5,
			ScaleDownGap:           0.2,
			MinConnections:         3,
			MaxConnections:         9,
			IdleTimeout:            time.Second,
			ScalingMonitorInterval: time.Minute,
		})
		assert.Equal(t, Config{
			DynamicScalingEnabled:  false,
			MaxConcurrentStreams:   7,
			ScaleUpThreshold:       0.5,
			ScaleDownGap:           0.2,
			MinConnections:         3,
			MaxConnections:         9,
			IdleTimeout:            time.Second,
			ScalingMonitorInterval: time.Minute,
		}, got)
	})

	t.Run("nil DynamicScalingEnabled keeps the base flag, set value replaces it", func(t *testing.T) {
		assert.True(t, base.Apply(&Override{MinConnections: 2}).DynamicScalingEnabled)
		off := base.Apply(&Override{DynamicScalingEnabled: boolPtr(false)})
		assert.False(t, off.DynamicScalingEnabled)
		assert.True(t, off.Apply(&Override{DynamicScalingEnabled: boolPtr(true)}).DynamicScalingEnabled)
	})

	t.Run("does not mutate the receiver", func(t *testing.T) {
		orig := base
		_ = base.Apply(&Override{MaxConnections: 99})
		assert.Equal(t, orig, base)
	})
}

func TestConfigValidate(t *testing.T) {
	require.NoError(t, baseConfig().Validate())

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"negative min", func(c *Config) { c.MinConnections = -1 }, "minConnections must be non-negative, got -1"},
		{"max below min", func(c *Config) { c.MinConnections, c.MaxConnections = 4, 2 }, "maxConnections (2) must be >= minConnections (4)"},
		{"zero streams", func(c *Config) { c.MaxConcurrentStreams = 0 }, "maxConcurrentStreams must be at least 1, got 0"},
		{"zero threshold", func(c *Config) { c.ScaleUpThreshold = 0 }, "scaleUpThreshold must be in (0, 1], got 0"},
		{"threshold above one", func(c *Config) { c.ScaleUpThreshold = 1.5 }, "scaleUpThreshold must be in (0, 1], got 1.5"},
		{"gap swallows threshold", func(c *Config) { c.ScaleUpThreshold, c.ScaleDownGap = 0.5, 0.5 }, "scaleUpThreshold (0.50) minus scaleDownGap (0.50) must be > 0"},
		{"negative idle timeout", func(c *Config) { c.IdleTimeout = -time.Second }, "idleTimeout must be non-negative, got -1s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "clientConnectionPool."+tt.wantErr)
		})
	}

	t.Run("min equal to max and scaling off are valid", func(t *testing.T) {
		cfg := baseConfig()
		cfg.MinConnections, cfg.MaxConnections = 3, 3
		cfg.DynamicScalingEnabled = false
		assert.NoError(t, cfg.Validate())
	})
}

func TestOverrideValidate(t *testing.T) {
	require.NoError(t, Override{}.Validate(), "the zero override means inherit everything")
	require.NoError(t, Override{MinConnections: 2, MaxConnections: 2, ScaleUpThreshold: 1, ScaleDownGap: 0.99}.Validate())

	tests := []struct {
		name    string
		o       Override
		wantErr string
	}{
		{"negative min", Override{MinConnections: -1}, "minConnections must be non-negative, got -1"},
		{"negative max", Override{MaxConnections: -1}, "maxConnections must be non-negative, got -1"},
		{"max below min", Override{MinConnections: 5, MaxConnections: 2}, "maxConnections (2) must be >= minConnections (5)"},
		{"negative streams", Override{MaxConcurrentStreams: -1}, "maxConcurrentStreams must be non-negative, got -1"},
		{"negative threshold", Override{ScaleUpThreshold: -0.1}, "scaleUpThreshold must be in [0, 1], got -0.1"},
		{"threshold above one", Override{ScaleUpThreshold: 1.1}, "scaleUpThreshold must be in [0, 1], got 1.1"},
		{"negative gap", Override{ScaleDownGap: -0.1}, "scaleDownGap must be in [0, 1), got -0.1"},
		{"gap of one", Override{ScaleDownGap: 1}, "scaleDownGap must be in [0, 1), got 1"},
		{"negative idle timeout", Override{IdleTimeout: -1}, "idleTimeout must be non-negative, got -1ns"},
		{"negative interval", Override{ScalingMonitorInterval: -1}, "scalingMonitorInterval must be non-negative, got -1ns"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.o.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "clientConnectionPool."+tt.wantErr)
		})
	}
}

func TestConfigResolver(t *testing.T) {
	startup := baseConfig()

	t.Run("without a provider it returns the startup config", func(t *testing.T) {
		r := NewConfigResolver(startup, nil, nil, "t", "peer")
		assert.False(t, r.HasProvider())
		assert.Equal(t, startup, r.Startup())
		assert.Equal(t, startup, r.Get())
	})

	t.Run("a valid overlay replaces set fields on every call", func(t *testing.T) {
		var max atomic.Int32
		max.Store(7)
		r := NewConfigResolver(startup, func() Override {
			return Override{MaxConnections: int(max.Load())}
		}, nil, "t", "peer")
		assert.True(t, r.HasProvider())

		assert.Equal(t, 7, r.Get().MaxConnections)
		max.Store(9)
		assert.Equal(t, 9, r.Get().MaxConnections, "a changed provider value takes effect on the next Get")
		assert.Equal(t, startup, r.Startup(), "the startup snapshot is never modified")
	})

	t.Run("an invalid overlay keeps the last valid config and warns once per streak", func(t *testing.T) {
		core, logs := observer.New(zap.WarnLevel)
		var overlay atomic.Pointer[Override]
		overlay.Store(&Override{MaxConnections: 8})
		r := NewConfigResolver(startup, func() Override { return *overlay.Load() }, zap.New(core), "http2", "10.0.0.1:80")

		require.Equal(t, 8, r.Get().MaxConnections)

		// min > max after applying: the resolved config is invalid.
		overlay.Store(&Override{MinConnections: 20})
		for range 5 {
			got := r.Get()
			assert.Equal(t, 8, got.MaxConnections, "falls back to the last valid snapshot, not the startup one")
			assert.Equal(t, 1, got.MinConnections)
		}
		require.Equal(t, 1, logs.Len(), "warn once per invalid streak, not on every call")
		entry := logs.All()[0]
		assert.Equal(t, "http2: ignoring invalid live connection pool config", entry.Message)
		assert.Equal(t, "10.0.0.1:80", entry.ContextMap()["peer"])
		assert.Contains(t, entry.ContextMap()["error"], "maxConnections (5) must be >= minConnections (20)")

		// Recovery re-arms the warning for the next invalid streak.
		overlay.Store(&Override{MaxConnections: 9})
		assert.Equal(t, 9, r.Get().MaxConnections)
		overlay.Store(&Override{MinConnections: 20})
		assert.Equal(t, 9, r.Get().MaxConnections)
		assert.Equal(t, 2, logs.Len(), "a new invalid streak warns again")
	})

	t.Run("an invalid overlay before any valid one falls back to startup", func(t *testing.T) {
		r := NewConfigResolver(startup, func() Override { return Override{MinConnections: 99} }, nil, "t", "peer")
		assert.Equal(t, startup, r.Get(), "a nil logger must not panic")
	})

	t.Run("live DynamicScalingEnabled flips the flag", func(t *testing.T) {
		var enabled atomic.Pointer[bool]
		enabled.Store(boolPtr(false))
		r := NewConfigResolver(startup, func() Override { return Override{DynamicScalingEnabled: enabled.Load()} }, nil, "t", "peer")
		assert.False(t, r.Get().DynamicScalingEnabled)
		enabled.Store(nil)
		assert.True(t, r.Get().DynamicScalingEnabled, "nil means no opinion, so the startup flag applies")
	})
}

func TestPool_StartMonitor(t *testing.T) {
	t.Run("starts the monitor once, after Start", func(t *testing.T) {
		cfg := baseConfig()
		cfg.MinConnections = 2
		cfg.ScalingMonitorInterval = time.Hour
		p, _ := newTestPool(t, cfg)
		require.NoError(t, p.Start(1, false))
		defer func() { p.Stop(); p.Wait() }()

		require.False(t, p.monitorStarted.Load())
		p.StartMonitor()
		p.StartMonitor() // idempotent
		assert.True(t, p.monitorStarted.Load())

		require.Eventually(t, func() bool {
			return len(p.LoadConns()) == 2
		}, time.Second, time.Millisecond, "the monitor's first pass must fill to MinConnections")
	})

	t.Run("is a no-op once the pool is stopped", func(t *testing.T) {
		p, _ := newTestPool(t, baseConfig())
		require.NoError(t, p.Start(1, false))
		p.Stop()
		p.Wait()

		p.StartMonitor()
		assert.False(t, p.monitorStarted.Load())
	})
}
