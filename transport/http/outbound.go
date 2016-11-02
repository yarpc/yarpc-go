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

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/peers"

	"github.com/uber-go/atomic"
)

var (
	errOutboundAlreadyStarted = errors.ErrOutboundAlreadyStarted("http.Outbound")
	errOutboundNotStarted     = errors.ErrOutboundNotStarted("http.Outbound")
)

// NewOutbound is deprecated
// TODO rename this func NewOutboundFromURL
func NewOutbound(urlStr string) transport.Outbound {
	agent := NewDefaultAgent()

	scheme, hostport, path := parseURL(urlStr)

	peerID := peers.NewPeerIdentifier(hostport)
	peerList := peers.NewSinglePeerList(peerID, agent)

	return NewPeerListOutbound(peerList, path, scheme)
}

func parseURL(urlStr string) (string, string, string) {
	parseURL, err := url.Parse(urlStr)

	if err != nil {
		return "", urlStr, ""
	}

	return parseURL.Scheme, parseURL.Host, parseURL.Path
}

// NewPeerListOutbound Builds a new HTTP outbound build around a PeerList for getting potential downstream hosts
// TODO rename this NewOutbound
func NewPeerListOutbound(peerList transport.PeerList, path, scheme string) transport.Outbound {
	return &outbound{
		started:  atomic.NewBool(false),
		PeerList: peerList,
		Path:     path,
		Scheme:   scheme,
	}
}

type outbound struct {
	started  *atomic.Bool
	Deps     transport.Deps
	PeerList transport.PeerList
	Path     string
	Scheme   string
}

func (o *outbound) Start(d transport.Deps) error {
	if o.started.Swap(true) {
		return errOutboundAlreadyStarted
	}
	o.Deps = d
	return o.PeerList.Start()
}

func (o *outbound) Stop() error {
	if !o.started.Swap(false) {
		return errOutboundNotStarted
	}
	return o.PeerList.Stop()
}

func (o *outbound) Call(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
	if !o.started.Load() {
		// panic because there's no recovery from this
		panic(errOutboundNotStarted)
	}
	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	peer, err := o.getPeerForRequest(ctx, treq)
	if err != nil {
		return nil, err
	}
	peer.IncPending()
	defer peer.DecPending()

	req, err := o.createRequest(peer, treq)
	if err != nil {
		return nil, err
	}

	req.Header = applicationHeaders.ToHTTPHeaders(treq.Headers, nil)
	ctx, req, span := o.withOpentracingSpan(ctx, req, treq, start)
	defer span.Finish()
	req = o.withCoreHeaders(req, treq, ttl)

	response, err := o.getHTTPClient(peer).Do(req.WithContext(ctx))

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

func (o *outbound) getPeerForRequest(ctx context.Context, treq *transport.Request) (*peers.HostPortPeer, error) {
	peer, err := o.PeerList.ChoosePeer(ctx, treq)
	if err != nil {
		return nil, err
	}

	hpPeer, ok := peer.(*peers.HostPortPeer)
	if ok {
		return hpPeer, nil

	}

	return nil, errors.ErrInvalidPeerConversion{
		Peer:         peer,
		ExpectedType: "*peers.HostPortPeer",
	}
}

func (o *outbound) createRequest(peer *peers.HostPortPeer, treq *transport.Request) (*http.Request, error) {
	reqURL := fmt.Sprintf("%s://%s%s", o.Scheme, peer.HostPort(), o.Path)
	return http.NewRequest("POST", reqURL, treq.Body)
}

func (o *outbound) withOpentracingSpan(ctx context.Context, req *http.Request, treq *transport.Request, start time.Time) (context.Context, *http.Request, opentracing.Span) {
	// Apply HTTP Context headers for tracing and baggage carried by tracing.
	tracer := o.Deps.Tracer()
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

func (o *outbound) withCoreHeaders(req *http.Request, treq *transport.Request, ttl time.Duration) *http.Request {
	req.Header.Set(CallerHeader, treq.Caller)
	req.Header.Set(ServiceHeader, treq.Service)
	req.Header.Set(ProcedureHeader, treq.Procedure)
	req.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))
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

func (o *outbound) getHTTPClient(peer *peers.HostPortPeer) *http.Client {
	agent := peer.GetAgent().(*Agent)
	return agent.client
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
