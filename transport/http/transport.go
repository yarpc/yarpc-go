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
	"go.uber.org/yarpc/internal/tracinginterceptor"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

type transportOptions struct {
	keepAlive                 time.Duration
	maxIdleConns              int
	maxIdleConnsPerHost       int
	idleConnTimeout           time.Duration
	disableKeepAlives         bool
	disableCompression        bool
	responseHeaderTimeout     time.Duration
	connTimeout               time.Duration
	connBackoffStrategy       backoffapi.Strategy
	innocenceWindow           time.Duration
	dialContext               func(ctx context.Context, network, addr string) (net.Conn, error)
	jitter                    func(int64) int64
	tracer                    opentracing.Tracer
	tracingInterceptorEnabled bool
	buildClient               func(*transportOptions) *http.Client
	logger                    *zap.Logger
	meter                     *metrics.Scope
	serviceName               string
	outboundTLSConfigProvider yarpctls.OutboundTLSConfigProvider
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
		once:                      lifecycle.NewOnce(),
		connTimeout:               o.connTimeout,
		connBackoffStrategy:       o.connBackoffStrategy,
		innocenceWindow:           o.innocenceWindow,
		jitter:                    o.jitter,
		peers:                     make(map[string]*httpPeer),
		tracer:                    tracer,
		logger:                    logger,
		meter:                     o.meter,
		serviceName:               o.serviceName,
		ouboundTLSConfigProvider:  o.outboundTLSConfigProvider,
		unaryInboundInterceptor:   inboundmiddleware.UnaryChain(unaryInbounds...),
		unaryOutboundInterceptor:  unaryOutbounds,
		onewayInboundInterceptor:  inboundmiddleware.OnewayChain(onewayInbounds...),
		onewayOutboundInterceptor: onewayOutbounds,
		h1Transport:               buildH1Transport(o),
		h2Transport:               buildH2Transport(o),
	}
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
		IdleConnTimeout:    options.idleConnTimeout,
		PingTimeout:        defaultHTTP2PingTimeout,
		ReadIdleTimeout:    defaultHTTP2ReadIdleTimeout,
	}
}

func buildHTTPClient(options *transportOptions) *http.Client {
	return &http.Client{
		Transport: buildH1Transport(options),
	}
}

// Transport keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Transport struct {
	lock sync.Mutex
	once *lifecycle.Once

	peers map[string]*httpPeer

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

	h1Transport *http.Transport
	h2Transport *http2.Transport
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
		a.h2Transport.CloseIdleConnections()
		a.connectorsGroup.Wait()
		return nil
	})
}

// IsRunning returns whether the HTTP transport is running.
func (a *Transport) IsRunning() bool {
	return a.once.IsRunning()
}

// RetainPeer gets or creates a Peer for the specified peer.Subscriber (usually a peer.Chooser)
func (a *Transport) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	p := a.getOrCreatePeer(pid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Transport) getOrCreatePeer(pid peer.Identifier) *httpPeer {
	addr := pid.Identifier()
	if p, ok := a.peers[addr]; ok {
		return p
	}
	p := newPeer(addr, a)
	a.peers[addr] = p
	a.connectorsGroup.Add(1)
	go p.MaintainConn()

	return p
}

// ReleasePeer releases a peer from the peer.Subscriber and removes that peer from the Transport if nothing is listening to it
func (a *Transport) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	p, ok := a.peers[pid.Identifier()]
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
		delete(a.peers, pid.Identifier())
		p.Release()
	}

	return nil
}
