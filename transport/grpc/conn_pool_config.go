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

	"go.uber.org/zap"
)

// applyPoolOverride overlays non-zero fields from override onto base.
// Explicit DynamicScalingEnabled false disables; explicit true is ignored.
func applyPoolOverride(base connPoolConfig, override *ClientConnectionPoolConfig) connPoolConfig {
	if override == nil {
		return base
	}
	cfg := base
	if override.DynamicScalingEnabled != nil && !*override.DynamicScalingEnabled {
		cfg.dynamicScalingEnabled = false
	}
	if override.MaxConcurrentStreams > 0 {
		cfg.maxConcurrentStreams = override.MaxConcurrentStreams
	}
	if override.ScaleUpThreshold > 0 {
		cfg.scaleUpThreshold = override.ScaleUpThreshold
	}
	if override.ScaleDownGap > 0 {
		cfg.scaleDownGap = override.ScaleDownGap
	}
	if override.MinConnections > 0 {
		cfg.minConnections = override.MinConnections
	}
	if override.MaxConnections > 0 {
		cfg.maxConnections = override.MaxConnections
	}
	if override.IdleTimeout > 0 {
		cfg.idleTimeout = override.IdleTimeout
	}
	if override.ScalingMonitorInterval > 0 {
		cfg.scalingMonitorInterval = override.ScalingMonitorInterval
	}
	return cfg
}

func validateResolvedConnPool(cfg connPoolConfig) error {
	if cfg.minConnections < 0 {
		return fmt.Errorf("clientConnectionPool.minConnections must be non-negative, got %d", cfg.minConnections)
	}
	if cfg.maxConnections < cfg.minConnections {
		return fmt.Errorf("clientConnectionPool.maxConnections (%d) must be >= minConnections (%d)", cfg.maxConnections, cfg.minConnections)
	}
	if cfg.maxConcurrentStreams < 1 {
		return fmt.Errorf("clientConnectionPool.maxConcurrentStreams must be at least 1, got %d", cfg.maxConcurrentStreams)
	}
	if cfg.scaleUpThreshold <= 0 || cfg.scaleUpThreshold > 1 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold must be in (0, 1], got %v", cfg.scaleUpThreshold)
	}
	if cfg.scaleUpThreshold-cfg.scaleDownGap <= 0 {
		return fmt.Errorf("clientConnectionPool.scaleUpThreshold (%.2f) minus scaleDownGap (%.2f) must be > 0",
			cfg.scaleUpThreshold, cfg.scaleDownGap)
	}
	if cfg.idleTimeout < 0 {
		return fmt.Errorf("clientConnectionPool.idleTimeout must be non-negative, got %v", cfg.idleTimeout)
	}
	return nil
}

func (p *grpcPeer) liveProvider() ConnectionPoolConfigProvider {
	if p.poolConfigProvider != nil {
		return p.poolConfigProvider
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
func (p *grpcPeer) livePoolCfg() connPoolConfig {
	base := p.poolCfg
	provider := p.liveProvider()
	if provider == nil {
		return base
	}
	overlay := provider()
	next := applyPoolOverride(base, &overlay)
	if err := validateResolvedConnPool(next); err != nil {
		if p.invalidLiveLogged.CompareAndSwap(false, true) {
			p.t.options.logger.Warn("grpc: ignoring invalid live connection pool config",
				zap.String("peer", p.HostPort()),
				zap.Error(err))
		}
		if v := p.liveCfg.Load(); v != nil {
			if prev, ok := v.(connPoolConfig); ok {
				return prev
			}
		}
		return base
	}
	p.invalidLiveLogged.Store(false)
	p.liveCfg.Store(next)
	return next
}
