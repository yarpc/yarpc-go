// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/single"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/atomic"
)

var errOutboundNotStarted = errors.ErrOutboundNotStarted("http.Outbound")

// this ensures the HTTP outbound implements both transport.Outbound interfaces
var (
	_ transport.UnaryOutbound  = (*Outbound)(nil)
	_ transport.OnewayOutbound = (*Outbound)(nil)
)

type outboundConfig struct {
	keepAlive time.Duration
}

// NewOutbound builds a new HTTP outbound that sends requests to the given
// URL.
//
// Deprecated: create outbounds through NewPeerListOutbound instead
func NewOutbound(urlStr string, opts ...TransportOption) *Outbound {
	transport := NewTransport(opts...)

	urlTemplate, hp := parseURL(urlStr)

	peerID := hostport.PeerIdentifier(hp)
	c := single.New(peerID, transport)

	err := c.Start()
	if err != nil {
		// This should never happen, single shouldn't return an error here
		panic(fmt.Sprintf("could not start single peerChooser, err: %s", err))
	}

	return NewChooserOutbound(c, urlTemplate)
}

func parseURL(urlStr string) (*url.URL, string) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		panic(fmt.Sprintf("invalid url: %s, err: %s", urlStr, err))
	}

	return parsedURL, parsedURL.Host
}

// NewChooserOutbound builds a new HTTP outbound built around a PeerList
// for getting potential downstream hosts.
// Chooser.Choose MUST return *hostport.Peer objects.
// Chooser.Start MUST be called before Outbound.Start
func NewChooserOutbound(chooser peer.Chooser, urlTemplate *url.URL) *Outbound {
	return &Outbound{
		started:     atomic.NewBool(false),
		chooser:     chooser,
		urlTemplate: urlTemplate,
	}
}

// Outbound is an HTTP UnaryOutbound and OnewayOutbound
type Outbound struct {
	started     *atomic.Bool
	deps        transport.Deps
	chooser     peer.Chooser
	urlTemplate *url.URL
}

// Start the HTTP outbound
func (o *Outbound) Start(d transport.Deps) error {
	if !o.started.Swap(true) {
		o.deps = d
	}
	return nil
}

// Stop the HTTP outbound
func (o *Outbound) Stop() error {
	o.started.Swap(false)
	return nil
}

// Call makes a HTTP request
func (o *Outbound) Call(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
	if !o.started.Load() {
		// panic because there's no recovery from this
		panic(errOutboundNotStarted)
	}
	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	return o.call(ctx, treq, start, ttl)
}

type ack struct {
	time time.Time
}

func (a ack) String() string {
	return a.time.String()
}

// CallOneway makes a oneway request
func (o *Outbound) CallOneway(ctx context.Context, treq *transport.Request) (transport.Ack, error) {
	if !o.started.Load() {
		// panic because there's no recovery from this
		panic(errOutboundNotStarted)
	}
	start := time.Now()
	var ttl time.Duration

	_, err := o.call(ctx, treq, start, ttl)
	if err != nil {
		return nil, err
	}

	return ack{time: time.Now()}, nil
}

func (o *Outbound) call(ctx context.Context, treq *transport.Request, start time.Time, ttl time.Duration) (*transport.Response, error) {
	p, err := o.getPeerForRequest(ctx, treq)
	if err != nil {
		return nil, err
	}
	p.StartRequest(nil)
	defer p.EndRequest(nil)

	req, err := o.createRequest(p, treq)
	if err != nil {
		return nil, err
	}

	req.Header = applicationHeaders.ToHTTPHeaders(treq.Headers, nil)
	ctx, req, span := o.withOpentracingSpan(ctx, req, treq, start)
	defer span.Finish()
	req = o.withCoreHeaders(req, treq, ttl)

	client, err := o.getHTTPClient(p)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req.WithContext(ctx))

	if err != nil {
		// Workaround borrowed from ctxhttp until
		// https://github.com/golang/go/issues/17711 is resolved.
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}

		span.SetTag("error", true)
		span.LogEvent(err.Error())
		if err == context.DeadlineExceeded {
			end := time.Now()
			return nil, errors.ClientTimeoutError(treq.Service, treq.Procedure, end.Sub(start))
		}

		return nil, err
	}

	span.SetTag("http.status_code", response.StatusCode)

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		appHeaders := applicationHeaders.FromHTTPHeaders(
			response.Header, transport.NewHeaders())
		return &transport.Response{
			Headers: appHeaders,
			Body:    response.Body,
		}, nil
	}

	return nil, getErrFromResponse(response)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, treq *transport.Request) (*hostport.Peer, error) {
	p, err := o.chooser.Choose(ctx, treq)
	if err != nil {
		return nil, err
	}

	hpPeer, ok := p.(*hostport.Peer)
	if !ok {
		return nil, peer.ErrInvalidPeerConversion{
			Peer:         p,
			ExpectedType: "*hostport.Peer",
		}
	}

	return hpPeer, nil
}

func (o *Outbound) createRequest(p *hostport.Peer, treq *transport.Request) (*http.Request, error) {
	newURL := *o.urlTemplate
	newURL.Host = p.HostPort()
	return http.NewRequest("POST", newURL.String(), treq.Body)
}

func (o *Outbound) withOpentracingSpan(ctx context.Context, req *http.Request, treq *transport.Request, start time.Time) (context.Context, *http.Request, opentracing.Span) {
	// Apply HTTP Context headers for tracing and baggage carried by tracing.
	tracer := o.deps.Tracer()
	var parent opentracing.SpanContext // ok to be nil
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parent = parentSpan.Context()
	}
	span := tracer.StartSpan(
		treq.Procedure,
		opentracing.StartTime(start),
		opentracing.ChildOf(parent),
		opentracing.Tags{
			"rpc.caller":    treq.Caller,
			"rpc.service":   treq.Service,
			"rpc.encoding":  treq.Encoding,
			"rpc.transport": "http",
		},
	)
	ext.PeerService.Set(span, treq.Service)
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, req.URL.String())
	ctx = opentracing.ContextWithSpan(ctx, span)

	tracer.Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	return ctx, req, span
}

func (o *Outbound) withCoreHeaders(req *http.Request, treq *transport.Request, ttl time.Duration) *http.Request {
	req.Header.Set(CallerHeader, treq.Caller)
	req.Header.Set(ServiceHeader, treq.Service)
	req.Header.Set(ProcedureHeader, treq.Procedure)
	if ttl != 0 {
		req.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))
	}
	if treq.ShardKey != "" {
		req.Header.Set(ShardKeyHeader, treq.ShardKey)
	}
	if treq.RoutingKey != "" {
		req.Header.Set(RoutingKeyHeader, treq.RoutingKey)
	}
	if treq.RoutingDelegate != "" {
		req.Header.Set(RoutingDelegateHeader, treq.RoutingDelegate)
	}

	encoding := string(treq.Encoding)
	if encoding != "" {
		req.Header.Set(EncodingHeader, encoding)
	}

	return req
}

func (o *Outbound) getHTTPClient(p *hostport.Peer) (*http.Client, error) {
	transport, ok := p.Transport().(*Transport)
	if !ok {
		return nil, peer.ErrInvalidTransportConversion{
			Transport:    p.Transport(),
			ExpectedType: "*http.Transport",
		}
	}
	return transport.client, nil
}

func getErrFromResponse(response *http.Response) error {
	// TODO Behavior for 300-range status codes is undefined
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if err := response.Body.Close(); err != nil {
		return err
	}

	// Trim the trailing newline from HTTP error messages
	message := strings.TrimSuffix(string(contents), "\n")

	if response.StatusCode >= 400 && response.StatusCode < 500 {
		return errors.RemoteBadRequestError(message)
	}

	if response.StatusCode == http.StatusGatewayTimeout {
		return errors.RemoteTimeoutError(message)
	}

	return errors.RemoteUnexpectedError(message)
}
