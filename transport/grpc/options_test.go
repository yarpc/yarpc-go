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
	"go.uber.org/yarpc/transport/internal/connpool"
)

func baseTestPoolConfig() connpool.Config {
	return connpool.Config{
		DynamicScalingEnabled:  true,
		MaxConcurrentStreams:   250,
		ScaleUpThreshold:       0.8,
		ScaleDownGap:           0.1,
		MinConnections:         1,
		MaxConnections:         5,
		IdleTimeout:            15 * time.Minute,
		ScalingMonitorInterval: 30 * time.Second,
	}
}

func TestResolvedPoolConfig_NoOverride(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{}

	got := d.resolvedPoolConfig(base)

	assert.Equal(t, base, got, "no override must leave the common config unchanged")
}

func TestResolvedPoolConfig_OverrideTakesPriority(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{
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

	assert.Equal(t, connpool.Config{
		DynamicScalingEnabled:  true,
		MaxConcurrentStreams:   50,
		ScaleUpThreshold:       0.5,
		ScaleDownGap:           0.05,
		MinConnections:         2,
		MaxConnections:         20,
		IdleTimeout:            5 * time.Minute,
		ScalingMonitorInterval: time.Minute,
	}, got)
}

func TestResolvedPoolConfig_PartialOverrideFallsBackToBase(t *testing.T) {
	base := baseTestPoolConfig()
	d := &dialOptions{
		connPoolOverride: &ClientConnectionPoolConfig{
			MaxConnections: 20,
		},
	}

	got := d.resolvedPoolConfig(base)

	want := base
	want.MaxConnections = 20
	assert.Equal(t, want, got, "fields left unset in the override must fall back to the common config")
}

func TestResolvedPoolConfig_ExplicitDisable(t *testing.T) {
	base := baseTestPoolConfig()
	disabled := false
	d := &dialOptions{
		connPoolOverride: &ClientConnectionPoolConfig{
			DynamicScalingEnabled: &disabled,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.False(t, got.DynamicScalingEnabled)
}

func TestResolvedPoolConfig_ExplicitEnable(t *testing.T) {
	base := baseTestPoolConfig()
	base.DynamicScalingEnabled = false
	enabled := true
	d := &dialOptions{
		connPoolOverride: &ClientConnectionPoolConfig{
			DynamicScalingEnabled: &enabled,
		},
	}

	got := d.resolvedPoolConfig(base)

	assert.True(t, got.DynamicScalingEnabled)
}
