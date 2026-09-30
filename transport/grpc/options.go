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
	"context"
	"crypto/tls"
	"math"
	"net"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	intbackoff "go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/internal/tracinginterceptor"
	"go.uber.org/yarpc/transport/internal/tls/dialer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	// defensive programming
	// these are copied from grpc-go but we set them explicitly here
	// in case these change in grpc-go so that yarpc stays consistent
	defaultServerMaxSendMsgSize = math.MaxInt32
	defaultClientMaxSendMsgSize = math.MaxInt32
	// Overriding default server and client maximum request and response
	// receive sizes to 64MB.
	defaultServerMaxRecvMsgSize = 1024 * 1024 * 64
	defaultClientMaxRecvMsgSize = 1024 * 1024 * 64
	// Client connection pool defaults.
	defaultClientConnPoolDynamicScalingEnabled  bool          = true
	defaultClientConnPoolMaxConcurrentStreams   int32         = 100
	defaultClientConnPoolScaleUpThreshold       float64       = 0.7
	defaultClientConnPoolScaleDownGap           float64       = 0.1
	defaultClientConnPoolMinConnections         int           = 1
	defaultClientConnPoolMaxConnections         int           = 50
	defaultClientConnPoolIdleTimeout            time.Duration = 5 * time.Minute
	defaultClientConnPoolScalingMonitorInterval time.Duration = 30 * time.Second
)

// Option is an interface shared by TransportOption, InboundOption, and OutboundOption
// allowing either to be recognized by TransportSpec().
type Option interface {
	grpcOption()
}

var _ Option = (TransportOption)(nil)
var _ Option = (InboundOption)(nil)
var _ Option = (OutboundOption)(nil)
var _ Option = (DialOption)(nil)

// TransportOption is an option for a transport.
type TransportOption func(*transportOptions)

func (TransportOption) grpcOption() {}

// ServiceName specifices the name of the service used in transport logging
// and metrics.
func ServiceName(name string) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serviceName = name
	}
}

// BackoffStrategy specifies the backoff strategy for delays between
// connection attempts for each peer.
//
// The default is exponential backoff starting with 10ms fully jittered,
// doubling each attempt, with a maximum interval of 30s.
func BackoffStrategy(backoffStrategy backoff.Strategy) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.backoffStrategy = backoffStrategy
	}
}

// Tracer specifies the tracer to use.
//
// By default, opentracing.GlobalTracer() is used.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.tracer = tracer
	}
}

// TracingInterceptorEnabled specifies whether to use the new tracing interceptor or the legacy implementation
func TracingInterceptorEnabled(enabled bool) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.tracingInterceptorEnabled = enabled
	}
}

// Logger sets a logger to use for internal logging.
//
// The default is to not write any logs.
func Logger(logger *zap.Logger) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.logger = logger
	}
}

// Meter sets a meter to use for transport metrics.
//
// The default is to not emit any metrics.
func Meter(meter *metrics.Scope) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.meter = meter
	}
}

// ServerMaxRecvMsgSize is the maximum message size the server can receive.
//
// The default is 4MB.
func ServerMaxRecvMsgSize(serverMaxRecvMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxRecvMsgSize = serverMaxRecvMsgSize
	}
}

// ServerMaxSendMsgSize is the maximum message size the server can send.
//
// The default is unlimited.
func ServerMaxSendMsgSize(serverMaxSendMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxSendMsgSize = serverMaxSendMsgSize
	}
}

// NumStreamWorkers sets the number of worker goroutines that should be used
// to process incoming streams on the gRPC server. Workers reuse goroutines
// with already-grown stacks, avoiding repeated stack allocations per request.
//
// If all workers are busy, the server falls back to spawning a new goroutine
// per stream (the default gRPC behavior), so there is no risk of starvation.
//
// A value of 0 (the default) disables the worker pool entirely.
func NumStreamWorkers(n uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.numStreamWorkers = &n
	}
}

// ServerMaxHeaderListSize returns a transport option for configuring maximum
// header list size the server must accept.
//
// The default is 16MB (gRPC default).
func ServerMaxHeaderListSize(serverMaxHeaderListSize uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverMaxHeaderListSize = &serverMaxHeaderListSize
	}
}

// ServerHeaderTableSize returns a transport option for configuring the
// HPACK dynamic table size for the server.
//
// The default is 4KB (HTTP/2 specification default).
func ServerHeaderTableSize(serverHeaderTableSize uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.serverHeaderTableSize = &serverHeaderTableSize
	}
}

// ClientMaxRecvMsgSize is the maximum message size the client can receive.
//
// The default is 4MB.
func ClientMaxRecvMsgSize(clientMaxRecvMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxRecvMsgSize = clientMaxRecvMsgSize
	}
}

// ClientMaxSendMsgSize is the maximum message size the client can send.
//
// The default is unlimited.
func ClientMaxSendMsgSize(clientMaxSendMsgSize int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxSendMsgSize = clientMaxSendMsgSize
	}
}

// ClientMaxHeaderListSize returns a transport option for configuring maximum
// header list size the client must accept.
//
// The default is 16MB (gRPC default).
func ClientMaxHeaderListSize(clientMaxHeaderListSize uint32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientMaxHeaderListSize = &clientMaxHeaderListSize
	}
}

// MaxConcurrentStreams sets the assumed HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS
// value enforced by the server.  YARPC uses this value to decide when to open
// an additional connection to a peer.
//
// The gRPC client does not expose the server-advertised limit, so this must be
// configured manually. The default is 100.
func MaxConcurrentStreams(n int32) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolMaxConcurrentStreams = n
	}
}

// ScaleUpThreshold sets the fraction of MaxConcurrentStreams at which YARPC
// opens an additional connection to a peer.  For example, a value of 0.7 means
// a new connection is opened when any existing connection carries more than
// 70% of its stream budget.
//
// The default is 0.7.
func ScaleUpThreshold(f float64) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolScaleUpThreshold = f
	}
}

// ScaleDownGap sets the hysteresis gap subtracted from ScaleUpThreshold to
// derive the scale-down threshold.  For example, with ScaleUpThreshold=0.7
// and ScaleDownGap=0.1, connections are drained only when aggregate load would
// fit within 60% of capacity on the reduced pool.  This prevents oscillation
// when stream counts hover near the scale-up boundary.
//
// The default is 0.1.
func ScaleDownGap(f float64) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolScaleDownGap = f
	}
}

// MinConnections sets the minimum number of connections YARPC maintains to
// each peer.  Connections are pre-established up to this count when the peer
// is first retained.
//
// The default is 1. Must be <= MaxConnections; newPeer rejects min > max.
func MinConnections(n int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolMinConnections = n
	}
}

// MaxConnections sets the maximum number of connections YARPC may open to a
// single peer.
//
// The default is 50. Must be >= MinConnections; newPeer rejects min > max.
func MaxConnections(n int) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolMaxConnections = n
	}
}

// ConnIdleTimeout sets how long a fully-drained connection remains idle before
// YARPC closes it and removes it from the pool.
//
// The default is 5 minutes.
func ConnIdleTimeout(d time.Duration) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolIdleTimeout = d
	}
}

// ScalingMonitorInterval sets how often the background monitor goroutine
// evaluates the connection pool for scale-down and idle cleanup.
//
// The default is 30 seconds. Values under 30s are clamped to 30s. A live
// provider may change the interval; the running monitor picks it up on the
// next wait without restarting.
func ScalingMonitorInterval(d time.Duration) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolScalingMonitorInterval = d
	}
}

// WithDynamicConnectionScaling enables or disables automatic gRPC connection
// pool scaling based on stream utilization.
//
// When enabled, YARPC monitors stream usage on each pooled connection and
// automatically opens new connections when any connection exceeds
// ScaleUpThreshold, and drains connections when aggregate utilization drops
// low enough to consolidate load onto fewer connections.
//
// The default is true (enabled). An omitted YAML field leaves this value in
// place. dynamicScalingEnabled: true or false in YAML overrides it.
//
// If this is false (startup or live), the pool does not grow. The monitor
// still winds extra connections down to 1 via drain → idle → close so
// in-flight RPCs finish. A later live true can fill minConnections again.
// Scale-up on RPC also starts the monitor if live config enables scaling.
func WithDynamicConnectionScaling(enabled bool) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.clientConnPoolDynamicScalingEnabled = enabled
	}
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

// WithGlobalLiveConnectionPoolProvider installs a live provider for the
// transport-wide pool (every peer unless that outbound has its own provider).
// The scaler reads the provider on every scale-up and monitor tick.
//
// scalingMonitorInterval is live: the monitor rereads it after each pass and
// waits that long (still clamped to 30s). Changing it does not restart the
// goroutine.
func WithGlobalLiveConnectionPoolProvider(p LiveConnectionPoolProvider) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.poolConfigProvider = p
	}
}

// WithOutboundLiveConnectionPoolProvider installs the per-outbound live hook
// for YAML-built gRPC outbounds. YARPC calls f once when building each
// outbound. The returned LiveConnectionPoolProvider is stored on that
// outbound's Dialer and replaces the global hook for those peers. Return nil
// to keep the global hook for that outbound.
//
// outbound is Kit.OutboundServiceName() (dest). After Kit.OutboundKey exists
// (https://github.com/yarpc/yarpc-go/pull/2573), that map key is used instead.
// yaml is the outbound's clientConnectionPool block, or nil if omitted.
//
// Programmatic NewDialer does not call f. This is the only outbound-specific
// live-hook API; there is no separate DialOption.
func WithOutboundLiveConnectionPoolProvider(f func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider) TransportOption {
	return func(transportOptions *transportOptions) {
		transportOptions.outboundPoolFactory = f
	}
}

// InboundOption is an option for an inbound.
type InboundOption func(*inboundOptions)

func (InboundOption) grpcOption() {}

// InboundCredentials returns an InboundOption that sets credentials for incoming
// connections.
func InboundCredentials(creds credentials.TransportCredentials) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.creds = creds
	}
}

// InboundTLSConfiguration returns an InboundOption that provides the TLS confiugration
// used for setting up TLS inbound.
func InboundTLSConfiguration(config *tls.Config) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.tlsConfig = config
	}
}

// InboundTLSMode returns an InboundOption that sets inbound TLS mode.
// It must be noted that TLS configuration must be passed separately using inbound
// option InboundTLSConfiguration.
func InboundTLSMode(mode yarpctls.Mode) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.tlsMode = mode
	}
}

// OutboundOption is an option for an outbound.
type OutboundOption func(*outboundOptions)

func (OutboundOption) grpcOption() {}

// OutboundTLSConfigProvider returns an OutboundOption that provides the
// outbound TLS config provider.
func OutboundTLSConfigProvider(provider yarpctls.OutboundTLSConfigProvider) OutboundOption {
	return func(outboundOptions *outboundOptions) {
		outboundOptions.tlsConfigProvider = provider
	}
}

// OutboundCompressor returns an OutboundOption that applies compressorion
// for requests in the outbound.
func OutboundCompressor(compressor transport.Compressor) OutboundOption {
	return func(outboundOptions *outboundOptions) {
		if compressor != nil {
			outboundOptions.compressor = compressor.Name()
		}
	}
}

// DialOption is an option that influences grpc.Dial.
type DialOption func(*dialOptions)

func (DialOption) grpcOption() {}

// DialerCredentials returns a DialOption which configures a
// connection level security credentials (e.g., TLS/SSL).
func DialerCredentials(creds credentials.TransportCredentials) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.creds = creds
	}
}

// DialerTLSConfig return a DialOption which configures the TLS config for the
// outbound.
func DialerTLSConfig(config *tls.Config) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.tlsConfig = config
	}
}

// DialerDestinationServiceName returns a DialOption which configures the
// destination service name of the dialer. This is used in TLS dialer metrics.
func DialerDestinationServiceName(service string) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.destServiceName = service
	}
}

// ContextDialer sets the dialer for creating outbound connections.
//
// See https://godoc.org/google.golang.org/grpc#WithContextDialer for more
// details.
func ContextDialer(f func(context.Context, string) (net.Conn, error)) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.contextDialer = f
	}
}

// Compressor sets the compressor to be used by default for gRPC connections
func Compressor(compressor transport.Compressor) DialOption {
	return func(dialOptions *dialOptions) {
		if compressor != nil {
			// We assume that the grpc-go compressor was also globally
			// registered and just use the name.
			// Future implementations may elect to actually use the compressor.
			dialOptions.defaultCompressor = compressor.Name()
		}
	}
}

// KeepaliveParams sets the gRPC keepalive parameters of the outbound
// connection.
// See https://pkg.go.dev/google.golang.org/grpc#WithKeepaliveParams for more
// details.
func KeepaliveParams(params keepalive.ClientParameters) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.keepaliveParams = &params
	}
}

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
	return func(dialOptions *dialOptions) {
		dialOptions.connPoolOverride = &cfg
	}
}

// outboundLiveProvider stores a live provider on a Dialer. YAML buildOutbound
// is the only caller; WithOutboundLiveConnectionPoolProvider is the public API.
func outboundLiveProvider(p LiveConnectionPoolProvider) DialOption {
	return func(dialOptions *dialOptions) {
		dialOptions.poolConfigProvider = p
	}
}

type transportOptions struct {
	backoffStrategy           backoff.Strategy
	tracer                    opentracing.Tracer
	tracingInterceptorEnabled bool
	logger                    *zap.Logger
	meter                     *metrics.Scope
	serviceName               string
	serverMaxRecvMsgSize      int
	serverMaxSendMsgSize      int
	clientMaxRecvMsgSize      int
	clientMaxSendMsgSize      int
	serverMaxHeaderListSize   *uint32
	serverHeaderTableSize     *uint32
	clientMaxHeaderListSize   *uint32
	numStreamWorkers          *uint32
	unaryInboundInterceptor   interceptor.UnaryInbound
	unaryOutboundInterceptor  []interceptor.UnaryOutbound
	streamInboundInterceptor  interceptor.StreamInbound
	streamOutboundInterceptor []interceptor.StreamOutbound
	// Client connection pool options.
	clientConnPoolMaxConcurrentStreams   int32
	clientConnPoolScaleUpThreshold       float64
	clientConnPoolScaleDownGap           float64
	clientConnPoolMinConnections         int
	clientConnPoolMaxConnections         int
	clientConnPoolIdleTimeout            time.Duration
	clientConnPoolScalingMonitorInterval time.Duration
	clientConnPoolDynamicScalingEnabled  bool
	poolConfigProvider                   LiveConnectionPoolProvider
	outboundPoolFactory                  func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider
}

func newTransportOptions(options []TransportOption) *transportOptions {
	transportOptions := &transportOptions{
		backoffStrategy:      intbackoff.DefaultExponential,
		serverMaxRecvMsgSize: defaultServerMaxRecvMsgSize,
		serverMaxSendMsgSize: defaultServerMaxSendMsgSize,
		clientMaxRecvMsgSize: defaultClientMaxRecvMsgSize,
		clientMaxSendMsgSize: defaultClientMaxSendMsgSize,
		// Client connection pool defaults.
		clientConnPoolMaxConcurrentStreams:   defaultClientConnPoolMaxConcurrentStreams,
		clientConnPoolScaleUpThreshold:       defaultClientConnPoolScaleUpThreshold,
		clientConnPoolScaleDownGap:           defaultClientConnPoolScaleDownGap,
		clientConnPoolMinConnections:         defaultClientConnPoolMinConnections,
		clientConnPoolMaxConnections:         defaultClientConnPoolMaxConnections,
		clientConnPoolIdleTimeout:            defaultClientConnPoolIdleTimeout,
		clientConnPoolScalingMonitorInterval: defaultClientConnPoolScalingMonitorInterval,
		clientConnPoolDynamicScalingEnabled:  defaultClientConnPoolDynamicScalingEnabled,
	}
	for _, option := range options {
		option(transportOptions)
	}
	if transportOptions.logger == nil {
		transportOptions.logger = zap.NewNop()
	}
	if transportOptions.tracer == nil {
		transportOptions.tracer = opentracing.GlobalTracer()
	}
	var (
		unaryInbounds  []interceptor.UnaryInbound
		streamInbounds []interceptor.StreamInbound
	)
	if transportOptions.tracingInterceptorEnabled {
		ti := tracinginterceptor.New(tracinginterceptor.Params{
			Tracer:    transportOptions.tracer,
			Transport: TransportName,
		})
		unaryInbounds = append(unaryInbounds, ti)
		streamInbounds = append(streamInbounds, ti)
		transportOptions.unaryOutboundInterceptor = []interceptor.UnaryOutbound{ti}
		transportOptions.streamOutboundInterceptor = []interceptor.StreamOutbound{ti}
		transportOptions.tracer = opentracing.NoopTracer{}
	}

	transportOptions.unaryInboundInterceptor = inboundmiddleware.UnaryChain(unaryInbounds...)
	transportOptions.streamInboundInterceptor = inboundmiddleware.StreamChain(streamInbounds...)
	return transportOptions
}

type inboundOptions struct {
	creds credentials.TransportCredentials

	tlsConfig *tls.Config
	tlsMode   yarpctls.Mode
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}

type outboundOptions struct {
	compressor        string
	tlsConfigProvider yarpctls.OutboundTLSConfigProvider
}

func newOutboundOptions(options []OutboundOption) *outboundOptions {
	outboundOptions := &outboundOptions{}
	for _, option := range options {
		option(outboundOptions)
	}
	return outboundOptions
}

type dialOptions struct {
	creds             credentials.TransportCredentials
	contextDialer     func(context.Context, string) (net.Conn, error)
	defaultCompressor string
	keepaliveParams   *keepalive.ClientParameters
	tlsConfig         *tls.Config
	destServiceName   string

	// connPoolOverride, when set, overrides the transport-wide dynamic
	// connection pool configuration for peers retained through this Dialer.
	// See OutboundConnectionPool.
	connPoolOverride *ClientConnectionPoolConfig

	// poolConfigProvider, when set, is the live provider for peers retained
	// through this Dialer. YAML outbounds get this from
	// WithOutboundLiveConnectionPoolProvider.
	poolConfigProvider LiveConnectionPoolProvider
}

func (d *dialOptions) grpcOptions(t *Transport) []grpc.DialOption {
	credsOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	if d.creds != nil {
		credsOption = grpc.WithTransportCredentials(d.creds)
	}

	opts := []grpc.DialOption{
		credsOption,
	}

	if d.defaultCompressor != "" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(d.defaultCompressor)))
	}

	if d.keepaliveParams != nil {
		opts = append(opts, grpc.WithKeepaliveParams(*d.keepaliveParams))
	}

	contextDialer := d.contextDialer
	if d.tlsConfig != nil {
		params := dialer.Params{
			Config:        d.tlsConfig,
			Meter:         t.options.meter,
			Logger:        t.options.logger,
			ServiceName:   t.options.serviceName,
			TransportName: TransportName,
			Dest:          d.destServiceName,
		}

		if d.contextDialer != nil {
			params.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.contextDialer(ctx, addr)
			}
		}
		tlsDialer := dialer.NewTLSDialer(params)
		contextDialer = func(ctx context.Context, addr string) (net.Conn, error) {
			return tlsDialer.DialContext(ctx, "tcp", addr)
		}
	}
	opts = append(opts, grpc.WithContextDialer(contextDialer))

	return opts
}

func newDialOptions(options []DialOption) *dialOptions {
	var dopts dialOptions
	for _, option := range options {
		option(&dopts)
	}
	return &dopts
}

// resolvedPoolConfig returns the connPoolConfig to use for a peer retained
// through this Dialer: the transport-wide base config, overridden field by
// field by connPoolOverride when set.
//
// Fields left at their zero value in the override are inherited from base,
// mirroring how TransportConfig.ClientConnectionPool fields are applied over
// programmatic TransportOption defaults in buildTransport.
func (d *dialOptions) resolvedPoolConfig(base connPoolConfig) connPoolConfig {
	if d.connPoolOverride == nil {
		return base
	}
	return applyPoolOverride(base, d.connPoolOverride)
}
