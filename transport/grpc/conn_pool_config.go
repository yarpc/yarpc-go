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

package grpc

import (
	"fmt"

	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
)

// applyPoolOverride overlays set fields from override onto base.
// A nil DynamicScalingEnabled leaves the base flag unchanged.
func applyPoolOverride(base connpool.Config, override *ClientConnectionPoolConfig) connpool.Config {
	if override == nil {
		return base
	}
	cfg := base
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

func (t *Transport) baseConnPoolConfig() connpool.Config {
	if t == nil || t.options == nil {
		return connpool.Config{}
	}
	o := t.options
	return connpool.Config{
		DynamicScalingEnabled:  o.clientConnPoolDynamicScalingEnabled,
		MaxConcurrentStreams:   o.clientConnPoolMaxConcurrentStreams,
		ScaleUpThreshold:       o.clientConnPoolScaleUpThreshold,
		ScaleDownGap:           o.clientConnPoolScaleDownGap,
		MinConnections:         o.clientConnPoolMinConnections,
		MaxConnections:         o.clientConnPoolMaxConnections,
		IdleTimeout:            o.clientConnPoolIdleTimeout,
		ScalingMonitorInterval: o.clientConnPoolScalingMonitorInterval,
	}
}

func validateResolvedConnPool(cfg connpool.Config) error {
	if cfg.MinConnections < 0 {
		return fmt.Errorf("clientConnectionPool.minConnections must be non-negative, got %d", cfg.MinConnections)
	}
	if cfg.MaxConnections < cfg.MinConnections {
		return fmt.Errorf("clientConnectionPool.maxConnections (%d) must be >= minConnections (%d)", cfg.MaxConnections, cfg.MinConnections)
	}
	if cfg.MaxConcurrentStreams < 1 {
		return fmt.Errorf("clientConnectionPool.maxConcurrentStreams must be at least 1, got %d", cfg.MaxConcurrentStreams)
	}
	if cfg.ScaleUpThreshold <= 0 || cfg.ScaleUpThreshold > 1 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold must be in (0, 1], got %v", cfg.ScaleUpThreshold)
	}
	if cfg.ScaleUpThreshold-cfg.ScaleDownGap <= 0 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold (%.2f) minus scaleDownGap (%.2f) must be > 0",
			cfg.ScaleUpThreshold, cfg.ScaleDownGap)
	}
	if cfg.IdleTimeout < 0 {
		return fmt.Errorf("clientConnectionPool.idleTimeout must be non-negative, got %v", cfg.IdleTimeout)
	}
	return nil
}

func (p *grpcPeer) liveProvider() LiveConnectionPoolProvider {
	if p.outboundLiveProvider != nil {
		return p.outboundLiveProvider
	}
	if p.t != nil && p.t.options != nil {
		return p.t.options.poolConfigProvider
	}
	return nil
}

// livePoolCfg returns the pool knobs for this scale decision: the startup
// snapshot, overlaid by the live hook when set. A broken live overlay is
// dropped and the last valid snapshot is used. The whole struct is swapped
// so the scaler never sees a torn mix of old and new fields.
func (p *grpcPeer) livePoolCfg() connpool.Config {
	base := p.startupPool
	provider := p.liveProvider()
	if provider == nil {
		return base
	}
	overlay := provider()
	next := applyPoolOverride(base, &overlay)
	if err := validateResolvedConnPool(next); err != nil {
		// Warn once per invalid streak. livePoolCfg runs on every RPC
		// scale-up and monitor tick; logging each time would flood.
		if p.invalidLiveWarned.CompareAndSwap(false, true) {
			p.t.options.logger.Warn("grpc: ignoring invalid live connection pool config",
				zap.String("peer", p.HostPort()),
				zap.Error(err))
		}
		if v := p.lastValidLivePool.Load(); v != nil {
			if prev, ok := v.(connpool.Config); ok {
				return prev
			}
		}
		return base
	}
	p.invalidLiveWarned.Store(false)
	p.lastValidLivePool.Store(next)
	return next
}
