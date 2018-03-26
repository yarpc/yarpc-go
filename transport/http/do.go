package http

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/yarpc/api/transport"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/yarpcerrors"
)

//RoundTrip implements the http.RoundTripper interface, making a YARPC HTTP outbound suitable as a
// Transport when constructing an HTTP Client. An HTTP client is suitable only for relative paths to
// a single outbound service.
//
//The HTTP outbound refuses to send HTTP requests that have a fully qualified path, since it cannot
// respect the host and protocol portions of the URL, instead routing through the outbound peer
// chooser. A request that specifies a host or protocol will return an error.
//
// Sample usage:
//
// client := http.Client{Transport: outbound}
// Thereafter use the Golang standard library HTTP to send requests with this client.
// ctx := context.Background()
// ctx, cancel = context.WithTimeout(ctx, time.Second)
// defer cancel()
// req := http.NewRequest("GET", "http://example.com/", nil)
// req = req.WithContext(ctx)
// res, err := client.Do(request)
//
// All requests must have a deadline on the context.
// The peer chooser for raw HTTP requests will receive a blank YARPC transport.Request, which is
// sufficient for load balancers like peer/pendingheap (fewest-pending-requests) and peer/roundrobin
// (round-robin).

func (o *Outbound) RoundTrip(hreq *http.Request) (*http.Response, error) {
	ctx := hreq.Context()
	treq := &transport.Request{}
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "error waiting for http unary outbound to start for service: %s", treq.Service)
	}

	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	return o.do(ctx, hreq, treq, start, ttl)
}

func (o *Outbound) do(
	ctx context.Context,
	hreq *http.Request,
	treq *transport.Request,
	start time.Time,
	ttl time.Duration,
) (*http.Response, error) {
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
	hreq.Header = applicationHeaders.ToHTTPHeaders(treq.Headers, nil)
	ctx, hreq, span, err := o.withOpentracingSpan(ctx, hreq, treq, start)
	if err != nil {
		return nil, err
	}
	defer span.Finish()

	hres, err := o.callWithTracing(ctx, treq, start, p, hreq, span)
	if err != nil {
		return nil, err
	}

	return hres, nil
}
