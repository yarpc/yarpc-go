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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/yarpc/v2"
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

// Call makes an HTTP request.
//
// If the outbound has a Chooser, the outbound will use the chooser to obtain a
// peer for the duration of the request.
// Assume that the Chooser ignores the req.Peer identifier unless the Chooser
// specifies otherwise a custom behavior.
// The Chooser implementation is free to interpret the req.Peer as a hint, a
// requirement, or ignore it altogether.
//
// Otherwise, if the request has a specified Peer, the outbound will use the
// Dialer to retain that peer for the duration of the request.
//
// Otherwise, the outbound will use the Dialer to retain the peer identified
// by the Host of the configured URL for the duration of the request.
func (o *Outbound) Call(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if req == nil {
		return nil, nil, yarpcerror.InvalidArgumentErrorf("request for http unary outbound was nil")
	}

	return o.call(ctx, req, reqBuf)
}

func (o *Outbound) call(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	start := time.Now()
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, nil, yarpcerror.New(yarpcerror.CodeInvalidArgument, "missing context deadline")
	}
	ttl := deadline.Sub(start)

	httpReq, err := o.createRequest(req, reqBuf)
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header = applicationHeaders.ToHTTPHeaders(req.Headers, nil)
	ctx, httpReq, span, err := o.withOpentracingSpan(ctx, httpReq, req, start)
	if err != nil {
		return nil, nil, err
	}
	defer span.Finish()

	httpReq = o.withCoreHeaders(httpReq, req, ttl).WithContext(ctx)
	httpRes, err := o.roundTrip(httpReq, req, start)
	if err != nil {
		return nil, nil, yarpctracing.UpdateSpanWithErr(span, err)
	}

	span.SetTag("http.status_code", httpRes.StatusCode)

	// Service name match validation, return yarpcerror.CodeInternal error if not match
	if match, responseService := checkServiceMatch(req.Service, httpRes.Header); !match {
		return nil, nil, yarpctracing.UpdateSpanWithErr(span,
			yarpcerror.InternalErrorf("service name sent from the request "+
				"does not match the service name received in the response, sent %q, got: %q", req.Service, responseService))
	}

	res := &yarpc.Response{
		Peer:    yarpc.Address(httpRes.Header.Get(PeerHeader)),
		Headers: applicationHeaders.FromHTTPHeaders(httpRes.Header, yarpc.NewHeaders()),
	}

	resBuf, err := readCloserToBuffer(httpRes.Body)
	if err != nil {
		return nil, nil, err
	}

	var appErr error
	if httpRes.Header.Get(ApplicationStatusHeader) == ApplicationErrorStatus {
		appErr = getYARPCErrorFromResponse(httpRes, resBuf, true)
		errorInfo := yarpcerror.ExtractInfo(appErr)
		res.ApplicationErrorInfo = &errorInfo
	}

	bothResponseError := httpRes.Header.Get(BothResponseErrorHeader) == AcceptTrue
	if bothResponseError && !o.legacyResponseError {
		if httpRes.StatusCode >= 300 {
			// TODO: This is a bit odd; we set the error in response AND return it.
			// However, to preserve the current behavior of YARPC, this is
			// necessary. This is most likely where the error details will be added,
			// so we expect this to change.
			return res, resBuf, appErr
		}
		return res, resBuf, nil
	}
	if httpRes.StatusCode >= 200 && httpRes.StatusCode < 300 {
		return res, resBuf, nil
	}
	return nil, nil, getYARPCErrorFromResponse(httpRes, resBuf, false)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, req *yarpc.Request) (*httpPeer, func(error), error) {
	var (
		peer     yarpc.Peer
		onFinish func(error)
		err      error
	)
	if o.Chooser != nil {
		peer, onFinish, err = o.Chooser.Choose(ctx, req)
	} else if req.Peer != nil {
		peer, onFinish, err = o.getEphemeralPeer(req.Peer)
	} else if o.URL != nil {
		id := yarpc.Address(o.URL.Host)
		peer, onFinish, err = o.getEphemeralPeer(id)
	} else {
		return nil, nil, yarpcerror.FailedPreconditionErrorf("HTTP outbound must have a chooser or URL with host, or request must address a specific peer")
	}

	if err != nil {
		return nil, nil, err
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

func (o *Outbound) getEphemeralPeer(id yarpc.Identifier) (yarpc.Peer, func(error), error) {
	if o.Dialer == nil {
		return nil, nil, yarpcpeer.ErrMissingDialer{Transport: "http"}
	}
	peer, err := o.Dialer.RetainPeer(id, yarpc.NopSubscriber)
	if err != nil {
		return nil, nil, err
	}
	err = o.Dialer.ReleasePeer(id, yarpc.NopSubscriber)
	if err != nil {
		return nil, nil, err
	}
	return peer, nopFinish, nil
}

func nopFinish(error) {}

func (o *Outbound) createRequest(req *yarpc.Request, reqBuf *yarpc.Buffer) (*http.Request, error) {
	url := defaultURL
	if o.URL != nil {
		url = o.URL
	}
	return http.NewRequest("POST", url.String(), reqBuf)
}

func (o *Outbound) withOpentracingSpan(ctx context.Context, httpReq *http.Request, req *yarpc.Request, start time.Time) (context.Context, *http.Request, opentracing.Span, error) {
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
		"rpc.caller":    req.Caller,
		"rpc.service":   req.Service,
		"rpc.encoding":  req.Encoding,
		"rpc.transport": "http",
	}
	for k, v := range yarpctracing.Tags {
		tags[k] = v
	}
	span := tracer.StartSpan(
		req.Procedure,
		opentracing.StartTime(start),
		opentracing.ChildOf(parent),
		tags,
	)
	ext.PeerService.Set(span, req.Service)
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, httpReq.URL.String())
	ctx = opentracing.ContextWithSpan(ctx, span)

	err := tracer.Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(httpReq.Header),
	)

	return ctx, httpReq, span, err
}

func (o *Outbound) withCoreHeaders(httpReq *http.Request, req *yarpc.Request, ttl time.Duration) *http.Request {
	// Add default headers to all requests.
	for k, vs := range o.Headers {
		if strings.HasPrefix(strings.ToLower(k), "rpc-") {
			panic(fmt.Errorf(
				"invalid header name %q: "+
					`headers starting with "Rpc-" are reserved by YARPC`, k))
		}
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}

	httpReq.Header.Set(CallerHeader, req.Caller)
	httpReq.Header.Set(ServiceHeader, req.Service)
	httpReq.Header.Set(ProcedureHeader, req.Procedure)
	if ttl != 0 {
		httpReq.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))
	}
	if req.ShardKey != "" {
		httpReq.Header.Set(ShardKeyHeader, req.ShardKey)
	}
	if req.RoutingKey != "" {
		httpReq.Header.Set(RoutingKeyHeader, req.RoutingKey)
	}
	if req.RoutingDelegate != "" {
		httpReq.Header.Set(RoutingDelegateHeader, req.RoutingDelegate)
	}
	if req.Peer != nil {
		httpReq.Header.Set(PeerHeader, req.Peer.Identifier())
	}

	encoding := string(req.Encoding)
	if encoding != "" {
		httpReq.Header.Set(EncodingHeader, encoding)
	}

	if !o.legacyResponseError {
		httpReq.Header.Set(AcceptsBothResponseErrorHeader, AcceptTrue)
	}

	return httpReq
}

// readCloserToBuffer converts a readCloser to a yarpc.Buffer. This attempts to
// close the given readCloser. This is useful for both inbound and outbound
// requests.
//
// All returned erros are yarpcerror.CodeInternal.
func readCloserToBuffer(readCloser io.ReadCloser) (*yarpc.Buffer, error) {
	body, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, yarpcerror.New(yarpcerror.CodeInternal, err.Error())
	}

	if err := readCloser.Close(); err != nil {
		return nil, yarpcerror.New(yarpcerror.CodeInternal, err.Error())
	}
	return yarpc.NewBufferBytes(body), nil
}

func getYARPCErrorFromResponse(httpRes *http.Response, resBuf *yarpc.Buffer, bothResponseError bool) error {
	var contents string
	if bothResponseError {
		contents = httpRes.Header.Get(ErrorMessageHeader)
	} else {
		contents = resBuf.String()
	}
	// use the status code if we can't get a code from the headers
	code := statusCodeToBestCode(httpRes.StatusCode)
	if errorCodeText := httpRes.Header.Get(ErrorCodeHeader); errorCodeText != "" {
		var errorCode yarpcerror.Code
		// TODO: what to do with error?
		if err := errorCode.UnmarshalText([]byte(errorCodeText)); err == nil {
			code = errorCode
		}
	}
	return yarpcerror.New(
		code,
		strings.TrimSuffix(contents, "\n"),
		yarpcerror.WithName(httpRes.Header.Get(ErrorNameHeader)),
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
func (o *Outbound) RoundTrip(httpReq *http.Request) (*http.Response, error) {
	return o.roundTrip(httpReq, nil /* req */, time.Now())
}

func (o *Outbound) roundTrip(httpReq *http.Request, req *yarpc.Request, start time.Time) (*http.Response, error) {
	ctx := httpReq.Context()

	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, yarpcerror.New(
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
	if req == nil {
		req = &yarpc.Request{
			Caller:          httpReq.Header.Get(CallerHeader),
			Service:         httpReq.Header.Get(ServiceHeader),
			Encoding:        yarpc.Encoding(httpReq.Header.Get(EncodingHeader)),
			Procedure:       httpReq.Header.Get(ProcedureHeader),
			ShardKey:        httpReq.Header.Get(ShardKeyHeader),
			RoutingKey:      httpReq.Header.Get(RoutingKeyHeader),
			RoutingDelegate: httpReq.Header.Get(RoutingDelegateHeader),
			Headers:         applicationHeaders.FromHTTPHeaders(httpReq.Header, yarpc.Headers{}),
		}
	}

	p, onFinish, err := o.getPeerForRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	hres, err := o.doWithPeer(ctx, httpReq, req, start, ttl, p)
	// Call the onFinish method before returning (with the error from call with peer)
	onFinish(err)
	return hres, err
}

func (o *Outbound) doWithPeer(
	ctx context.Context,
	httpReq *http.Request,
	req *yarpc.Request,
	start time.Time,
	ttl time.Duration,
	p *httpPeer,
) (*http.Response, error) {
	httpReq.URL.Host = p.addr

	response, err := p.dialer.client.Do(httpReq.WithContext(ctx))

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
			return nil, yarpcerror.New(
				yarpcerror.CodeDeadlineExceeded,
				fmt.Sprintf(
					"client timeout for procedure %q of service %q after %v",
					req.Procedure, req.Service, end.Sub(start),
				),
			)
		}

		// Note that the connection may have been lost so the peer connection
		// maintenance loop resumes probing for availability.
		p.OnDisconnected()

		return nil, yarpcerror.New(yarpcerror.CodeUnknown, fmt.Sprintf("unknown error from http client: %s", err.Error()))
	}

	return response, nil
}
