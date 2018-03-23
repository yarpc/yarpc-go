package http

import (
	"context"
	"net/http"
	"time"

	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/yarpc/api/transport"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/yarpcerrors"
	"github.com/opentracing/opentracing-go"
)

// Do makes an old school HTTP request
//
//  hreq, err := http.NewRequest("GET", "http://example.com/path", nil)
//  treq := &transport.Request{} // ShardKey anyone?
//  hres, err := o.Do(ctx, hreq, treq)
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

func (o *Outbound) do(ctx context.Context, hreq *http.Request, treq *transport.Request, start time.Time, ttl time.Duration) (*http.Response, error) {
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
	var err error
	var span opentracing.Span
	ctx, hreq, span, err = o.withOpentracingSpan(ctx, hreq, treq, start)

	hreq.Header = applicationHeaders.ToHTTPHeaders(treq.Headers, nil)
	if err != nil {
		return nil, err
	}
	defer span.Finish()

	hres, err := p.transport.client.Do(hreq.WithContext(ctx))

	if err != nil {
		// Workaround borrowed from ctxhttp until
		// https://github.com/golang/go/issues/17711 is resolved.
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}

		span.SetTag("error", true)
		span.LogFields(opentracinglog.String("event", err.Error()))
		if err == context.DeadlineExceeded {
			// Note that the connection experienced a time out, which may
			// indicate that the connection is half-open, that the destination
			// died without sending a TCP FIN packet.

			// TODO: add it after PR lands on master
			//p.OnSuspect()

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

	span.SetTag("http.status_code", hres.StatusCode)

	return hres, nil
}
