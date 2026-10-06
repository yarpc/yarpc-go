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
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Override is a partial Config: fields left at their zero value mean "no
// opinion" and keep whatever Config the override is applied to. It is the
// transport-agnostic shape of a YAML clientConnectionPool block or of a live
// provider's return value; transports convert their public config types
// into it.
type Override struct {
	// DynamicScalingEnabled replaces Config.DynamicScalingEnabled when
	// non-nil. A pointer is needed because a plain bool left out of YAML
	// decodes as false and would turn scaling off for everyone that omits
	// the field.
	DynamicScalingEnabled  *bool
	MaxConcurrentStreams   int32
	ScaleUpThreshold       float64
	ScaleDownGap           float64
	MinConnections         int
	MaxConnections         int
	IdleTimeout            time.Duration
	ScalingMonitorInterval time.Duration
}

// Apply overlays the set fields of override onto c and returns the result.
// A nil override returns c unchanged. The result is not validated; see
// Validate.
func (c Config) Apply(override *Override) Config {
	if override == nil {
		return c
	}
	cfg := c
	if override.DynamicScalingEnabled != nil {
		cfg.DynamicScalingEnabled = *override.DynamicScalingEnabled
	}
	if override.MaxConcurrentStreams > 0 {
		cfg.MaxConcurrentStreams = override.MaxConcurrentStreams
	}
	if override.ScaleUpThreshold > 0 {
		cfg.ScaleUpThreshold = override.ScaleUpThreshold
	}
	if override.ScaleDownGap > 0 {
		cfg.ScaleDownGap = override.ScaleDownGap
	}
	if override.MinConnections > 0 {
		cfg.MinConnections = override.MinConnections
	}
	if override.MaxConnections > 0 {
		cfg.MaxConnections = override.MaxConnections
	}
	if override.IdleTimeout > 0 {
		cfg.IdleTimeout = override.IdleTimeout
	}
	if override.ScalingMonitorInterval > 0 {
		cfg.ScalingMonitorInterval = override.ScalingMonitorInterval
	}
	return cfg
}

// Validate checks a fully resolved Config: every field the scaler divides
// by or compares against must be usable. Error messages use the
// clientConnectionPool.* names that the YAML configuration exposes.
func (c Config) Validate() error {
	if c.MinConnections < 0 {
		return fmt.Errorf("clientConnectionPool.minConnections must be non-negative, got %d", c.MinConnections)
	}
	if c.MaxConnections < c.MinConnections {
		return fmt.Errorf("clientConnectionPool.maxConnections (%d) must be >= minConnections (%d)", c.MaxConnections, c.MinConnections)
	}
	if c.MaxConcurrentStreams < 1 {
		return fmt.Errorf("clientConnectionPool.maxConcurrentStreams must be at least 1, got %d", c.MaxConcurrentStreams)
	}
	if c.ScaleUpThreshold <= 0 || c.ScaleUpThreshold > 1 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold must be in (0, 1], got %v", c.ScaleUpThreshold)
	}
	if c.ScaleUpThreshold-c.ScaleDownGap <= 0 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold (%.2f) minus scaleDownGap (%.2f) must be > 0",
			c.ScaleUpThreshold, c.ScaleDownGap)
	}
	if c.IdleTimeout < 0 {
		return fmt.Errorf("clientConnectionPool.idleTimeout must be non-negative, got %v", c.IdleTimeout)
	}
	return nil
}

// Validate checks an Override on its own, before it is applied to a base
// Config: zero values are allowed (they mean "inherit"), but set values must
// be in range. Cross-field constraints that depend on the base config are
// checked by Config.Validate on the resolved result.
func (o Override) Validate() error {
	if o.MinConnections < 0 {
		return fmt.Errorf("clientConnectionPool.minConnections must be non-negative, got %d", o.MinConnections)
	}
	if o.MaxConnections < 0 {
		return fmt.Errorf("clientConnectionPool.maxConnections must be non-negative, got %d", o.MaxConnections)
	}
	if o.MaxConnections > 0 && o.MinConnections > o.MaxConnections {
		return fmt.Errorf("clientConnectionPool.maxConnections (%d) must be >= minConnections (%d)", o.MaxConnections, o.MinConnections)
	}
	if o.MaxConcurrentStreams < 0 {
		return fmt.Errorf("clientConnectionPool.maxConcurrentStreams must be non-negative, got %d", o.MaxConcurrentStreams)
	}
	if o.ScaleUpThreshold < 0 || o.ScaleUpThreshold > 1 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold must be in [0, 1], got %v", o.ScaleUpThreshold)
	}
	if o.ScaleDownGap < 0 || o.ScaleDownGap >= 1 {
		return fmt.Errorf("clientConnectionPool.scaleDownGap must be in [0, 1), got %v", o.ScaleDownGap)
	}
	if o.IdleTimeout < 0 {
		return fmt.Errorf("clientConnectionPool.idleTimeout must be non-negative, got %v", o.IdleTimeout)
	}
	if o.ScalingMonitorInterval < 0 {
		return fmt.Errorf("clientConnectionPool.scalingMonitorInterval must be non-negative, got %v", o.ScalingMonitorInterval)
	}
	return nil
}

// ConfigResolver resolves the Config a Pool should use at each scaling decision:
// a startup snapshot, overlaid by a live provider when one is set. Its Get
// method is meant to be passed as the Pool's Config function.
//
// A live overlay that resolves to an invalid Config is dropped and the last
// valid snapshot is used instead. The whole Config struct is swapped, so the
// scaler never sees a torn mix of old and new fields.
type ConfigResolver struct {
	startup  Config
	provider func() Override

	logger    *zap.Logger
	logPrefix string
	id        string

	// lastValid holds the last Config that passed validation (a Config).
	lastValid atomic.Value
	// invalidWarned limits the invalid-overlay warning to once per invalid
	// streak: Get runs on every RPC scale-up check and monitor tick, so
	// logging each time would flood.
	invalidWarned atomic.Bool
}

// NewConfigResolver creates a ConfigResolver. provider may be nil, in which case Get
// always returns startup. logger may be nil. logPrefix and id label the
// warning the same way Pool.LogPrefix and Pool.ID label the pool's own logs.
func NewConfigResolver(startup Config, provider func() Override, logger *zap.Logger, logPrefix, id string) *ConfigResolver {
	l := &ConfigResolver{
		startup:   startup,
		provider:  provider,
		logger:    logger,
		logPrefix: logPrefix,
		id:        id,
	}
	l.lastValid.Store(startup)
	return l
}

// Startup returns the startup snapshot, before any live overlay.
func (l *ConfigResolver) Startup() Config { return l.startup }

// HasProvider reports whether a live provider is set. A pool with a provider
// needs its scaling monitor even when scaling is off at startup, because the
// provider may turn it on later (or turn it off, in which case the monitor
// winds extra connections down).
func (l *ConfigResolver) HasProvider() bool { return l.provider != nil }

// Get returns the current Config: the startup snapshot overlaid by the live
// provider, or the last valid snapshot if the overlay is invalid. Safe for
// concurrent use.
func (l *ConfigResolver) Get() Config {
	if l.provider == nil {
		return l.startup
	}
	overlay := l.provider()
	next := l.startup.Apply(&overlay)
	if err := next.Validate(); err != nil {
		if l.logger != nil && l.invalidWarned.CompareAndSwap(false, true) {
			l.logger.Warn(l.logPrefix+": ignoring invalid live connection pool config",
				zap.String("peer", l.id),
				zap.Error(err))
		}
		if prev, ok := l.lastValid.Load().(Config); ok {
			return prev
		}
		return l.startup
	}
	l.invalidWarned.Store(false)
	l.lastValid.Store(next)
	return next
}
