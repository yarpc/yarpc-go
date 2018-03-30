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
	"net/http"
	"time"

	"go.uber.org/yarpc/api/transport"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/yarpcerrors"
)

// RoundTrip implements the http.RoundTripper interface, making a YARPC HTTP outbound suitable as a
// Transport when constructing an HTTP Client. An HTTP client is suitable only for relative paths to
// a single outbound service.
//
// The HTTP outbound refuses to send HTTP requests that have a fully qualified path, since it cannot
// respect the host and protocol portions of the URL, instead routing through the outbound peer
// chooser. A request that specifies a host or protocol will return an error.
//
// Sample usage:
//
//  client := http.Client{Transport: outbound}
//
// Thereafter use the Golang standard library HTTP to send requests with this client.
//
//  ctx := context.Background()
//  ctx, cancel := context.WithTimeout(ctx, time.Second)
//  defer cancel()
//  req := http.NewRequest("GET", "http://example.com/", nil)
//  req = req.WithContext(ctx)
//  res, err := client.Do(req)
//
// All requests must have a deadline on the context.
// The peer chooser for raw HTTP requests will receive a blank YARPC transport.Request, which is
// sufficient for load balancers like peer/pendingheap (fewest-pending-requests) and peer/roundrobin
// (round-robin).
func (o *Outbound) RoundTrip(hreq *http.Request) (*http.Response, error) {
	return o.roundTrip(hreq, nil, time.Now())
}

func (o *Outbound) roundTrip(hreq *http.Request, treq *transport.Request, start time.Time) (*http.Response, error) {
	ctx := hreq.Context()

	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, yarpcerrors.Newf(
			yarpcerrors.CodeInvalidArgument,
			"missing context deadline")
	}
	ttl := deadline.Sub(start)

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
			"error waiting for http unary outbound to start for service: %s",
			treq.Service)
	}
	p, onFinish, err := o.getPeerForRequest(ctx, treq)
	if err != nil {
		return nil, err
	}

	hres, err := o.doWithPeer(ctx, hreq, treq, start, ttl, p)

	// Call the onFinish method right before returning (with the error from call with peer)
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
) (*http.Response, error) {
	hreq.URL.Host = p.HostPort()

	response, err := o.transport.client.Do(hreq.WithContext(ctx))

	if err != nil {
		// Workaround borrowed from ctxhttp until
		// https://github.com/golang/go/issues/17711 is resolved.
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
		if err == context.DeadlineExceeded {
			end := time.Now()
			return nil, yarpcerrors.Newf(
				yarpcerrors.CodeDeadlineExceeded,
				"client timeout for procedure %q of service %q after %v",
				treq.Procedure, treq.Service, end.Sub(start))
		}

		// Note that the connection may have been lost so the peer connection
		// maintenance loop resumes probing for availability.
		p.OnDisconnected()

		return nil, yarpcerrors.Newf(yarpcerrors.CodeUnknown, "unknown error from http client: %s", err.Error())
	}

	return response, nil
}
