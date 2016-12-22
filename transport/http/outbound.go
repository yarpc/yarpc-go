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
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/sync"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var errOutboundNotStarted = errors.ErrOutboundNotStarted("http.Outbound")

// this ensures the HTTP outbound implements both transport.Outbound interfaces
var (
	_ transport.UnaryOutbound  = (*Outbound)(nil)
	_ transport.OnewayOutbound = (*Outbound)(nil)
)

var defaultURLTemplate, _ = url.Parse("http://localhost")

// OutboundOption is suitable as an argument to NewOutbound
type OutboundOption func(*Outbound)

// URLTemplate specifies a template for URLs to this outbound.
// The peer (host:port) may vary from call to call.
// The URL template specifies the protocol and path.
func URLTemplate(template string) OutboundOption {
	return func(o *Outbound) {
		o.setURLTemplate(template)
	}
}

// NewOutbound builds a new HTTP outbound built around a peer.Chooser
// for getting potential downstream hosts.
// Chooser.Choose MUST return *hostport.Peer objects.
// Chooser.Start MUST be called before Outbound.Start
func (t *Transport) NewOutbound(chooser peer.Chooser) *Outbound {
	o := &Outbound{
		chooser:     chooser,
		urlTemplate: defaultURLTemplate,
		tracer:      t.tracer,
	}
	return o
}

// NewOutbound builds a new HTTP outbound built around a peer.Chooser
// for getting potential downstream hosts.
// Chooser.Choose MUST return *hostport.Peer objects.
// Chooser.Start MUST be called before Outbound.Start
func NewOutbound(chooser peer.Chooser, opts ...OutboundOption) *Outbound {
	o := &Outbound{
		chooser:     chooser,
		urlTemplate: defaultURLTemplate,
		tracer:      opentracing.GlobalTracer(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// NewSingleOutbound creates an outbound from a single URL (a bare host:port is
// not sufficient).
// This form defers to the underlying HTTP agent's peer selection and load
// balancing, using DNS.
func (t *Transport) NewSingleOutbound(URL string, opts ...OutboundOption) *Outbound {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		panic(err.Error())
	}
	o := t.NewOutbound(peerchooser.NewSingle(hostport.PeerIdentifier(parsedURL.Host), t))
	o.setURLTemplate(URL)
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Outbound is an HTTP UnaryOutbound and OnewayOutbound
type Outbound struct {
	chooser     peer.Chooser
	urlTemplate *url.URL
	tracer      opentracing.Tracer

	once sync.LifecycleOnce
}

// setURLTemplate configures an alternate URL template.
// The host:port portion of the URL template gets replaced by the chosen peer's
// identifier for each outbound request.
func (o *Outbound) setURLTemplate(URL string) {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Fatalf("failed to configure HTTP outbound: invalid URL template %q: %s", URL, err)
	}
	o.urlTemplate = parsedURL
}

// Transports returns the outbound's HTTP transport.
func (o *Outbound) Transports() []transport.Transport {
	// TODO factor out transport and return it here.
	return []transport.Transport{}
}

// Start the HTTP outbound
func (o *Outbound) Start() error {
	return o.once.Start(o.chooser.Start)
}

// Stop the HTTP outbound
func (o *Outbound) Stop() error {
	return o.once.Stop(o.chooser.Stop)
}

// IsRunning returns whether the Outbound is running.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Call makes a HTTP request
func (o *Outbound) Call(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
	if !o.IsRunning() {
		// TODO replace with "panicInDebug"
		return nil, errOutboundNotStarted
	}
	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	return o.call(ctx, treq, start, ttl)
}

// CallOneway makes a oneway request
func (o *Outbound) CallOneway(ctx context.Context, treq *transport.Request) (transport.Ack, error) {
	if !o.IsRunning() {
		// TODO replace with "panicInDebug"
		return nil, errOutboundNotStarted
	}
	start := time.Now()
	var ttl time.Duration

	_, err := o.call(ctx, treq, start, ttl)
	if err != nil {
		return nil, err
	}

	return time.Now(), nil
}

func (o *Outbound) call(ctx context.Context, treq *transport.Request, start time.Time, ttl time.Duration) (*transport.Response, error) {
	p, onFinish, err := o.getPeerForRequest(ctx, treq)
	if err != nil {
		return nil, err
	}

	resp, err := o.callWithPeer(ctx, treq, start, ttl, p)

	// Call the onFinish method right before returning (with the error from call with peer)
	onFinish(err)
	return resp, err
}

func (o *Outbound) callWithPeer(
	ctx context.Context,
	treq *transport.Request,
	start time.Time,
	ttl time.Duration,
	p *hostport.Peer,
) (*transport.Response, error) {
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
		appError := response.Header.Get(ApplicationStatusHeader) == ApplicationErrorStatus
		return &transport.Response{
			Headers:          appHeaders,
			Body:             response.Body,
			ApplicationError: appError,
		}, nil
	}

	return nil, getErrFromResponse(response)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, treq *transport.Request) (*hostport.Peer, func(error), error) {
	p, onFinish, err := o.chooser.Choose(ctx, treq)
	if err != nil {
		return nil, nil, err
	}

	hpPeer, ok := p.(*hostport.Peer)
	if !ok {
		return nil, nil, peer.ErrInvalidPeerConversion{
			Peer:         p,
			ExpectedType: "*hostport.Peer",
		}
	}

	return hpPeer, onFinish, nil
}

func (o *Outbound) createRequest(p *hostport.Peer, treq *transport.Request) (*http.Request, error) {
	newURL := *o.urlTemplate
	newURL.Host = p.HostPort()
	return http.NewRequest("POST", newURL.String(), treq.Body)
}

func (o *Outbound) withOpentracingSpan(ctx context.Context, req *http.Request, treq *transport.Request, start time.Time) (context.Context, *http.Request, opentracing.Span) {
	// Apply HTTP Context headers for tracing and baggage carried by tracing.
	tracer := o.tracer
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
	t, ok := p.Transport().(*Transport)
	if !ok {
		return nil, peer.ErrInvalidTransportConversion{
			Transport:    p.Transport(),
			ExpectedType: "*http.Transport",
		}
	}
	return t.client, nil
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
