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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/zap"
)

type poolProviderFunc func(dest string) ClientConnectionPoolConfig

func (f poolProviderFunc) ConnectionPoolConfig(dest string) ClientConnectionPoolConfig {
	return f(dest)
}

type destPoolProvider map[string]ClientConnectionPoolConfig

func (m destPoolProvider) ConnectionPoolConfig(dest string) ClientConnectionPoolConfig {
	return m[dest]
}

func liveTestPeer(t *testing.T, base connPoolConfig, isolated bool, dest string, provider ConnectionPoolConfigProvider) *grpcPeer {
	t.Helper()
	transport := NewTransport(WithConnectionPoolConfigProvider(provider))
	p := &grpcPeer{
		Peer:            abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:               transport,
		poolCfg:         base,
		isolated:        isolated,
		destServiceName: dest,
	}
	p.liveCfg.Store(base)
	return p
}

func TestLivePoolCfg_NoProviderUsesStartup(t *testing.T) {
	base := baseTestPoolConfig()
	transport := NewTransport()
	p := &grpcPeer{
		t:       transport,
		poolCfg: base,
	}

	assert.Equal(t, base, p.livePoolCfg())
}

func TestLivePoolCfg_SharedDestIsEmpty(t *testing.T) {
	base := baseTestPoolConfig()
	var gotDest string
	p := liveTestPeer(t, base, false, "ignored", poolProviderFunc(func(dest string) ClientConnectionPoolConfig {
		gotDest = dest
		return ClientConnectionPoolConfig{MaxConnections: 9}
	}))

	got := p.livePoolCfg()

	assert.Equal(t, "", gotDest, "shared transport pool must query dest \"\"")
	assert.Equal(t, 9, got.maxConnections)
	assert.Equal(t, base.maxConcurrentStreams, got.maxConcurrentStreams, "unset live fields inherit startup")
}

func TestLivePoolCfg_IsolatedUsesDestService(t *testing.T) {
	base := baseTestPoolConfig()
	provider := destPoolProvider{
		"":        {MaxConnections: 9},
		"lottery": {MaxConnections: 20},
	}
	p := liveTestPeer(t, base, true, "lottery", provider)

	assert.Equal(t, 20, p.livePoolCfg().maxConnections)
}

func TestLivePoolCfg_InvalidKeepsLastGood(t *testing.T) {
	base := baseTestPoolConfig()
	var mu sync.Mutex
	overlay := ClientConnectionPoolConfig{MaxConnections: 12}
	p := liveTestPeer(t, base, false, "", poolProviderFunc(func(string) ClientConnectionPoolConfig {
		mu.Lock()
		defer mu.Unlock()
		return overlay
	}))

	assert.Equal(t, 12, p.livePoolCfg().maxConnections)

	mu.Lock()
	// scaleUp - gap <= 0 is rejected by validateResolvedConnPool.
	overlay = ClientConnectionPoolConfig{ScaleUpThreshold: 0.8, ScaleDownGap: 0.85}
	mu.Unlock()

	got := p.livePoolCfg()
	assert.Equal(t, 12, got.maxConnections, "invalid live overlay must keep the last valid snapshot")
	assert.Equal(t, base.scaleUpThreshold, got.scaleUpThreshold)
}

func TestLivePoolCfg_InvalidFallsBackToStartup(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, false, "", destPoolProvider{
		"": {ScaleUpThreshold: 0.8, ScaleDownGap: 0.85},
	})

	assert.Equal(t, base, p.livePoolCfg())
}

func TestValidateResolvedConnPool(t *testing.T) {
	t.Parallel()

	valid := baseTestPoolConfig()
	require.NoError(t, validateResolvedConnPool(valid))

	tests := []struct {
		name string
		mut  func(*connPoolConfig)
	}{
		{"min greater than max", func(c *connPoolConfig) { c.minConnections = 8; c.maxConnections = 2 }},
		{"zero streams", func(c *connPoolConfig) { c.maxConcurrentStreams = 0 }},
		{"zero scale-up threshold", func(c *connPoolConfig) { c.scaleUpThreshold = 0 }},
		{"hysteresis collapse", func(c *connPoolConfig) { c.scaleUpThreshold = 0.5; c.scaleDownGap = 0.5 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mut(&cfg)
			assert.Error(t, validateResolvedConnPool(cfg))
		})
	}
}

func TestMaybeScaleDown_LiveDisabled(t *testing.T) {
	disabled := false
	conns := []*grpcClientConnWrapper{
		makeConn(connStateActive, 5),
		makeConn(connStateActive, 5),
		makeConn(connStateActive, 5),
	}
	transport := NewTransport(WithConnectionPoolConfigProvider(destPoolProvider{
		"": {DynamicScalingEnabled: &disabled},
	}))
	cfg := defaultScaleDownCfg
	cfg.dynamicScalingEnabled = true
	cfg.maxConnections = 5
	cfg.scaleDownGap = 0.1
	p := &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:       transport,
		poolCfg: cfg,
	}
	p.liveCfg.Store(p.poolCfg)
	p.storeConns(conns)

	p.maybeScaleDown()

	for i, c := range conns {
		assert.Equal(t, connStateActive, c.getState(), "conn[%d] must stay active when live config disables scaling", i)
	}
}

func TestTryScaleUp_LiveDisabled(t *testing.T) {
	disabled := false
	transport := NewTransport(
		WithDynamicConnectionScaling(true),
		WithConnectionPoolConfigProvider(destPoolProvider{
			"": {DynamicScalingEnabled: &disabled},
		}),
	)
	p := peerForPool(t)
	p.t = transport
	p.poolCfg.dynamicScalingEnabled = true
	p.liveCfg.Store(p.poolCfg)

	p.tryScaleUp(makeConn(connStateActive, 85))

	assert.Equal(t, int32(0), p.isScaling)
}

func TestWithConnectionPoolConfigProvider(t *testing.T) {
	provider := destPoolProvider{"": {MaxConnections: 3}}
	opts := newTransportOptions([]TransportOption{
		WithConnectionPoolConfigProvider(provider),
	})
	require.NotNil(t, opts.poolConfigProvider)
	assert.Equal(t, 3, opts.poolConfigProvider.ConnectionPoolConfig("").MaxConnections)
}

func TestLivePoolCfg_LogsInvalidOnce(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, false, "", destPoolProvider{
		"": {ScaleUpThreshold: 0.8, ScaleDownGap: 0.85},
	})
	p.t.options.logger = zap.NewNop()

	_ = p.livePoolCfg()
	_ = p.livePoolCfg()
	assert.True(t, p.invalidLiveLogged.Load())
}
