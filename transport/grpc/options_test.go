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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func baseTestPoolConfig() connPoolConfig {
	return connPoolConfig{
		dynamicScalingEnabled:  true,
		maxConcurrentStreams:   250,
		scaleUpThreshold:       0.8,
		scaleDownGap:           0.1,
		minConnections:         1,
		maxConnections:         5,
		idleTimeout:            15 * time.Minute,
		scalingMonitorInterval: 30 * time.Second,
	}
}

func TestResolvedPoolConfig_NotIsolated_OverrideIgnored(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{
		connectionPerOutbound: false,
		connPoolOverride: &ClientConnectionPoolConfig{
			MaxConcurrentStreams: 50,
			MaxConnections:       20,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.Equal(t, base, got, "override on a non-isolated dialer must be ignored")
}

func TestResolvedPoolConfig_Isolated_NoOverride(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{connectionPerOutbound: true}

	got := d.resolvedPoolConfig(base)

	assert.Equal(t, base, got, "isolation alone (no override set) must leave the common config unchanged")
}

func TestResolvedPoolConfig_Isolated_OverrideTakesPriority(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{
		connectionPerOutbound: true,
		connPoolOverride: &ClientConnectionPoolConfig{
			MaxConcurrentStreams:   50,
			ScaleUpThreshold:       0.5,
			ScaleDownGap:           0.05,
			MinConnections:         2,
			MaxConnections:         20,
			IdleTimeout:            5 * time.Minute,
			ScalingMonitorInterval: time.Minute,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.Equal(t, connPoolConfig{
		dynamicScalingEnabled:  true,
		maxConcurrentStreams:   50,
		scaleUpThreshold:       0.5,
		scaleDownGap:           0.05,
		minConnections:         2,
		maxConnections:         20,
		idleTimeout:            5 * time.Minute,
		scalingMonitorInterval: time.Minute,
	}, got)
}

func TestResolvedPoolConfig_Isolated_PartialOverrideFallsBackToBase(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{
		connectionPerOutbound: true,
		connPoolOverride: &ClientConnectionPoolConfig{
			MaxConnections: 20,
		},
	}

	got := d.resolvedPoolConfig(base)

	want := base
	want.maxConnections = 20
	assert.Equal(t, want, got, "fields left unset in the override must fall back to the common config")
}

func TestResolvedPoolConfig_Isolated_ExplicitDisable(t *testing.T) {
	base := baseTestPoolConfig()
	disabled := false
	d := &dialOptions{
		connectionPerOutbound: true,
		connPoolOverride: &ClientConnectionPoolConfig{
			DynamicScalingEnabled: &disabled,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.False(t, got.dynamicScalingEnabled)
}

func TestResolvedPoolConfig_Isolated_ExplicitEnableIgnored(t *testing.T) {
	base := baseTestPoolConfig()
	base.dynamicScalingEnabled = false
	enabled := true
	d := &dialOptions{
		connectionPerOutbound: true,
		connPoolOverride: &ClientConnectionPoolConfig{
			DynamicScalingEnabled: &enabled,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.False(t, got.dynamicScalingEnabled, "explicit true must be ignored, matching the transport-level rule")
}
