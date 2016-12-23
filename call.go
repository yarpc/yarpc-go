package yarpc

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// CallOption defines options that may be passed in at call sites to other
// services.
//
// These may be used to add or alter the request.
type CallOption struct{ apply func(*OutboundCall) }

type callHeader struct{ k, v string }

// OutboundCall represents an outgoing call.
//
// It holds any per-call options for a request. Encoding authors may use
// OutboundCall to hydrate Requests from call-site options.
type OutboundCall struct {
	// request attributes to fill if non-nil
	headers         []callHeader
	shardKey        *string
	routingKey      *string
	routingDelegate *string

	// If non-nil, response headers should be written here.
	responseHeaders *Headers
}

// NewOutboundCall constructs a new OutboundCall with the given options.
func NewOutboundCall(options ...CallOption) *OutboundCall {
	var call OutboundCall
	for _, opt := range options {
		opt.apply(&call)
	}
	return &call
}

// WriteToRequest fills the given request with request-specific options from
// the call.
//
// The context MAY be replaced by the OutboundCall.
func (c *OutboundCall) WriteToRequest(ctx context.Context, req *transport.Request) (context.Context, error) {
	for _, h := range c.headers {
		req.Headers = req.Headers.With(h.k, h.v)
	}

	if c.shardKey != nil {
		req.ShardKey = *c.shardKey
	}
	if c.routingKey != nil {
		req.RoutingKey = *c.routingKey
	}
	if c.routingDelegate != nil {
		req.RoutingDelegate = *c.routingDelegate
	}

	// NB(abg): context and error are unused for now but we want to leave room
	// for CallOptions which can fail or modify the context.
	return ctx, nil
}

// ReadFromResponse reads information from the response for this call.
//
// This should be called only if the request is unary.
func (c *OutboundCall) ReadFromResponse(ctx context.Context, res *transport.Response) (context.Context, error) {
	// We're not using ctx right now but we may in the future.
	if c.responseHeaders != nil {
		*c.responseHeaders = Headers(res.Headers)
	}

	// NB(abg): context and error are unused for now but we want to leave room
	// for CallOptions which can fail or modify the context.
	return ctx, nil
}

// ResponseHeaders specifies that headers received in response to this request
// should be fed into the given object.
//
// 	var resHeaders yarpc.Headers
// 	resBody, err := client.SetValue(ctx, key, value, yarpc.ResponseHeaders(&resHeaders))
func ResponseHeaders(h *Headers) CallOption {
	return CallOption{func(o *OutboundCall) { o.responseHeaders = h }}
}

// TODO(abg): Example tests to document the different options

// WithHeader adds a new header to the request.
//
// 	resBody, err := client.GetValue(ctx, reqBody, yarpc.WithHeader("Token", "10"))
func WithHeader(k, v string) CallOption {
	return CallOption{func(o *OutboundCall) {
		o.headers = append(o.headers, callHeader{k: k, v: v})
	}}
}

// WithShardKey sets the shard key for the request.
func WithShardKey(sk string) CallOption {
	return CallOption{func(o *OutboundCall) { o.shardKey = &sk }}
}

// WithRoutingKey sets the routing key for the request.
func WithRoutingKey(rk string) CallOption {
	return CallOption{func(o *OutboundCall) { o.routingKey = &rk }}
}

// WithRoutingDelegate sets the routing delegate for the request.
func WithRoutingDelegate(rd string) CallOption {
	return CallOption{func(o *OutboundCall) { o.routingDelegate = &rd }}
}
