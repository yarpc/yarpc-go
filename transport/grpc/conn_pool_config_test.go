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

func staticPoolProvider(cfg ClientConnectionPoolConfig) ConnectionPoolConfigProvider {
	return func() ClientConnectionPoolConfig { return cfg }
}

func liveTestPeer(t *testing.T, base connPoolConfig, provider ConnectionPoolConfigProvider) *grpcPeer {
	t.Helper()
	transport := NewTransport(WithConnectionPoolConfigProvider(provider))
	p := &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:       transport,
		poolCfg: base,
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

func TestLivePoolCfg_TransportHookOverlays(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 9}))

	got := p.livePoolCfg()

	assert.Equal(t, 9, got.maxConnections)
	assert.Equal(t, base.maxConcurrentStreams, got.maxConcurrentStreams, "unset live fields inherit startup")
}

func TestLivePoolCfg_DialerHookOverridesTransport(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 9}))
	p.poolConfigProvider = staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 20})

	assert.Equal(t, 20, p.livePoolCfg().maxConnections)
}

func TestLivePoolCfg_InvalidKeepsLastGood(t *testing.T) {
	base := baseTestPoolConfig()
	var mu sync.Mutex
	overlay := ClientConnectionPoolConfig{MaxConnections: 12}
	p := liveTestPeer(t, base, func() ClientConnectionPoolConfig {
		mu.Lock()
		defer mu.Unlock()
		return overlay
	})

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
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{
		ScaleUpThreshold: 0.8,
		ScaleDownGap:     0.85,
	}))

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
	transport := NewTransport(WithConnectionPoolConfigProvider(staticPoolProvider(ClientConnectionPoolConfig{
		DynamicScalingEnabled: &disabled,
	})))
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
		WithConnectionPoolConfigProvider(staticPoolProvider(ClientConnectionPoolConfig{
			DynamicScalingEnabled: &disabled,
		})),
	)
	p := peerForPool(t)
	p.t = transport
	p.poolCfg.dynamicScalingEnabled = true
	p.liveCfg.Store(p.poolCfg)

	p.tryScaleUp(makeConn(connStateActive, 85))

	assert.Equal(t, int32(0), p.isScaling)
}

func TestWithConnectionPoolConfigProvider(t *testing.T) {
	opts := newTransportOptions([]TransportOption{
		WithConnectionPoolConfigProvider(staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 3})),
	})
	require.NotNil(t, opts.poolConfigProvider)
	assert.Equal(t, 3, opts.poolConfigProvider().MaxConnections)
}

func TestClientConnectionPoolProvider_IsolatedOnly(t *testing.T) {
	address := startTestServer(t)
	transport := NewTransport(
		MaxConnections(5),
		MinConnections(1),
		WithConnectionPoolConfigProvider(staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 7})),
	)
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{address}
	shared := transport.NewDialer(ClientConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{
		MaxConnections: 20,
	})))
	sp, err := shared.RetainPeer(id, idSubscriber{1})
	require.NoError(t, err)
	sharedPeer := sp.(*grpcPeer)
	assert.Nil(t, sharedPeer.poolConfigProvider)
	assert.Equal(t, 7, sharedPeer.livePoolCfg().maxConnections, "shared dialer uses the transport hook")

	isolated := transport.NewDialer(ClientConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{
		MaxConnections: 20,
	}))).WithConnectionIsolation()
	ip, err := isolated.RetainPeer(id, idSubscriber{2})
	require.NoError(t, err)
	isolatedPeer := ip.(*grpcPeer)
	require.NotNil(t, isolatedPeer.poolConfigProvider)
	assert.Equal(t, 20, isolatedPeer.livePoolCfg().maxConnections, "isolated dialer uses its own hook")

	require.NoError(t, shared.ReleasePeer(id, idSubscriber{1}))
	require.NoError(t, isolated.ReleasePeer(id, idSubscriber{2}))
}

func TestLivePoolCfg_LogsInvalidOnce(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{
		ScaleUpThreshold: 0.8,
		ScaleDownGap:     0.85,
	}))
	p.t.options.logger = zap.NewNop()

	_ = p.livePoolCfg()
	_ = p.livePoolCfg()
	assert.True(t, p.invalidLiveLogged.Load())
}
