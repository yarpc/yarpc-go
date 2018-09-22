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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalyarpcerror"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctracing"
)

// This type assertion ensures that the HTTP outbound implements both yarpc.Outbound interfaces.
var _ yarpc.UnaryOutbound = (*Outbound)(nil)

// This type assertion ensures that the HTTP outbound implements the HTTP
// RoundTripper transport interface.
var _ http.RoundTripper = (*Outbound)(nil)

var defaultURL = &url.URL{Scheme: "http", Host: "localhost"}

// Outbound sends YARPC requests over HTTP.
// It may be constructed using the NewOutbound function or NewSingleOutbound
// methods on the HTTP dialer.
// It is recommended that services use a single HTTP dialer to construct all
// HTTP outbounds, ensuring efficient sharing of resources across the different
// outbounds.
type Outbound struct {
	// Chooser is a peer chooser for outbound requests.
	Chooser yarpc.Chooser

	// Dialer is an alternative to specifying a Chooser.
	// The outbound will dial the address specified in the URL.
	Dialer yarpc.Dialer

	// URL specifies the template for the URL this outbound makes requests to.
	// For yarpc.Chooser-based outbounds, the peer (host:port) spection of the
	// URL may vary from call to call but the REST will remain unchanged.
	// For single-peer outbounds, the URL will be used as-is.
	URL *url.URL

	// Tracer attaches a tracer for the outbound.
	Tracer opentracing.Tracer

	// Headers to add to all outgoing requests.
	Headers http.Header

	// legacyResponseError forces the legacy behavior for HTTP response errors.
	// Outbound calls will not have the Rpc-Both-Response-Error header, so servers
	// will respond with error messages in the response body instead of the
	// Rpc-Error-Message header.
	// This is for tests only.
	legacyResponseError bool
}

// Call makes a HTTP request
func (o *Outbound) Call(ctx context.Context, yRequest *yarpc.Request, buf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if yRequest == nil {
		return nil, nil, yarpcerror.InvalidArgumentErrorf("request for http unary outbound was nil")
	}

	return o.call(ctx, yRequest, buf)
}

func (o *Outbound) call(ctx context.Context, yRequest *yarpc.Request, buf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	start := time.Now()
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, nil, yarpcerror.Newf(yarpcerror.CodeInvalidArgument, "missing context deadline")
	}
	ttl := deadline.Sub(start)

	hRequest, err := o.createRequest(yRequest, buf)
	if err != nil {
		return nil, nil, err
	}
	hRequest.Header = applicationHeaders.ToHTTPHeaders(yRequest.Headers, nil)
	ctx, hRequest, span, err := o.withOpentracingSpan(ctx, hRequest, yRequest, start)
	if err != nil {
		return nil, nil, err
	}
	defer span.Finish()

	hRequest = o.withCoreHeaders(hRequest, yRequest, ttl)
	hRequest = hRequest.WithContext(ctx)

	hResponse, err := o.roundTrip(hRequest, yRequest, start)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(opentracinglog.String("event", err.Error()))
		return nil, nil, err
	}

	span.SetTag("http.status_code", hResponse.StatusCode)

	// Service name match validation, return yarpcerror.CodeInternal error if not match
	if match, responseService := checkServiceMatch(yRequest.Service, hResponse.Header); !match {
		return nil, nil, yarpctracing.UpdateSpanWithErr(span,
			yarpcerror.InternalErrorf("service name sent from the request "+
				"does not match the service name received in the response, sent %q, got: %q", yRequest.Service, responseService))
	}

	yResponse := &yarpc.Response{
		Headers:          applicationHeaders.FromHTTPHeaders(hResponse.Header, yarpc.NewHeaders()),
		ApplicationError: hResponse.Header.Get(ApplicationStatusHeader) == ApplicationErrorStatus,
	}
	body, err := ioutil.ReadAll(hResponse.Body)
	if err != nil {
		return nil, nil, err
	}
	responseBuf := yarpc.NewBufferBytes(body)

	bothResponseError := hResponse.Header.Get(BothResponseErrorHeader) == AcceptTrue
	if bothResponseError && !o.legacyResponseError {
		if hResponse.StatusCode >= 300 {
			return yResponse, responseBuf, getYARPCErrorFromResponse(hResponse, true)
		}
		return yResponse, responseBuf, nil
	}
	if hResponse.StatusCode >= 200 && hResponse.StatusCode < 300 {
		return yResponse, responseBuf, nil
	}
	return nil, nil, getYARPCErrorFromResponse(hResponse, false)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, yRequest *yarpc.Request) (*httpPeer, func(error), error) {
	var peer yarpc.Peer
	var onFinish func(error)
	var err error

	if o.Chooser != nil {
		peer, onFinish, err = o.Chooser.Choose(ctx, yRequest)
		if err != nil {
			return nil, nil, err
		}
	} else if o.Dialer != nil && o.URL != nil {
		id := yarpc.Address(o.URL.Host)
		peer, err = o.Dialer.RetainPeer(id, yarpc.NopSubscriber)
		if err != nil {
			return nil, nil, err
		}
		err = o.Dialer.ReleasePeer(id, yarpc.NopSubscriber)
		if err != nil {
			return nil, nil, err
		}
		onFinish = nopFinish
	} else {
		// TODO capture error type
		return nil, nil, fmt.Errorf("HTTP Outbound must have either Chooser or Dialer and URL to make a Call")
	}

	hp, ok := peer.(*httpPeer)
	if !ok {
		return nil, nil, yarpcpeer.ErrInvalidPeerConversion{
			Peer:         peer,
			ExpectedType: "*httpPeer",
		}
	}

	return hp, onFinish, nil
}

func nopFinish(error) {}

func (o *Outbound) createRequest(yRequest *yarpc.Request, requestBuf *yarpc.Buffer) (*http.Request, error) {
	url := defaultURL
	if o.URL != nil {
		url = o.URL
	}
	return http.NewRequest("POST", url.String(), requestBuf)
}

func (o *Outbound) withOpentracingSpan(ctx context.Context, req *http.Request, yRequest *yarpc.Request, start time.Time) (context.Context, *http.Request, opentracing.Span, error) {
	// Apply HTTP Context headers for tracing and baggage carried by tracing.
	tracer := o.Tracer
	if tracer == nil {
		tracer = opentracing.GlobalTracer()
	}
	var parent opentracing.SpanContext // ok to be nil
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parent = parentSpan.Context()
	}
	tags := opentracing.Tags{
		"rpc.caller":    yRequest.Caller,
		"rpc.service":   yRequest.Service,
		"rpc.encoding":  yRequest.Encoding,
		"rpc.transport": "http",
	}
	for k, v := range yarpctracing.Tags {
		tags[k] = v
	}
	span := tracer.StartSpan(
		yRequest.Procedure,
		opentracing.StartTime(start),
		opentracing.ChildOf(parent),
		tags,
	)
	ext.PeerService.Set(span, yRequest.Service)
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

func (o *Outbound) withCoreHeaders(req *http.Request, yRequest *yarpc.Request, ttl time.Duration) *http.Request {
	// Add default headers to all requests.
	for k, vs := range o.Headers {
		if strings.HasPrefix(strings.ToLower(k), "rpc-") {
			panic(fmt.Errorf(
				"invalid header name %q: "+
					`headers starting with "Rpc-" are reserved by YARPC`, k))
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	req.Header.Set(CallerHeader, yRequest.Caller)
	req.Header.Set(ServiceHeader, yRequest.Service)
	req.Header.Set(ProcedureHeader, yRequest.Procedure)
	if ttl != 0 {
		req.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))
	}
	if yRequest.ShardKey != "" {
		req.Header.Set(ShardKeyHeader, yRequest.ShardKey)
	}
	if yRequest.RoutingKey != "" {
		req.Header.Set(RoutingKeyHeader, yRequest.RoutingKey)
	}
	if yRequest.RoutingDelegate != "" {
		req.Header.Set(RoutingDelegateHeader, yRequest.RoutingDelegate)
	}

	encoding := string(yRequest.Encoding)
	if encoding != "" {
		req.Header.Set(EncodingHeader, encoding)
	}

	if !o.legacyResponseError {
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
			return yarpcerror.Newf(yarpcerror.CodeInternal, err.Error())
		}
		contents = string(contentsBytes)
		if err := response.Body.Close(); err != nil {
			return yarpcerror.Newf(yarpcerror.CodeInternal, err.Error())
		}
	}
	// use the status code if we can't get a code from the headers
	code := statusCodeToBestCode(response.StatusCode)
	if errorCodeText := response.Header.Get(ErrorCodeHeader); errorCodeText != "" {
		var errorCode yarpcerror.Code
		// TODO: what to do with error?
		if err := errorCode.UnmarshalText([]byte(errorCodeText)); err == nil {
			code = errorCode
		}
	}
	return internalyarpcerror.NewWithNamef(
		code,
		response.Header.Get(ErrorNameHeader),
		strings.TrimSuffix(contents, "\n"),
	)
}

// Only does verification if there is a response header
func checkServiceMatch(requestService string, resHeaders http.Header) (bool, string) {
	serviceName := resHeaders.Get(ServiceHeader)
	return serviceName == "" || serviceName == requestService, serviceName
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
// The peer chooser for raw HTTP requests will receive a yarpc.Request with no body.
//
// OpenTracing information must be added manually, before this call, to support context propagation.
func (o *Outbound) RoundTrip(hRequest *http.Request) (*http.Response, error) {
	return o.roundTrip(hRequest, nil /* yRequest */, time.Now())
}

func (o *Outbound) roundTrip(hRequest *http.Request, yRequest *yarpc.Request, start time.Time) (*http.Response, error) {
	ctx := hRequest.Context()

	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, yarpcerror.Newf(
			yarpcerror.CodeInvalidArgument,
			"missing context deadline")
	}
	ttl := deadline.Sub(start)

	// When sending requests through the RoundTrip method, we construct the
	// transport request from the HTTP headers as if it were an inbound
	// request.
	// The API for setting transport metadata for an outbound request when
	// using the go stdlib HTTP client is to use headers as the YAPRC HTTP
	// transport header conventions.
	if yRequest == nil {
		yRequest = &yarpc.Request{
			Caller:          hRequest.Header.Get(CallerHeader),
			Service:         hRequest.Header.Get(ServiceHeader),
			Encoding:        yarpc.Encoding(hRequest.Header.Get(EncodingHeader)),
			Procedure:       hRequest.Header.Get(ProcedureHeader),
			ShardKey:        hRequest.Header.Get(ShardKeyHeader),
			RoutingKey:      hRequest.Header.Get(RoutingKeyHeader),
			RoutingDelegate: hRequest.Header.Get(RoutingDelegateHeader),
			Headers:         applicationHeaders.FromHTTPHeaders(hRequest.Header, yarpc.Headers{}),
		}
	}

	p, onFinish, err := o.getPeerForRequest(ctx, yRequest)
	if err != nil {
		return nil, err
	}

	hres, err := o.doWithPeer(ctx, hRequest, yRequest, start, ttl, p)
	// Call the onFinish method before returning (with the error from call with peer)
	onFinish(err)
	return hres, err
}

func (o *Outbound) doWithPeer(
	ctx context.Context,
	hRequest *http.Request,
	yRequest *yarpc.Request,
	start time.Time,
	ttl time.Duration,
	p *httpPeer,
) (*http.Response, error) {
	hRequest.URL.Host = p.addr

	response, err := p.dialer.client.Do(hRequest.WithContext(ctx))

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
			return nil, yarpcerror.Newf(
				yarpcerror.CodeDeadlineExceeded,
				"client timeout for procedure %q of service %q after %v",
				yRequest.Procedure, yRequest.Service, end.Sub(start))
		}

		// Note that the connection may have been lost so the peer connection
		// maintenance loop resumes probing for availability.
		p.OnDisconnected()

		return nil, yarpcerror.Newf(yarpcerror.CodeUnknown, "unknown error from http client: %s", err.Error())
	}

	return response, nil
}
