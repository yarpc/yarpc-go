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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

// this ensures the HTTP outbound implements both transport.Outbound interfaces
var (
	_ transport.UnaryOutbound              = (*Outbound)(nil)
	_ transport.OnewayOutbound             = (*Outbound)(nil)
	_ introspection.IntrospectableOutbound = (*Outbound)(nil)
)

var defaultURLTemplate, _ = url.Parse("http://localhost")

// OutboundOption customizes an HTTP Outbound.
type OutboundOption func(*Outbound)

func (OutboundOption) httpOption() {}

// URLTemplate specifies the URL this outbound makes requests to. For
// peer.Chooser-based outbounds, the peer (host:port) spection of the URL may
// vary from call to call but the rest will remain unchanged. For single-peer
// outbounds, the URL will be used as-is.
func URLTemplate(template string) OutboundOption {
	return func(o *Outbound) {
		o.setURLTemplate(template)
	}
}

// AddHeader specifies that an HTTP outbound should always include the given
// header in outgoung requests.
//
// 	httpTransport.NewOutbound(chooser, http.AddHeader("X-Token", "TOKEN"))
//
// Note that headers starting with "Rpc-" are reserved by YARPC. This function
// will panic if the header starts with "Rpc-".
func AddHeader(key, value string) OutboundOption {
	if strings.HasPrefix(strings.ToLower(key), "rpc-") {
		panic(fmt.Errorf(
			"invalid header name %q: "+
				`headers starting with "Rpc-" are reserved by YARPC`, key))
	}

	return func(o *Outbound) {
		if o.headers == nil {
			o.headers = make(http.Header)
		}
		o.headers.Add(key, value)
	}
}

// NewOutbound builds an HTTP outbound that sends requests to peers supplied
// by the given peer.Chooser. The URL template for used for the different
// peers may be customized using the URLTemplate option.
//
// The peer chooser and outbound must share the same transport, in this case
// the HTTP transport.
// The peer chooser must use the transport's RetainPeer to obtain peer
// instances and return those peers to the outbound when it calls Choose.
// The concrete peer type is private and intrinsic to the HTTP transport.
func (t *Transport) NewOutbound(chooser peer.Chooser, opts ...OutboundOption) *Outbound {
	o := &Outbound{
		once:              lifecycle.NewOnce(),
		chooser:           chooser,
		urlTemplate:       defaultURLTemplate,
		tracer:            t.tracer,
		transport:         t,
		bothResponseError: true,
	}
	for _, opt := range opts {
		opt(o)
	}
	o.sender = &transportSender{Client: t.client}
	return o
}

// NewOutbound builds an HTTP outbound that sends requests to peers supplied
// by the given peer.Chooser. The URL template for used for the different
// peers may be customized using the URLTemplate option.
//
// The peer chooser and outbound must share the same transport, in this case
// the HTTP transport.
// The peer chooser must use the transport's RetainPeer to obtain peer
// instances and return those peers to the outbound when it calls Choose.
// The concrete peer type is private and intrinsic to the HTTP transport.
func NewOutbound(chooser peer.Chooser, opts ...OutboundOption) *Outbound {
	return NewTransport().NewOutbound(chooser, opts...)
}

// NewSingleOutbound builds an outbound that sends YARPC requests over HTTP
// to the specified URL.
//
// The URLTemplate option has no effect in this form.
func (t *Transport) NewSingleOutbound(uri string, opts ...OutboundOption) *Outbound {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	chooser := peerchooser.NewSingle(hostport.PeerIdentifier(parsedURL.Host), t)
	o := t.NewOutbound(chooser)
	for _, opt := range opts {
		opt(o)
	}
	o.setURLTemplate(uri)
	return o
}

// Outbound sends YARPC requests over HTTP. It may be constructed using the
// NewOutbound function or the NewOutbound or NewSingleOutbound methods on the
// HTTP Transport. It is recommended that services use a single HTTP transport
// to construct all HTTP outbounds, ensuring efficient sharing of resources
// across the different outbounds.
type Outbound struct {
	chooser     peer.Chooser
	urlTemplate *url.URL
	tracer      opentracing.Tracer
	transport   *Transport
	sender      sender

	// Headers to add to all outgoing requests.
	headers http.Header

	once *lifecycle.Once

	// should only be false in testing
	bothResponseError bool
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
	return []transport.Transport{o.transport}
}

// Chooser returns the outbound's peer chooser.
func (o *Outbound) Chooser() peer.Chooser {
	return o.chooser
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
	if treq == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("request for http unary outbound was nil")
	}

	return o.call(ctx, treq)
}

// CallOneway makes a oneway request
func (o *Outbound) CallOneway(ctx context.Context, treq *transport.Request) (transport.Ack, error) {
	if treq == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("request for http oneway outbound was nil")
	}

	_, err := o.call(ctx, treq)
	if err != nil {
		return nil, err
	}

	return time.Now(), nil
}

func (o *Outbound) call(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
	start := time.Now()
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing context deadline")
	}
	ttl := deadline.Sub(start)

	hreq, err := o.createRequest(treq)
	if err != nil {
		return nil, err
	}
	hreq.Header = applicationHeaders.ToHTTPHeaders(treq.Headers, nil)
	ctx, hreq, span, err := o.withOpentracingSpan(ctx, hreq, treq, start)
	if err != nil {
		return nil, err
	}
	defer span.Finish()

	hreq = o.withCoreHeaders(hreq, treq, ttl)
	hreq = hreq.WithContext(ctx)

	response, err := o.roundTrip(hreq, treq, start, o.transport.client)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(opentracinglog.String("event", err.Error()))
		return nil, err
	}

	span.SetTag("http.status_code", response.StatusCode)

	// Service name match validation, return yarpcerrors.CodeInternal error if not match
	if match, resSvcName := checkServiceMatch(treq.Service, response.Header); !match {
		return nil, transport.UpdateSpanWithErr(span,
			yarpcerrors.InternalErrorf("service name sent from the request "+
				"does not match the service name received in the response, sent %q, got: %q", treq.Service, resSvcName))
	}

	tres := &transport.Response{
		Headers:          applicationHeaders.FromHTTPHeaders(response.Header, transport.NewHeaders()),
		Body:             response.Body,
		ApplicationError: response.Header.Get(ApplicationStatusHeader) == ApplicationErrorStatus,
	}

	bothResponseError := response.Header.Get(BothResponseErrorHeader) == AcceptTrue
	if bothResponseError && o.bothResponseError {
		if response.StatusCode >= 300 {
			return tres, getYARPCErrorFromResponse(response, true)
		}
		return tres, nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return tres, nil
	}
	return nil, getYARPCErrorFromResponse(response, false)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, treq *transport.Request) (*httpPeer, func(error), error) {
	p, onFinish, err := o.chooser.Choose(ctx, treq)
	if err != nil {
		return nil, nil, err
	}

	hpPeer, ok := p.(*httpPeer)
	if !ok {
		return nil, nil, peer.ErrInvalidPeerConversion{
			Peer:         p,
			ExpectedType: "*httpPeer",
		}
	}

	return hpPeer, onFinish, nil
}

func (o *Outbound) createRequest(treq *transport.Request) (*http.Request, error) {
	newURL := *o.urlTemplate
	return http.NewRequest("POST", newURL.String(), treq.Body)
}

func (o *Outbound) withOpentracingSpan(ctx context.Context, req *http.Request, treq *transport.Request, start time.Time) (context.Context, *http.Request, opentracing.Span, error) {
	// Apply HTTP Context headers for tracing and baggage carried by tracing.
	tracer := o.tracer
	var parent opentracing.SpanContext // ok to be nil
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parent = parentSpan.Context()
	}
	tags := opentracing.Tags{
		"rpc.caller":    treq.Caller,
		"rpc.service":   treq.Service,
		"rpc.encoding":  treq.Encoding,
		"rpc.transport": "http",
	}
	for k, v := range yarpc.OpentracingTags {
		tags[k] = v
	}
	span := tracer.StartSpan(
		treq.Procedure,
		opentracing.StartTime(start),
		opentracing.ChildOf(parent),
		tags,
	)
	ext.PeerService.Set(span, treq.Service)
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, req.URL.String())
	ctx = opentracing.ContextWithSpan(ctx, span)

	err := tracer.Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	return ctx, req, span, err
}

func (o *Outbound) withCoreHeaders(req *http.Request, treq *transport.Request, ttl time.Duration) *http.Request {
	// Add default headers to all requests.
	for k, vs := range o.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

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

	if o.bothResponseError {
		req.Header.Set(AcceptsBothResponseErrorHeader, AcceptTrue)
	}

	return req
}

func getYARPCErrorFromResponse(response *http.Response, bothResponseError bool) error {
	var contents string
	if bothResponseError {
		contents = response.Header.Get(ErrorMessageHeader)
	} else {
		contentsBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return yarpcerrors.Newf(yarpcerrors.CodeInternal, err.Error())
		}
		contents = string(contentsBytes)
		if err := response.Body.Close(); err != nil {
			return yarpcerrors.Newf(yarpcerrors.CodeInternal, err.Error())
		}
	}
	// use the status code if we can't get a code from the headers
	code := statusCodeToBestCode(response.StatusCode)
	if errorCodeText := response.Header.Get(ErrorCodeHeader); errorCodeText != "" {
		var errorCode yarpcerrors.Code
		// TODO: what to do with error?
		if err := errorCode.UnmarshalText([]byte(errorCodeText)); err == nil {
			code = errorCode
		}
	}
	return intyarpcerrors.NewWithNamef(
		code,
		response.Header.Get(ErrorNameHeader),
		strings.TrimSuffix(contents, "\n"),
	)
}

// Only does verification if there is a response header
func checkServiceMatch(reqSvcName string, resHeaders http.Header) (bool, string) {
	serviceName := resHeaders.Get(ServiceHeader)
	return serviceName == "" || serviceName == reqSvcName, serviceName
}

// RoundTrip implements the http.RoundTripper interface, making a YARPC HTTP outbound suitable as a
// Transport when constructing an HTTP Client. An HTTP client is suitable only for relative paths to
// a single outbound service. The HTTP outbound overrides the host:port portion of the URL of the
// provided request.
//
// Sample usage:
//
//  client := http.Client{Transport: outbound}
//
// Thereafter use the Golang standard library HTTP to send requests with this client.
//
//  ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//  defer cancel()
//  req, err := http.NewRequest("GET", "http://example.com/", nil /* body */)
//  req = req.WithContext(ctx)
//  res, err := client.Do(req)
//
// All requests must have a deadline on the context.
// The peer chooser for raw HTTP requests will receive a YARPC transport.Request with no body.
//
// OpenTracing information must be added manually, before this call, to support context propagation.
func (o *Outbound) RoundTrip(hreq *http.Request) (*http.Response, error) {
	return o.roundTrip(hreq, nil /* treq */, time.Now(), o.sender)
}

func (o *Outbound) roundTrip(hreq *http.Request, treq *transport.Request, start time.Time, sender sender) (*http.Response, error) {
	ctx := hreq.Context()

	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, yarpcerrors.Newf(
			yarpcerrors.CodeInvalidArgument,
			"missing context deadline")
	}
	ttl := deadline.Sub(start)

	// When sending requests through the RoundTrip method, we construct the
	// transport request from the HTTP headers as if it were an inbound
	// request.
	// The API for setting transport metadata for an outbound request when
	// using the go stdlib HTTP client is to use headers as the YAPRC HTTP
	// transport header conventions.
	if treq == nil {
		treq = &transport.Request{
			Caller:          hreq.Header.Get(CallerHeader),
			Service:         hreq.Header.Get(ServiceHeader),
			Encoding:        transport.Encoding(hreq.Header.Get(EncodingHeader)),
			Procedure:       hreq.Header.Get(ProcedureHeader),
			ShardKey:        hreq.Header.Get(ShardKeyHeader),
			RoutingKey:      hreq.Header.Get(RoutingKeyHeader),
			RoutingDelegate: hreq.Header.Get(RoutingDelegateHeader),
			Headers:         applicationHeaders.FromHTTPHeaders(hreq.Header, transport.Headers{}),
		}
	}

	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, intyarpcerrors.AnnotateWithInfo(
			yarpcerrors.FromError(err),
			"error waiting for HTTP outbound to start for service: %s",
			treq.Service)
	}

	p, onFinish, err := o.getPeerForRequest(ctx, treq)
	if err != nil {
		return nil, err
	}

	hres, err := o.doWithPeer(ctx, hreq, treq, start, ttl, p, sender)
	// Call the onFinish method before returning (with the error from call with peer)
	onFinish(err)
	return hres, err
}

func (o *Outbound) doWithPeer(
	ctx context.Context,
	hreq *http.Request,
	treq *transport.Request,
	start time.Time,
	ttl time.Duration,
	p *httpPeer,
	sender sender,
) (*http.Response, error) {
	hreq.URL.Host = p.HostPort()

	response, err := sender.Do(hreq.WithContext(ctx))
	if err != nil {
		// Workaround borrowed from ctxhttp until
		// https://github.com/golang/go/issues/17711 is resolved.
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
		if err == context.DeadlineExceeded {
			// Note that the connection experienced a time out, which may
			// indicate that the connection is half-open, that the destination
			// died without sending a TCP FIN packet.
			p.OnSuspect()

			end := time.Now()
			return nil, yarpcerrors.Newf(
				yarpcerrors.CodeDeadlineExceeded,
				"client timeout for procedure %q of service %q after %v",
				treq.Procedure, treq.Service, end.Sub(start))
		}

		// Note that the connection may have been lost so the peer connection
		// maintenance loop resumes probing for availability.
		p.onDisconnected()

		return nil, yarpcerrors.Newf(yarpcerrors.CodeUnknown, "unknown error from http client: %s", err.Error())
	}

	return response, nil
}

// Introspect returns basic status about this outbound.
func (o *Outbound) Introspect() introspection.OutboundStatus {
	state := "Stopped"
	if o.IsRunning() {
		state = "Running"
	}
	var chooser introspection.ChooserStatus
	if i, ok := o.chooser.(introspection.IntrospectableChooser); ok {
		chooser = i.Introspect()
	} else {
		chooser = introspection.ChooserStatus{
			Name: "Introspection not available",
		}
	}
	return introspection.OutboundStatus{
		Transport: "http",
		Endpoint:  o.urlTemplate.String(),
		State:     state,
		Chooser:   chooser,
	}
}
