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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func staticPoolProvider(cfg ClientConnectionPoolConfig) LiveConnectionPoolProvider {
	return func() ClientConnectionPoolConfig { return cfg }
}

func liveTestPeer(t *testing.T, base connpool.Config, provider LiveConnectionPoolProvider) *grpcPeer {
	t.Helper()
	transport := NewTransport(WithGlobalLiveConnectionPoolProvider(provider))
	p := &grpcPeer{
		Peer:        abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:           transport,
		startupPool: base,
	}
	p.lastValidLivePool.Store(base)
	return p
}

func TestLivePoolCfg_NoProviderUsesStartup(t *testing.T) {
	base := baseTestPoolConfig()
	transport := NewTransport()
	p := &grpcPeer{
		t:           transport,
		startupPool: base,
	}

	assert.Equal(t, base, p.livePoolCfg())
}

func TestLivePoolCfg_TransportHookOverlays(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 9}))

	got := p.livePoolCfg()

	assert.Equal(t, 9, got.MaxConnections)
	assert.Equal(t, base.MaxConcurrentStreams, got.MaxConcurrentStreams, "unset live fields inherit startup")
}

func TestLivePoolCfg_DialerHookOverridesTransport(t *testing.T) {
	base := baseTestPoolConfig()
	p := liveTestPeer(t, base, staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 9}))
	p.outboundLiveProvider = staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 20})

	assert.Equal(t, 20, p.livePoolCfg().MaxConnections)
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

	assert.Equal(t, 12, p.livePoolCfg().MaxConnections)

	mu.Lock()
	// scaleUp - gap <= 0 is rejected by validateResolvedConnPool.
	overlay = ClientConnectionPoolConfig{ScaleUpThreshold: 0.8, ScaleDownGap: 0.85}
	mu.Unlock()

	got := p.livePoolCfg()
	assert.Equal(t, 12, got.MaxConnections, "invalid live overlay must keep the last valid snapshot")
	assert.Equal(t, base.ScaleUpThreshold, got.ScaleUpThreshold)
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
		mut  func(*connpool.Config)
	}{
		{"min greater than max", func(c *connpool.Config) { c.MinConnections = 8; c.MaxConnections = 2 }},
		{"zero streams", func(c *connpool.Config) { c.MaxConcurrentStreams = 0 }},
		{"zero scale-up threshold", func(c *connpool.Config) { c.ScaleUpThreshold = 0 }},
		{"hysteresis collapse", func(c *connpool.Config) { c.ScaleUpThreshold = 0.5; c.ScaleDownGap = 0.5 }},
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
	transport := NewTransport(WithGlobalLiveConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{
		DynamicScalingEnabled: &disabled,
	})))
	cfg := connpool.Config{
		DynamicScalingEnabled: true,
		MinConnections:        1,
		MaxConnections:        5,
		MaxConcurrentStreams:  100,
		ScaleUpThreshold:      0.8,
		ScaleDownGap:          0.1,
	}
	p := peerForPoolOnTransport(t, transport, cfg, nil)
	var conns []*connpool.Wrapper[*grpc.ClientConn]
	for _, streams := range []int{30, 5, 20} {
		w, err := p.pool.AddConn()
		require.NoError(t, err)
		for range streams {
			w.IncStreamCount()
		}
		conns = append(conns, w)
	}

	p.pool.MaybeScaleDown()

	assert.Equal(t, connpool.StateActive, conns[0].GetState(), "busiest conn should stay active")
	assert.Equal(t, connpool.StateDraining, conns[1].GetState(), "least-loaded extra should drain when scaling is off")
	assert.Equal(t, connpool.StateActive, conns[2].GetState())
}

func TestTryScaleUp_LiveDisabled(t *testing.T) {
	disabled := false
	transport := NewTransport(
		WithDynamicConnectionScaling(true),
		WithGlobalLiveConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{
			DynamicScalingEnabled: &disabled,
		})),
	)
	cfg := connpool.Config{
		DynamicScalingEnabled: true,
		MinConnections:        1,
		MaxConnections:        5,
		MaxConcurrentStreams:  100,
		ScaleUpThreshold:      0.8,
	}
	p := peerForPoolOnTransport(t, transport, cfg, nil)
	w, err := p.pool.AddConn()
	require.NoError(t, err)
	for range 85 {
		w.IncStreamCount()
	}

	p.tryScaleUp(w)

	// Live config turned scaling off, so no scale-up may happen.
	assert.Never(t, func() bool { return len(p.pool.LoadConns()) > 1 }, 200*time.Millisecond, 5*time.Millisecond)
}

func TestWithGlobalLiveConnectionPoolProvider(t *testing.T) {
	opts := newTransportOptions([]TransportOption{
		WithGlobalLiveConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 3})),
	})
	require.NotNil(t, opts.poolConfigProvider)
	assert.Equal(t, 3, opts.poolConfigProvider().MaxConnections)
}

func TestWithOutboundLiveConnectionPoolProvider(t *testing.T) {
	opts := newTransportOptions([]TransportOption{
		WithOutboundLiveConnectionPoolProvider(func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider {
			return staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 3})
		}),
	})
	require.NotNil(t, opts.outboundPoolFactory)
	assert.Equal(t, 3, opts.outboundPoolFactory("x", nil)().MaxConnections)
}

func TestWithOutboundLiveConnectionPoolProvider_NotCalledOnNewDialer(t *testing.T) {
	called := false
	tr := NewTransport(WithOutboundLiveConnectionPoolProvider(func(string, *ClientConnectionPoolConfig) LiveConnectionPoolProvider {
		called = true
		return staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 3})
	}))
	_ = tr.NewDialer()
	assert.False(t, called, "outbound hook is YAML buildOutbound only, not NewDialer")
}

func TestOutboundLiveProvider_AppliesOnDialer(t *testing.T) {
	address := startTestServer(t)
	transport := NewTransport(
		MaxConnections(5),
		MinConnections(1),
		WithGlobalLiveConnectionPoolProvider(staticPoolProvider(ClientConnectionPoolConfig{MaxConnections: 7})),
	)
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{address}
	shared := transport.NewDialer(outboundLiveProvider(staticPoolProvider(ClientConnectionPoolConfig{
		MaxConnections: 20,
	})))
	sp, err := shared.RetainPeer(id, idSubscriber{1})
	require.NoError(t, err)
	sharedPeer := sp.(*grpcPeer)
	require.NotNil(t, sharedPeer.outboundLiveProvider)
	assert.Equal(t, 20, sharedPeer.livePoolCfg().MaxConnections, "dialer live provider replaces the transport hook")

	isolated := transport.NewDialer(outboundLiveProvider(staticPoolProvider(ClientConnectionPoolConfig{
		MaxConnections: 20,
	}))).WithConnectionIsolation()
	ip, err := isolated.RetainPeer(id, idSubscriber{2})
	require.NoError(t, err)
	isolatedPeer := ip.(*grpcPeer)
	require.NotNil(t, isolatedPeer.outboundLiveProvider)
	assert.Equal(t, 20, isolatedPeer.livePoolCfg().MaxConnections)
	assert.NotSame(t, sharedPeer, isolatedPeer)

	require.NoError(t, shared.ReleasePeer(id, idSubscriber{1}))
	require.NoError(t, isolated.ReleasePeer(id, idSubscriber{2}))
}

func TestNewPeer_OutboundOverrideValidatedAfterMerge(t *testing.T) {
	tr := NewTransport()

	_, err := tr.newPeer("127.0.0.1:1", &dialOptions{
		connPoolOverride: &ClientConnectionPoolConfig{MinConnections: 80},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minConnections (80)")

	_, err = tr.newPeer("127.0.0.1:1", &dialOptions{
		connPoolOverride: &ClientConnectionPoolConfig{ScaleDownGap: 0.8},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scaleDownGap")
}

func TestNewPeer_ProgrammaticMinGreaterThanMax(t *testing.T) {
	tr := NewTransport(MinConnections(100), MaxConnections(50))
	_, err := tr.newPeer("127.0.0.1:1", emptyDialOpts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxConnections (50) must be >= minConnections (100)")
}

func TestNewPeer_ProgrammaticMinExceedsDefaultMax(t *testing.T) {
	tr := NewTransport(MinConnections(80))
	_, err := tr.newPeer("127.0.0.1:1", emptyDialOpts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxConnections (50) must be >= minConnections (80)")
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
	assert.True(t, p.invalidLiveWarned.Load())
}
