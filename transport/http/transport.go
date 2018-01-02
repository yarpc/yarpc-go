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

package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

type transportOptions struct {
	keepAlive           time.Duration
	maxIdleConnsPerHost int
	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	tracer              opentracing.Tracer
	buildClient         func(*transportOptions) *http.Client
	logger              *zap.Logger
}

var defaultTransportOptions = transportOptions{
	keepAlive:           30 * time.Second,
	maxIdleConnsPerHost: 2,
	connTimeout:         defaultConnTimeout,
	connBackoffStrategy: backoff.DefaultExponential,
	buildClient:         buildHTTPClient,
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

// Tracer configures a tracer for the transport and all its inbounds and
// outbounds.
func Tracer(tracer opentracing.Tracer) TransportOption {
	return func(options *transportOptions) {
		options.tracer = tracer
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
	return &Transport{
		once:                lifecycle.NewOnce(),
		client:              o.buildClient(o),
		connTimeout:         o.connTimeout,
		connBackoffStrategy: o.connBackoffStrategy,
		peers:               make(map[string]*httpPeer),
		tracer:              o.tracer,
		logger:              logger,
	}
}

func buildHTTPClient(options *transportOptions) *http.Client {
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
			MaxIdleConnsPerHost:   options.maxIdleConnsPerHost,
		},
	}
}

// Transport keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Transport struct {
	lock sync.Mutex
	once *lifecycle.Once

	client *http.Client
	peers  map[string]*httpPeer

	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	connectorsGroup     sync.WaitGroup

	tracer opentracing.Tracer
	logger *zap.Logger
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

	hppid, ok := pid.(hostport.PeerIdentifier)
	if !ok {
		return nil, peer.ErrInvalidPeerType{
			ExpectedType:   "hostport.PeerIdentifier",
			PeerIdentifier: pid,
		}
	}

	p := a.getOrCreatePeer(hppid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (a *Transport) getOrCreatePeer(pid hostport.PeerIdentifier) *httpPeer {
	if p, ok := a.peers[pid.Identifier()]; ok {
		return p
	}

	p := newPeer(pid, a)
	a.peers[p.Identifier()] = p
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
