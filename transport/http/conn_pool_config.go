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
	"time"

	"go.uber.org/yarpc/transport/internal/connpool"
)

// ClientConnectionPoolConfig configures the dynamic HTTP/2 connection pool
// that every peer of an HTTP/2 outbound (see UseHTTP2) owns. It has no effect
// on HTTP/1.x outbounds.
//
// Set it under transports.http for every outbound, or under a single
// outbound's http section to override the transport-wide values for that
// outbound only (see OutboundConfig.ClientConnectionPool).
//
//	transports:
//	  http:
//	    clientConnectionPool:
//	      dynamicScalingEnabled: true   # default; false stops growth and drains extras to 1
//	      maxConcurrentStreams: 100     # assumed server HTTP/2 stream limit
//	      scaleUpThreshold: 0.7         # open a new conn at 70% utilization
//	      scaleDownGap: 0.1             # hysteresis gap below scaleUpThreshold for drain decisions
//	      minConnections: 1             # minimum connections per peer
//	      maxConnections: 50            # maximum connections per peer
//	      idleTimeout: 5m               # close idle connections after this duration
//	      scalingMonitorInterval: 30s   # how often to evaluate scale-down and idle cleanup
type ClientConnectionPoolConfig struct {
	// DynamicScalingEnabled controls the background connection scaling monitor.
	// The transport default is enabled. This field is a pointer because a plain
	// bool left out of YAML decodes as false and would turn scaling off for
	// every service that omits the field.
	//
	//   - nil (field omitted): no opinion. The programmatic option is kept,
	//     or the default (enabled) when that option was not passed.
	//   - false: the pool does not grow; extras drain to 1 (drain → idle → close).
	//   - true: scaling is turned on.
	//
	// When the field is set, it overrides WithDynamicConnectionScaling, same
	// as the other clientConnectionPool fields.
	DynamicScalingEnabled *bool `config:"dynamicScalingEnabled"`

	// MaxConcurrentStreams is the assumed HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS
	// value enforced by the server. YARPC uses this to decide when to open an
	// additional connection. Defaults to 100.
	MaxConcurrentStreams int32 `config:"maxConcurrentStreams"`

	// ScaleUpThreshold is the fraction of MaxConcurrentStreams at which a
	// new connection is opened (e.g. 0.7 → scale up at 70 active streams when
	// MaxConcurrentStreams is 100).
	// Must be in the range (0, 1]. Defaults to 0.7.
	ScaleUpThreshold float64 `config:"scaleUpThreshold"`

	// ScaleDownGap is the hysteresis gap subtracted from ScaleUpThreshold to
	// derive the scale-down threshold. Prevents oscillation when stream counts
	// hover near the scale-up boundary. Defaults to 0.1.
	ScaleDownGap float64 `config:"scaleDownGap"`

	// MinConnections is the minimum number of connections kept per peer.
	// The first connection is dialed on the peer's first request; the
	// scaling monitor then fills the pool up to this count. Defaults to 1.
	MinConnections int `config:"minConnections"`

	// MaxConnections is the maximum number of connections allowed per peer.
	// Defaults to 50.
	MaxConnections int `config:"maxConnections"`

	// IdleTimeout is how long a fully-drained connection stays idle before
	// YARPC closes it. Defaults to 5 minutes.
	IdleTimeout time.Duration `config:"idleTimeout"`

	// ScalingMonitorInterval is how often the background monitor evaluates
	// the pool for scale-down and idle cleanup.
	// Defaults to 30 seconds. Values under 30s are clamped to 30s.
	// The running monitor rereads this after each pass, including live overlays.
	ScalingMonitorInterval time.Duration `config:"scalingMonitorInterval"`
}

// override converts the public config into the transport-agnostic form the
// shared connpool package resolves and validates.
func (c ClientConnectionPoolConfig) override() connpool.Override {
	return connpool.Override{
		DynamicScalingEnabled:  c.DynamicScalingEnabled,
		MaxConcurrentStreams:   c.MaxConcurrentStreams,
		ScaleUpThreshold:       c.ScaleUpThreshold,
		ScaleDownGap:           c.ScaleDownGap,
		MinConnections:         c.MinConnections,
		MaxConnections:         c.MaxConnections,
		IdleTimeout:            c.IdleTimeout,
		ScalingMonitorInterval: c.ScalingMonitorInterval,
	}
}

// validateClientConnectionPoolConfig checks the values set in a YAML or
// DialOption pool block. Zero values mean "inherit" and are accepted; the
// resolved result is validated separately, once it is merged onto the
// transport-wide config.
func validateClientConnectionPoolConfig(cp ClientConnectionPoolConfig) error {
	return cp.override().Validate()
}

// LiveConnectionPoolProvider is a live provider that returns pool overrides
// after peer creation. It must be safe for concurrent calls from the RPC
// and monitor paths.
//
// There are two installers:
//   - WithGlobalLiveConnectionPoolProvider — every peer on the transport
//   - WithOutboundLiveConnectionPoolProvider — one YAML outbound (replaces
//     the global hook for that outbound's peers)
//
// Zero-value fields mean "no opinion" and keep the peer's startup config
// (TransportOptions, YAML, and OutboundConnectionPool). A set
// DynamicScalingEnabled replaces the startup flag. Invalid resolved configs
// are ignored and the last valid snapshot is kept.
//
// The live provider returns ClientConnectionPoolConfig only. Callers that
// subscribe to an external config system resolve that system themselves.
type LiveConnectionPoolProvider func() ClientConnectionPoolConfig

// toOverrideFunc adapts a LiveConnectionPoolProvider to the shared package's
// provider type. A nil provider stays nil so LiveConfig.HasProvider is accurate.
func (p LiveConnectionPoolProvider) toOverrideFunc() func() connpool.Override {
	if p == nil {
		return nil
	}
	return func() connpool.Override { return p().override() }
}

// MaxConcurrentStreams sets the assumed HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS
// value enforced by the server. YARPC uses this value to decide when to open
// an additional connection to a peer.
//
// The value is not read from the server's SETTINGS frame, so it must be
// configured manually, matching the gRPC transport. The default is 100.
//
// Like every connection pool option, this only affects HTTP/2 outbounds (see
// UseHTTP2).
func MaxConcurrentStreams(n int32) TransportOption {
	return func(options *transportOptions) {
		options.connPool.MaxConcurrentStreams = n
	}
}

// ScaleUpThreshold sets the fraction of MaxConcurrentStreams at which YARPC
// opens an additional connection to a peer. For example, a value of 0.7 means
// a new connection is opened when every existing connection carries at least
// 70% of its stream budget.
//
// The default is 0.7.
func ScaleUpThreshold(f float64) TransportOption {
	return func(options *transportOptions) {
		options.connPool.ScaleUpThreshold = f
	}
}

// ScaleDownGap sets the hysteresis gap subtracted from ScaleUpThreshold to
// derive the scale-down threshold. For example, with ScaleUpThreshold=0.7
// and ScaleDownGap=0.1, connections are drained only when aggregate load would
// fit within 60% of capacity on the reduced pool. This prevents oscillation
// when stream counts hover near the scale-up boundary.
//
// The default is 0.1.
func ScaleDownGap(f float64) TransportOption {
	return func(options *transportOptions) {
		options.connPool.ScaleDownGap = f
	}
}

// MinConnections sets the minimum number of connections YARPC maintains to
// each peer. The first connection is dialed on the peer's first request; the
// scaling monitor then fills the pool up to this count.
//
// The default is 1. Must be <= MaxConnections; retaining a peer fails
// otherwise.
func MinConnections(n int) TransportOption {
	return func(options *transportOptions) {
		options.connPool.MinConnections = n
	}
}

// MaxConnections sets the maximum number of connections YARPC may open to a
// single peer.
//
// The default is 50. Must be >= MinConnections; retaining a peer fails
// otherwise.
func MaxConnections(n int) TransportOption {
	return func(options *transportOptions) {
		options.connPool.MaxConnections = n
	}
}

// ConnIdleTimeout sets how long a fully-drained connection remains idle before
// YARPC closes it and removes it from the pool. It is distinct from
// IdleConnTimeout, which bounds how long any connection may sit without
// requests before the HTTP client closes it.
//
// The default is 5 minutes.
func ConnIdleTimeout(d time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.connPool.IdleTimeout = d
	}
}

// ScalingMonitorInterval sets how often the background monitor goroutine
// evaluates the connection pool for scale-down and idle cleanup.
//
// The default is 30 seconds. Values under 30s are clamped to 30s. A live
// provider may change the interval; the running monitor picks it up on the
// next wait without restarting.
func ScalingMonitorInterval(d time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.connPool.ScalingMonitorInterval = d
	}
}

// WithDynamicConnectionScaling enables or disables automatic HTTP/2
// connection pool scaling based on stream utilization.
//
// When enabled, YARPC monitors stream usage on each pooled connection and
// automatically opens new connections when every connection exceeds
// ScaleUpThreshold, and drains connections when aggregate utilization drops
// low enough to consolidate load onto fewer connections.
//
// The default is true (enabled). An omitted YAML field leaves this value in
// place. dynamicScalingEnabled: true or false in YAML overrides it.
//
// If this is false (startup or live), the pool does not grow. The monitor
// still winds extra connections down to 1 via drain → idle → close so
// in-flight requests finish. A later live true can fill minConnections again.
// Scale-up on a request also starts the monitor if live config enables
// scaling.
func WithDynamicConnectionScaling(enabled bool) TransportOption {
	return func(options *transportOptions) {
		options.connPool.DynamicScalingEnabled = enabled
	}
}

// WithGlobalLiveConnectionPoolProvider installs a live provider for the
// transport-wide pool (every peer unless that outbound has its own provider).
// The scaler reads the provider on every scale-up and monitor tick.
//
// scalingMonitorInterval is live: the monitor rereads it after each pass and
// waits that long (still clamped to 30s). Changing it does not restart the
// goroutine.
func WithGlobalLiveConnectionPoolProvider(p LiveConnectionPoolProvider) TransportOption {
	return func(options *transportOptions) {
		options.poolConfigProvider = p
	}
}

// WithOutboundLiveConnectionPoolProvider installs the per-outbound live hook
// for YAML-built HTTP outbounds. YARPC calls f once when building each
// outbound. The returned LiveConnectionPoolProvider is stored on that
// outbound's Dialer and replaces the global hook for those peers. Return nil
// to keep the global hook for that outbound.
//
// outbound is Kit.OutboundKey() (the yarpc.outbounds map key). yaml is the
// outbound's clientConnectionPool block, or nil if omitted.
//
// Programmatic NewDialer does not call f. This is the only outbound-specific
// live-hook API; there is no separate DialOption.
func WithOutboundLiveConnectionPoolProvider(f func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider) TransportOption {
	return func(options *transportOptions) {
		options.outboundPoolFactory = f
	}
}

// DialOption customizes the peers a Dialer retains.
type DialOption func(*dialOptions)

func (DialOption) httpOption() {}

// OutboundConnectionPool returns a DialOption that overrides the transport's
// shared dynamic connection pool configuration (see
// TransportConfig.ClientConnectionPool and the MaxConcurrentStreams /
// ScaleUpThreshold / ScaleDownGap / MinConnections / MaxConnections /
// ConnIdleTimeout / ScalingMonitorInterval / WithDynamicConnectionScaling
// TransportOptions) for peers retained through this Dialer.
//
// Applies to peers retained through this Dialer. Use WithConnectionIsolation
// so two outbounds do not share a peer (and therefore this override). Any
// field left at its zero value falls back to the transport-wide configuration.
func OutboundConnectionPool(cfg ClientConnectionPoolConfig) DialOption {
	return func(options *dialOptions) {
		options.connPoolOverride = &cfg
	}
}

// outboundLiveProvider stores a live provider on a Dialer. YAML buildOutbound
// is the only caller; WithOutboundLiveConnectionPoolProvider is the public API.
func outboundLiveProvider(p LiveConnectionPoolProvider) DialOption {
	return func(options *dialOptions) {
		options.poolConfigProvider = p
	}
}

type dialOptions struct {
	// connPoolOverride, when set, overrides the transport-wide dynamic
	// connection pool configuration for peers retained through this Dialer.
	// See OutboundConnectionPool.
	connPoolOverride *ClientConnectionPoolConfig

	// poolConfigProvider, when set, is the live provider for peers retained
	// through this Dialer. YAML outbounds get this from
	// WithOutboundLiveConnectionPoolProvider.
	poolConfigProvider LiveConnectionPoolProvider
}

var emptyDialOpts = &dialOptions{}

func newDialOptions(options []DialOption) *dialOptions {
	var dopts dialOptions
	for _, option := range options {
		option(&dopts)
	}
	return &dopts
}

// resolvedPoolConfig returns the pool config to use for a peer retained
// through this Dialer: the transport-wide base config, overridden field by
// field by connPoolOverride when set.
//
// Fields left at their zero value in the override are inherited from base,
// mirroring how TransportConfig.ClientConnectionPool fields are applied over
// programmatic TransportOption defaults in buildTransport.
func (d *dialOptions) resolvedPoolConfig(base connpool.Config) connpool.Config {
	if d.connPoolOverride == nil {
		return base
	}
	o := d.connPoolOverride.override()
	return base.Apply(&o)
}
