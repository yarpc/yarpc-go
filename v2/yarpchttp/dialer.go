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
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/zap"
)

func defaultBuildClient(dialer *Dialer) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// options lifted from https://golang.org/src/net/http/transport.go
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: dialer.KeepAlive,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          dialer.MaxIdleConns,
			MaxIdleConnsPerHost:   dialer.MaxIdleConnsPerHost,
			IdleConnTimeout:       dialer.IdleConnTimeout,
			DisableKeepAlives:     dialer.DisableKeepAlives,
			DisableCompression:    dialer.DisableCompression,
			ResponseHeaderTimeout: dialer.ResponseHeaderTimeout,
		},
	}
}

// Dialer keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Dialer struct {
	// KeepAlive specifies the keep-alive period for the network connection. If
	// zero, keep-alives are disabled.
	//
	// Defaults to 30 seconds.
	KeepAlive time.Duration

	// MaxIdleConns controls the maximum number of idle (keep-alive) connections
	// across all hosts. Zero means no limit.
	MaxIdleConns int

	// MaxIdleConnsPerHost specifies the number of idle (keep-alive) HTTP
	// connections that will be maintained per host.
	// Existing idle connections will be used instead of creating new HTTP
	// connections.
	//
	// Defaults to 2 connections.
	MaxIdleConnsPerHost int

	// IdleConnTimeout is the maximum amount of time an idle (keep-alive)
	// connection will remain idle before closing itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration

	// DisableKeepAlives prevents re-use of TCP connections between different HTTP
	// requests.
	DisableKeepAlives bool

	// DisableCompression if true prevents the Dialer from requesting
	// compression with an "Accept-Encoding: gzip" request header when the Request
	// contains no existing Accept-Encoding value. If the Dialer requests gzip
	// on its own and gets a gzipped response, it's transparently decoded in the
	// Response.Body. However, if the user explicitly requested gzip it is not
	// automatically uncompressed.
	DisableCompression bool

	// ResponseHeaderTimeout if non-zero specifies the amount of time to wait for
	// a server's response headers after fully writing the request (including its
	// body, if any).  This time does not include the time to read the response
	// body.
	ResponseHeaderTimeout time.Duration

	// ConnTimeout is the time that the dialer will wait for a connection attempt.
	// If a peer has been retained by a peer list, connection attempts are
	// performed in a goroutine off the request path.
	//
	// The default is half a second.
	ConnTimeout time.Duration

	// ConnBackoff specifies the connection backoff strategy for delays between
	// connection attempts for each peer.
	//
	// The default is exponential backoff starting with 10ms fully jittered,
	// doubling each attempt, with a maximum interval of 30s.
	ConnBackoff backoffapi.Strategy

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
	InnocenceWindow time.Duration

	Tracer opentracing.Tracer

	// Logger sets a logger to use for internal logging.
	//
	// The default is to not write any logs.
	Logger *zap.Logger

	// buildClient is a hidden option to override the buildClient function.
	// This is used only for testing.
	buildClient func(*Dialer) *http.Client

	// jitter is a hidden option to override the jitter function for backoff.
	jitter func(int64) int64

	// internal is the internal state of a running dialer.
	internal *dialerInternals
}

var _ yarpc.Dialer = (*Dialer)(nil)

type dialerInternals struct {
	lock            sync.Mutex
	client          *http.Client
	peers           map[string]*httpPeer
	connectorsGroup sync.WaitGroup

	keepAlive           time.Duration
	maxIdleConnsPerHost int
	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	buildClient         func(*Dialer) *http.Client
	innocenceWindow     time.Duration
	jitter              func(int64) int64

	tracer opentracing.Tracer
	logger *zap.Logger
}

// Start starts the HTTP dialer.
func (d *Dialer) Start(_ context.Context) error {
	d.internal = &dialerInternals{
		keepAlive:           30 * time.Second,
		maxIdleConnsPerHost: 2,
		connTimeout:         defaultConnTimeout,
		connBackoffStrategy: backoff.DefaultExponential,
		innocenceWindow:     defaultInnocenceWindow,
		jitter:              rand.Int63n,
		peers:               make(map[string]*httpPeer),
	}

	if d.ConnTimeout != 0 {
		d.internal.connTimeout = d.ConnTimeout
	}
	if d.InnocenceWindow != 0 {
		d.internal.innocenceWindow = d.InnocenceWindow
	}
	if d.Logger != nil {
		d.internal.logger = d.Logger
	}
	if d.Tracer != nil {
		d.internal.tracer = d.Tracer
	}
	if d.jitter != nil {
		d.internal.jitter = d.jitter
	}

	buildClient := defaultBuildClient
	if d.buildClient != nil {
		buildClient = d.buildClient
	}
	d.internal.client = buildClient(d)

	return nil
}

// Stop stops the HTTP dialer.
func (d *Dialer) Stop(ctx context.Context) error {
	return d.internal.stop(ctx)
}

func (d *dialerInternals) stop(ctx context.Context) error {
	d.lock.Lock()
	for id, peer := range d.peers {
		peer.Release()
		delete(d.peers, id)
	}
	d.lock.Unlock()

	done := make(chan struct{}, 0)
	go func() {
		d.connectorsGroup.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// NewSingleOutbound builds an outbound that sends YARPC requests over HTTP
// to the specified URL.
//
// The URLTemplate option has no effect in this form.
func (d *Dialer) NewSingleOutbound(uri string, opts ...OutboundOption) *Outbound {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	chooser := yarpcpeer.NewSingle(yarpc.Address(parsedURL.Host), d)
	o := NewOutbound(chooser)
	for _, opt := range opts {
		opt(o)
	}
	o.setURLTemplate(uri)
	return o
}

// RetainPeer gets or creates a Peer for the specified yarpc.Subscriber (usually a yarpc.Chooser)
func (d *Dialer) RetainPeer(pid yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	if d.internal == nil {
		return nil, fmt.Errorf("yarpchttp.Dialer.RetainPeer must be called after Start")
	}
	return d.internal.retainPeer(pid, sub)
}

func (d *dialerInternals) retainPeer(pid yarpc.Identifier, sub yarpc.Subscriber) (yarpc.Peer, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	p := d.getOrCreatePeer(pid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (d *dialerInternals) getOrCreatePeer(pid yarpc.Identifier) *httpPeer {
	addr := pid.Identifier()
	if p, ok := d.peers[addr]; ok {
		return p
	}
	p := newPeer(addr, d)
	d.peers[addr] = p
	d.connectorsGroup.Add(1)
	go p.MaintainConn()

	return p
}

// ReleasePeer releases a peer from the yarpc.Subscriber and removes that peer from the Dialer if nothing is listening to it
func (d *Dialer) ReleasePeer(pid yarpc.Identifier, sub yarpc.Subscriber) error {
	if d.internal == nil {
		return fmt.Errorf("yarpchttp.Dialer.ReleasePeer must be called after Start")
	}
	return d.internal.releasePeer(pid, sub)
}

func (d *dialerInternals) releasePeer(pid yarpc.Identifier, sub yarpc.Subscriber) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	p, ok := d.peers[pid.Identifier()]
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
		delete(d.peers, pid.Identifier())
		p.Release()
	}

	return nil
}
