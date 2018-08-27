// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpchttp

import (
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/zap"
)

type dialerOptions struct {
	keepAlive             time.Duration
	maxIdleConns          int
	maxIdleConnsPerHost   int
	idleConnTimeout       time.Duration
	disableKeepAlives     bool
	disableCompression    bool
	responseHeaderTimeout time.Duration
	connTimeout           time.Duration
	connBackoffStrategy   backoffapi.Strategy
	innocenceWindow       time.Duration
	jitter                func(int64) int64
	tracer                opentracing.Tracer
	buildClient           func(*dialerOptions) *http.Client
	logger                *zap.Logger
}

var defaultDialerOptions = dialerOptions{
	keepAlive:           30 * time.Second,
	maxIdleConnsPerHost: 2,
	connTimeout:         defaultConnTimeout,
	connBackoffStrategy: backoff.DefaultExponential,
	buildClient:         buildHTTPClient,
	innocenceWindow:     defaultInnocenceWindow,
	jitter:              rand.Int63n,
}

func newDialerOptions() dialerOptions {
	options := defaultDialerOptions
	options.tracer = opentracing.GlobalTracer()
	return options
}

// DialerOption customizes the behavior of an HTTP dialer.
type DialerOption func(*dialerOptions)

func (DialerOption) httpOption() {}

// KeepAlive specifies the keep-alive period for the network connection. If
// zero, keep-alives are disabled.
//
// Defaults to 30 seconds.
func KeepAlive(t time.Duration) DialerOption {
	return func(options *dialerOptions) {
		options.keepAlive = t
	}
}

// MaxIdleConns controls the maximum number of idle (keep-alive) connections
// across all hosts. Zero means no limit.
func MaxIdleConns(i int) DialerOption {
	return func(options *dialerOptions) {
		options.maxIdleConns = i
	}
}

// MaxIdleConnsPerHost specifies the number of idle (keep-alive) HTTP
// connections that will be maintained per host.
// Existing idle connections will be used instead of creating new HTTP
// connections.
//
// Defaults to 2 connections.
func MaxIdleConnsPerHost(i int) DialerOption {
	return func(options *dialerOptions) {
		options.maxIdleConnsPerHost = i
	}
}

// IdleConnTimeout is the maximum amount of time an idle (keep-alive)
// connection will remain idle before closing itself.
// Zero means no limit.
func IdleConnTimeout(t time.Duration) DialerOption {
	return func(options *dialerOptions) {
		options.idleConnTimeout = t
	}
}

// DisableKeepAlives prevents re-use of TCP connections between different HTTP
// requests.
func DisableKeepAlives() DialerOption {
	return func(options *dialerOptions) {
		options.disableKeepAlives = true
	}
}

// DisableCompression if true prevents the Dialer from requesting
// compression with an "Accept-Encoding: gzip" request header when the Request
// contains no existing Accept-Encoding value. If the Dialer requests gzip
// on its own and gets a gzipped response, it's transparently decoded in the
// Response.Body. However, if the user explicitly requested gzip it is not
// automatically uncompressed.
func DisableCompression() DialerOption {
	return func(options *dialerOptions) {
		options.disableCompression = true
	}
}

// ResponseHeaderTimeout if non-zero specifies the amount of time to wait for
// a server's response headers after fully writing the request (including its
// body, if any).  This time does not include the time to read the response
// body.
func ResponseHeaderTimeout(t time.Duration) DialerOption {
	return func(options *dialerOptions) {
		options.responseHeaderTimeout = t
	}
}

// ConnTimeout is the time that the dialer will wait for a connection attempt.
// If a peer has been retained by a peer list, connection attempts are
// performed in a goroutine off the request path.
//
// The default is half a second.
func ConnTimeout(d time.Duration) DialerOption {
	return func(options *dialerOptions) {
		options.connTimeout = d
	}
}

// ConnBackoff specifies the connection backoff strategy for delays between
// connection attempts for each peer.
//
// The default is exponential backoff starting with 10ms fully jittered,
// doubling each attempt, with a maximum interval of 30s.
func ConnBackoff(s backoffapi.Strategy) DialerOption {
	return func(options *dialerOptions) {
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
func InnocenceWindow(d time.Duration) DialerOption {
	return func(options *dialerOptions) {
		options.innocenceWindow = d
	}
}

// Logger sets a logger to use for internal logging.
//
// The default is to not write any logs.
func Logger(logger *zap.Logger) DialerOption {
	return func(options *dialerOptions) {
		options.logger = logger
	}
}

// Hidden option to override the buildHTTPClient function. This is used only
// for testing.
func buildClient(f func(*dialerOptions) *http.Client) DialerOption {
	return func(options *dialerOptions) {
		options.buildClient = f
	}
}

// NewDialer creates a new HTTP dialer for managing peers and sending requests
func NewDialer(opts ...DialerOption) *Dialer {
	options := newDialerOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return options.newDialer()
}

func (o *dialerOptions) newDialer() *Dialer {
	logger := o.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Dialer{
		once:                lifecycle.NewOnce(),
		client:              o.buildClient(o),
		connTimeout:         o.connTimeout,
		connBackoffStrategy: o.connBackoffStrategy,
		innocenceWindow:     o.innocenceWindow,
		jitter:              o.jitter,
		peers:               make(map[string]*httpPeer),
		tracer:              o.tracer,
		logger:              logger,
	}
}

func buildHTTPClient(options *dialerOptions) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// options lifted from https://golang.org/src/net/http/transport.go
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: options.keepAlive,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          options.maxIdleConns,
			MaxIdleConnsPerHost:   options.maxIdleConnsPerHost,
			IdleConnTimeout:       options.idleConnTimeout,
			DisableKeepAlives:     options.disableKeepAlives,
			DisableCompression:    options.disableCompression,
			ResponseHeaderTimeout: options.responseHeaderTimeout,
		},
	}
}

// Dialer keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Dialer struct {
	lock sync.Mutex
	once *lifecycle.Once

	client *http.Client
	peers  map[string]*httpPeer

	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	connectorsGroup     sync.WaitGroup
	innocenceWindow     time.Duration
	jitter              func(int64) int64

	tracer opentracing.Tracer
	logger *zap.Logger
}

var _ yarpc.Dialer = (*Dialer)(nil)

// Start starts the HTTP dialer.
func (a *Dialer) Start() error {
	return a.once.Start(func() error {
		return nil // Nothing to do
	})
}

// Stop stops the HTTP dialer.
func (a *Dialer) Stop() error {
	return a.once.Stop(func() error {
		a.connectorsGroup.Wait()
		return nil
	})
}

// NewSingleOutbound builds an outbound that sends YARPC requests over HTTP
// to the specified URL.
//
// The URLTemplate option has no effect in this form.
func (a *Dialer) NewSingleOutbound(uri string, opts ...OutboundOption) *Outbound {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	chooser := yarpcpeer.NewSingle(yarpc.Address(parsedURL.Host), a)
	o := NewOutbound(chooser)
	for _, opt := range opts {
		opt(o)
	}
	o.setURLTemplate(uri)
	return o
}

// IsRunning returns whether the HTTP dialer is running.
func (a *Dialer) IsRunning() bool {
	return a.once.IsRunning()
}

// RetainPeer gets or creates a Peer for the specified yarpc.Subscriber (usually a yarpc.Chooser)
func (a *Dialer) RetainPeer(pid yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	p := a.getOrCreatePeer(pid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Dialer) getOrCreatePeer(pid yarpc.Identifier) *httpPeer {
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

// ReleasePeer releases a peer from the yarpc.Subscriber and removes that peer from the Dialer if nothing is listening to it
func (a *Dialer) ReleasePeer(pid yarpc.Identifier, sub yarpc.Subscriber) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	p, ok := a.peers[pid.Identifier()]
	if !ok {
		return yarpc.ErrTransportHasNoReferenceToPeer{
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
