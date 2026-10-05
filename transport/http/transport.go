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
	"context"
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http2"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/net/metrics"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/internal/peeraddr"
	"go.uber.org/yarpc/internal/tracinginterceptor"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
)

type transportOptions struct {
	keepAlive                  time.Duration
	maxIdleConns               int
	maxIdleConnsPerHost        int
	idleConnTimeout            time.Duration
	disableKeepAlives          bool
	disableCompression         bool
	responseHeaderTimeout      time.Duration
	connTimeout                time.Duration
	connBackoffStrategy        backoffapi.Strategy
	innocenceWindow            time.Duration
	dialContext                func(ctx context.Context, network, addr string) (net.Conn, error)
	jitter                     func(int64) int64
	tracer                     opentracing.Tracer
	tracingInterceptorEnabled  bool
	buildClient                func(*transportOptions) *http.Client
	logger                     *zap.Logger
	meter                      *metrics.Scope
	serviceName                string
	outboundTLSConfigProvider  yarpctls.OutboundTLSConfigProvider
	isolateOutboundConnections bool

	// connPool is the transport-wide HTTP/2 connection pool config every
	// peer starts from. See the pool options in conn_pool_config.go.
	connPool connpool.Config
	// poolConfigProvider is the global live pool provider. See
	// WithGlobalLiveConnectionPoolProvider.
	poolConfigProvider LiveConnectionPoolProvider
	// outboundPoolFactory builds per-outbound live providers for YAML
	// outbounds. See WithOutboundLiveConnectionPoolProvider.
	outboundPoolFactory func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider
}

var defaultTransportOptions = transportOptions{
	keepAlive:           30 * time.Second,
	maxIdleConnsPerHost: 2,
	connTimeout:         defaultConnTimeout,
	connBackoffStrategy: backoff.DefaultExponential,
	buildClient:         buildHTTPClient,
	innocenceWindow:     defaultInnocenceWindow,
	idleConnTimeout:     defaultIdleConnTimeout,
	jitter:              rand.Int63n,
	connPool:            defaultH2PoolConfig,
}

func newTransportOptions() transportOptions {
	options := defaultTransportOptions
	options.tracer = opentracing.GlobalTracer()
	return options
}

// TransportOption customizes the behavior of an HTTP transport.
type TransportOption func(*transportOptions)

func (TransportOption) httpOption() {}

// KeepAlive specifies the keep-alive period for the network connection. If
// zero, keep-alives are disabled.
//
// Defaults to 30 seconds.
func KeepAlive(t time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.keepAlive = t
	}
}

// MaxIdleConns controls the maximum number of idle (keep-alive) connections
// across all hosts. Zero means no limit.
func MaxIdleConns(i int) TransportOption {
	return func(options *transportOptions) {
		options.maxIdleConns = i
	}
}

// MaxIdleConnsPerHost specifies the number of idle (keep-alive) HTTP
// connections that will be maintained per host.
// Existing idle connections will be used instead of creating new HTTP
// connections.
//
// Defaults to 2 connections.
func MaxIdleConnsPerHost(i int) TransportOption {
	return func(options *transportOptions) {
		options.maxIdleConnsPerHost = i
	}
}

// IdleConnTimeout is the maximum amount of time an idle (keep-alive)
// connection will remain idle before closing itself.
// Zero means no limit.
//
// This applies to HTTP/1 connections only. Connections of an HTTP/2 outbound
// are closed by its connection pool instead (see ConnIdleTimeout), which never
// goes below MinConnections.
//
// Defaults to 15 minutes.
func IdleConnTimeout(t time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.idleConnTimeout = t
	}
}

// DisableKeepAlives prevents re-use of TCP connections between different HTTP
// requests.
func DisableKeepAlives() TransportOption {
	return func(options *transportOptions) {
		options.disableKeepAlives = true
	}
}

// DisableCompression if true prevents the Transport from requesting
// compression with an "Accept-Encoding: gzip" request header when the Request
// contains no existing Accept-Encoding value. If the Transport requests gzip
// on its own and gets a gzipped response, it's transparently decoded in the
// Response.Body. However, if the user explicitly requested gzip it is not
// automatically uncompressed.
func DisableCompression() TransportOption {
	return func(options *transportOptions) {
		options.disableCompression = true
	}
}

// ResponseHeaderTimeout if non-zero specifies the amount of time to wait for
// a server's response headers after fully writing the request (including its
// body, if any).  This time does not include the time to read the response
// body.
func ResponseHeaderTimeout(t time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.responseHeaderTimeout = t
	}
}

// ConnTimeout is the time that the transport will wait for a connection attempt.
// If a peer has been retained by a peer list, connection attempts are
// performed in a goroutine off the request path.
//
// The default is half a second.
func ConnTimeout(d time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.connTimeout = d
	}
}

// ConnBackoff specifies the connection backoff strategy for delays between
// connection attempts for each peer.
//
// The default is exponential backoff starting with 10ms fully jittered,
// doubling each attempt, with a maximum interval of 30s.
func ConnBackoff(s backoffapi.Strategy) TransportOption {
	return func(options *transportOptions) {
		options.connBackoffStrategy = s
	}
}

// InnocenceWindow is the duration after the peer connection management loop
// will suspend suspicion for a peer after successfully checking whether the
// peer is live with a fresh TCP connection.
//
// The default innocence window is 5 seconds.
//
// A timeout does not necessarily indicate that a peer is unavailable,
// but it could indicate that the connection is half-open, that the peer died
// without sending a TCP FIN packet.
// In this case, the peer connection management loop attempts to open a TCP
// connection in the background, once per innocence window, while suspicious of
// the connection, leaving the peer available until it fails.
func InnocenceWindow(d time.Duration) TransportOption {
	return func(options *transportOptions) {
		options.innocenceWindow = d
	}
}

// DialContext specifies the dial function for creating TCP connections on the
// outbound. This will override the default dial context, which has a 30 second
// timeout and respects the KeepAlive option.
//
// See https://golang.org/pkg/net/http/#Transport.DialContext for details.
func DialContext(f func(ctx context.Context, network, addr string) (net.Conn, error)) TransportOption {
	return func(options *transportOptions) {
		options.dialContext = f
	}
}

// Tracer configures a tracer for the transport and all its inbounds and
// outbounds.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(options *transportOptions) {
		options.tracer = tracer
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
	return func(options *transportOptions) {
		options.logger = logger
	}
}

// Meter sets a meter to use for internal transport metrics.
//
// The default is to not emit any metrics.
func Meter(meter *metrics.Scope) TransportOption {
	return func(options *transportOptions) {
		options.meter = meter
	}
}

// ServiceName sets the name of the service used in transport logging
// and metrics.
func ServiceName(name string) TransportOption {
	return func(options *transportOptions) {
		options.serviceName = name
	}
}

// OutboundTLSConfigProvider returns an TransportOption that provides the
// outbound TLS config provider.
func OutboundTLSConfigProvider(provider yarpctls.OutboundTLSConfigProvider) TransportOption {
	return func(options *transportOptions) {
		options.outboundTLSConfigProvider = provider
	}
}

// IsolateConnectionsPerOutbound makes every outbound this transport builds
// from a URL (NewSingleOutbound) or from configuration (TransportSpec) retain
// its own peers, and therefore its own connections or connection pools, instead
// of sharing them with other outbounds that resolve to the same address. It is
// the transport-wide form of retaining peers through an isolated Dialer (see
// Dialer.WithConnectionIsolation), for callers that build their outbounds
// through the transport or through configuration and so cannot supply a Dialer
// themselves.
//
// It does not affect outbounds built with NewOutbound from a peer chooser the
// caller created: that chooser already holds the transport or Dialer it
// retains peers through.
//
// Defaults to false: outbounds to the same address share a peer.
func IsolateConnectionsPerOutbound(enabled bool) TransportOption {
	return func(options *transportOptions) {
		options.isolateOutboundConnections = enabled
	}
}

// Hidden option to override the buildHTTPClient function. This is used only
// for testing.
func buildClient(f func(*transportOptions) *http.Client) TransportOption {
	return func(options *transportOptions) {
		options.buildClient = f
	}
}

// NewTransport creates a new HTTP transport for managing peers and sending requests
func NewTransport(opts ...TransportOption) *Transport {
	options := newTransportOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return options.newTransport()
}

func (o *transportOptions) newTransport() *Transport {
	logger := o.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	var (
		unaryInbounds   []interceptor.UnaryInbound
		unaryOutbounds  []interceptor.UnaryOutbound
		onewayInbounds  []interceptor.OnewayInbound
		onewayOutbounds []interceptor.OnewayOutbound
	)
	tracer := o.tracer
	if o.tracingInterceptorEnabled {
		ti := tracinginterceptor.New(tracinginterceptor.Params{
			Tracer:    tracer,
			Transport: TransportName,
		})
		unaryInbounds = append(unaryInbounds, ti)
		unaryOutbounds = append(unaryOutbounds, ti)
		onewayInbounds = append(onewayInbounds, ti)
		onewayOutbounds = append(onewayOutbounds, ti)

		tracer = opentracing.NoopTracer{}
	}
	return &Transport{
		once:                       lifecycle.NewOnce(),
		connTimeout:                o.connTimeout,
		connBackoffStrategy:        o.connBackoffStrategy,
		innocenceWindow:            o.innocenceWindow,
		jitter:                     o.jitter,
		peers:                      make(map[peerKey]*httpPeer),
		tracer:                     tracer,
		logger:                     logger,
		meter:                      o.meter,
		serviceName:                o.serviceName,
		ouboundTLSConfigProvider:   o.outboundTLSConfigProvider,
		unaryInboundInterceptor:    inboundmiddleware.UnaryChain(unaryInbounds...),
		unaryOutboundInterceptor:   unaryOutbounds,
		onewayInboundInterceptor:   inboundmiddleware.OnewayChain(onewayInbounds...),
		onewayOutboundInterceptor:  onewayOutbounds,
		isolateOutboundConnections: o.isolateOutboundConnections,
		h1Transport:                buildH1Transport(o),
		h2Transport:                buildH2Transport(o),
		h2PoolConfig:               o.connPool,
		h2PoolProvider:             o.poolConfigProvider,
		h2OutboundPoolFactory:      o.outboundPoolFactory,
		h2PoolMetrics: connpool.NewMetrics(connpool.MetricsParams{
			Meter:       o.meter,
			Logger:      logger,
			ServiceName: o.serviceName,
			Transport:   "http2",
			// MetricPrefix is left at its "conn_pool" default so this
			// transport's connection-pool metrics share the same metric
			// family as any other transport built on connpool.Pool (e.g. a
			// future transport/grpc migration onto this package) --
			// distinguished only by the "transport" tag above, not by a
			// different metric name.
		}),
		h2ActivePeers: newH2ActivePeersGauge(o.meter, logger, o.serviceName),
	}
}

// newH2ActivePeersGauge creates the gauge tracking how many peers currently
// hold a dedicated HTTP/2 connection pool, aggregated across all peers. It
// is created once here and shared by every peer (see
// getOrCreatePeer/releasePeer) rather than per peer, since
// go.uber.org/net/metrics errors on a second registration of the same
// metric name and tags.
//
// This is not tagged by peer (host:port) or connection scope, for the same
// cardinality reason connpool's metrics aren't: duplicate-peer identifiers
// or isolated Dialers (see Dialer.WithConnectionIsolation) can create many
// peers for what is logically one destination, and go.uber.org/net/metrics
// rejects re-registering the same series (transport/grpc's connection pool
// metrics hit exactly this failure mode when briefly tagged by peer
// address, and were redesigned to aggregate at the transport level
// instead).
func newH2ActivePeersGauge(meter *metrics.Scope, logger *zap.Logger, serviceName string) *metrics.Gauge {
	g, err := meter.Gauge(metrics.Spec{
		Name: "http2_peer_dedicated_transports",
		Help: "Number of peers with a dedicated HTTP/2 transport, aggregated across all peers.",
		ConstTags: metrics.Tags{
			"component": "yarpc",
			"service":   serviceName,
			"transport": "http",
		},
	})
	if err != nil {
		logger.Warn("failed to create http2 active peers gauge", zap.Error(err))
	}
	return g
}

func buildH1Transport(options *transportOptions) *http.Transport {
	dialContext := options.dialContext
	if dialContext == nil {
		dialContext = (&net.Dialer{
			Timeout:   defaultDialerTimeout,
			KeepAlive: options.keepAlive,
		}).DialContext
	}

	return &http.Transport{
		// options lifted from https://golang.org/src/net/http/transport.go
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          options.maxIdleConns,
		MaxIdleConnsPerHost:   options.maxIdleConnsPerHost,
		IdleConnTimeout:       options.idleConnTimeout,
		DisableKeepAlives:     options.disableKeepAlives,
		DisableCompression:    options.disableCompression,
		ResponseHeaderTimeout: options.responseHeaderTimeout,
	}
}

func buildH2Transport(options *transportOptions) *http2.Transport {
	dialContext := options.dialContext
	if dialContext == nil {
		dialContext = (&net.Dialer{
			Timeout:   defaultDialerTimeout,
			KeepAlive: options.keepAlive,
		}).DialContext
	}

	return &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return dialContext(ctx, network, addr)
		},
		DisableCompression: options.disableCompression,
		// IdleConnTimeout is deliberately left zero (never): this transport
		// only builds the pool's *http2.ClientConns, and the pool owns when
		// an idle connection is closed (ConnIdleTimeout, and never below
		// MinConnections). A ClientConn closing itself on its own idle timer
		// would drop the pool under its floor until the monitor re-dialed.
		PingTimeout:     defaultHTTP2PingTimeout,
		ReadIdleTimeout: defaultHTTP2ReadIdleTimeout,
	}
}

func buildHTTPClient(options *transportOptions) *http.Client {
	return &http.Client{
		Transport: buildH1Transport(options),
	}
}

// connectionScope identifies a set of peers that must not be shared with
// other dialers. The non-zero-sized type gives each allocation a distinct
// address.
type connectionScope byte

// peerKey identifies a peer in the transport's peer map.
//
// Peers are keyed by their full peer identifier, which may carry a
// duplicate-peer suffix and so is not always the address the peer dials (see
// peeraddr.Address). Dialers that opt into connection isolation (see
// Dialer.WithConnectionIsolation) add a stable scope to the key so they do
// not share peers, connections, or connection pools with other dialers.
type peerKey struct {
	identifier      string
	connectionScope *connectionScope
}

// Transport keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Transport struct {
	lock sync.Mutex
	once *lifecycle.Once

	peers map[peerKey]*httpPeer

	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	connectorsGroup     sync.WaitGroup
	innocenceWindow     time.Duration
	jitter              func(int64) int64

	tracer                    opentracing.Tracer
	logger                    *zap.Logger
	meter                     *metrics.Scope
	serviceName               string
	ouboundTLSConfigProvider  yarpctls.OutboundTLSConfigProvider
	unaryInboundInterceptor   interceptor.UnaryInbound
	unaryOutboundInterceptor  []interceptor.UnaryOutbound
	onewayInboundInterceptor  interceptor.OnewayInbound
	onewayOutboundInterceptor []interceptor.OnewayOutbound

	// isolateOutboundConnections makes outbounds built by the transport retain
	// peers through their own isolated Dialer. See
	// IsolateConnectionsPerOutbound.
	isolateOutboundConnections bool

	h1Transport *http.Transport
	// h2Transport is shared across all peers, but only as a *http2.ClientConn
	// factory (via dialH2Conn) and holder of the transport's configured
	// options (TLS/dial config, timeouts) -- its own internal connection
	// pooling is never used. Each httpPeer instead owns its own
	// *connpool.Pool[*http2.ClientConn] (see peer.go and h2PoolConfig below)
	// so that duplicate peers pointed at the same address end up with
	// independent HTTP/2 connections rather than sharing one.
	h2Transport *http2.Transport
	// h2PoolConfig is the transport-wide connpool.Config every peer's HTTP/2
	// pool starts from, before a Dialer's OutboundConnectionPool override
	// and any live provider are applied (see resolveH2PoolConfig and
	// httpPeer.poolConfig).
	h2PoolConfig connpool.Config
	// h2PoolProvider is the global live pool provider, used by every peer
	// whose Dialer has no provider of its own.
	h2PoolProvider LiveConnectionPoolProvider
	// h2OutboundPoolFactory builds the per-outbound live provider for
	// YAML-built outbounds (see buildOutbound).
	h2OutboundPoolFactory func(outbound string, yaml *ClientConnectionPoolConfig) LiveConnectionPoolProvider
	// releasedPoolWg tracks the HTTP/2 pools of peers released via
	// ReleasePeer. Release only stops a pool (asynchronously), so Stop joins
	// these to make sure no scaling monitor or connection watcher outlives
	// the transport. They cannot be waited on inside releasePeer: it runs
	// under a.lock, which other transport paths take.
	releasedPoolWg sync.WaitGroup
	// h2PoolMetrics holds the shared, transport-wide connection-pool metric
	// handles (active/draining/idle connection gauges, scale-event and
	// dial-outcome counters). It is created once here and each peer's pool
	// gets its own connpool.Reporter feeding into it (see h2Sender in
	// peer.go), so registration happens exactly once regardless of how many
	// peers - including duplicate peers - are created or recreated over the
	// Transport's lifetime. This is the same connpool.Metrics/Reporter
	// mechanism transport/grpc uses (or will use once it migrates onto
	// connpool), so the pool-health and dial metrics are reusable across
	// transports rather than reimplemented per transport.
	h2PoolMetrics *connpool.Metrics
	// h2ActivePeers tracks the one HTTP/2 metric connpool has no notion of:
	// how many peers currently hold a dedicated connection pool. See
	// newH2ActivePeersGauge for why it's created once here and shared by
	// every peer rather than per peer.
	h2ActivePeers *metrics.Gauge
}

var _ transport.Transport = (*Transport)(nil)

// Start starts the HTTP transport.
func (a *Transport) Start() error {
	return a.once.Start(func() error {
		return nil // Nothing to do
	})
}

// Stop stops the HTTP transport.
func (a *Transport) Stop() error {
	return a.once.Stop(func() error {
		a.h1Transport.CloseIdleConnections()
		a.stopPeerH2Pools()
		a.releasedPoolWg.Wait()
		a.connectorsGroup.Wait()
		return nil
	})
}

// stopPeerH2Pools stops every peer's HTTP/2 connection pool and waits for
// each to finish closing its connections and tearing down its background
// goroutines.
func (a *Transport) stopPeerH2Pools() {
	a.lock.Lock()
	peers := make([]*httpPeer, 0, len(a.peers))
	for _, p := range a.peers {
		peers = append(peers, p)
	}
	a.lock.Unlock()

	pools := make([]*connpool.Pool[*http2.ClientConn], 0, len(peers))
	for _, p := range peers {
		if pool := p.loadH2Pool(); pool != nil {
			pools = append(pools, pool)
		}
	}
	for _, pool := range pools {
		pool.Stop()
	}
	for _, pool := range pools {
		pool.Wait()
	}
}

// dialH2Conn dials a raw connection to addr using the transport's configured
// dial settings and wraps it as an *http2.ClientConn. Used as the Dial
// function for every peer's HTTP/2 connpool.Pool (see peer.go).
func (a *Transport) dialH2Conn(ctx context.Context, addr string) (*http2.ClientConn, error) {
	conn, err := a.h2Transport.DialTLSContext(ctx, "tcp", addr, nil)
	if err != nil {
		return nil, err
	}
	cc, err := a.h2Transport.NewClientConn(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return cc, nil
}

// IsRunning returns whether the HTTP transport is running.
func (a *Transport) IsRunning() bool {
	return a.once.IsRunning()
}

// RetainPeer gets or creates a Peer for the specified peer.Subscriber (usually a peer.Chooser)
func (a *Transport) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	return a.retainPeer(pid, emptyDialOpts, nil, sub)
}

func (a *Transport) retainPeer(pid peer.Identifier, options *dialOptions, connectionScope *connectionScope, sub peer.Subscriber) (peer.Peer, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	p, err := a.getOrCreatePeer(pid, options, connectionScope)
	if err != nil {
		return nil, err
	}
	p.Subscribe(sub)
	return p, nil
}

// resolveH2PoolConfig returns the startup pool config for a peer retained
// through a Dialer with the given options: the transport-wide config, overlaid
// field by field by the Dialer's OutboundConnectionPool override. It fails if
// the result is not a usable pool config, so that a bad option surfaces when
// the peer is retained rather than as a misbehaving scaler later.
func (a *Transport) resolveH2PoolConfig(options *dialOptions) (connpool.Config, error) {
	cfg := options.resolvedPoolConfig(a.h2PoolConfig)
	if err := cfg.Validate(); err != nil {
		return connpool.Config{}, err
	}
	return cfg, nil
}

// h2PoolProviderFor returns the live pool provider for a peer retained
// through a Dialer with the given options: the Dialer's own (per-outbound)
// provider when set, else the transport's global one. Nil means no live
// config.
func (a *Transport) h2PoolProviderFor(options *dialOptions) LiveConnectionPoolProvider {
	if options.poolConfigProvider != nil {
		return options.poolConfigProvider
	}
	return a.h2PoolProvider
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Transport) getOrCreatePeer(pid peer.Identifier, options *dialOptions, connectionScope *connectionScope) (*httpPeer, error) {
	key := peerKey{identifier: pid.Identifier(), connectionScope: connectionScope}
	if p, ok := a.peers[key]; ok {
		return p, nil
	}
	startupPool, err := a.resolveH2PoolConfig(options)
	if err != nil {
		return nil, err
	}
	realAddr := peeraddr.Address(pid.Identifier())
	p := newPeerWithPool(realAddr, a, startupPool, a.h2PoolProviderFor(options))
	a.peers[key] = p
	a.connectorsGroup.Add(1)
	go p.MaintainConn()

	return p, nil
}

// ReleasePeer releases a peer from the peer.Subscriber and removes that peer from the Transport if nothing is listening to it
func (a *Transport) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	return a.releasePeer(pid, nil, sub)
}

func (a *Transport) releasePeer(pid peer.Identifier, connectionScope *connectionScope, sub peer.Subscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	key := peerKey{identifier: pid.Identifier(), connectionScope: connectionScope}
	p, ok := a.peers[key]
	if !ok {
		return peer.ErrTransportHasNoReferenceToPeer{
			TransportName:  "http.Transport",
			PeerIdentifier: pid.Identifier(),
		}
	}

	if err := p.Unsubscribe(sub); err != nil {
		return err
	}

	if p.NumSubscribers() == 0 {
		delete(a.peers, key)
		p.Release()
		// Release only asks the peer's HTTP/2 pool to stop. Track its
		// teardown so Transport.Stop can wait for it.
		if pool := p.loadH2Pool(); pool != nil {
			a.releasedPoolWg.Add(1)
			go func() {
				defer a.releasedPoolWg.Done()
				pool.Wait()
			}()
		}
	}

	return nil
}
