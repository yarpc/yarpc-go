package yarpc

import (
	"context"
	"errors"
	"sort"

	"go.uber.org/yarpc/api/transport"
)

type keyValuePair struct{ k, v string }

// CallOption defines options that may be passed in at call sites to other
// services.
//
// These may be used to add or alter the request.
type CallOption struct{ apply func(*OutboundCall) }

// OutboundCall is an outgoing call. It holds per-call options for a request.
//
// Encoding authors may use OutboundCall to provide a CallOption-based request
// customization mechanism, including returning response headers through
// ResponseHeaders.
type OutboundCall struct {
	// request attributes to fill if non-nil
	headers         []keyValuePair
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
		o.headers = append(o.headers, keyValuePair{k: k, v: v})
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

// InboundCall is an incoming call. It holds information about the inbound
// call and its response.
//
// Encoding authors may use InboundCall to provide information about the
// incoming request on the Context and receive response headers through
// WriteResponseHeader.
type InboundCall struct {
	resHeaders []keyValuePair
	req        *transport.Request
}

type inboundCallKey struct{} // context key for *InboundCall

// NewInboundCall builds a new InboundCall with the given context.
//
// A new context is returned and must be used in place of the original.
func NewInboundCall(ctx context.Context) (context.Context, *InboundCall) {
	call := &InboundCall{}
	return context.WithValue(ctx, inboundCallKey{}, call), call
}

// getInboundCall returns the inbound call on this context or nil.
func getInboundCall(ctx context.Context) (*InboundCall, bool) {
	call, ok := ctx.Value(inboundCallKey{}).(*InboundCall)
	return call, ok
}

// ReadFromRequest reads information from the given request.
//
// This information may be queried on the context using functions like Caller,
// Service, Procedure, etc.
func (ic *InboundCall) ReadFromRequest(req *transport.Request) error {
	// TODO(abg): Maybe we should copy attributes over so that changes to the
	// Request don't change the output.
	ic.req = req
	return nil
}

// WriteToResponse writes response information from the InboundCall onto the
// given ResponseWriter.
//
// If used, this must be called before writing the response body to the
// ResponseWriter.
func (ic *InboundCall) WriteToResponse(resw transport.ResponseWriter) error {
	var headers transport.Headers
	for _, h := range ic.resHeaders {
		headers = headers.With(h.k, h.v)
	}

	if headers.Len() > 0 {
		resw.AddHeaders(headers)
	}

	return nil
}

// WriteResponseHeader writes headers to the response of this request context.
//
// The operation will fail if the context is not an inbound request context.
func WriteResponseHeader(ctx context.Context, k, v string) error {
	call, ok := getInboundCall(ctx)
	if !ok {
		return errors.New("failed to write response header: " +
			"make sure that the context is a inbound request context")
	}

	call.resHeaders = append(call.resHeaders, keyValuePair{k: k, v: v})
	return nil
}

// Caller returns the name of the service making this request.
func Caller(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.Caller
	}
	return ""
}

// Service returns the name of the service being called.
func Service(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.Service
	}
	return ""
}

// Procedure returns the name of the procedure being called.
func Procedure(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.Procedure
	}
	return ""
}

// Encoding returns the encoding for this request.
func Encoding(ctx context.Context) transport.Encoding {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.Encoding
	}
	return ""
}

// Header returns the value of the given request header provided with the
// request.
func Header(ctx context.Context, k string) string {
	call, ok := getInboundCall(ctx)
	if !ok {
		return ""
	}

	if v, ok := call.req.Headers.Get(k); ok {
		return v
	}

	return ""
}

// HeaderNames returns a sorted list of the names of user defined headers
// provided with this request.
func HeaderNames(ctx context.Context) []string {
	call, ok := getInboundCall(ctx)
	if !ok {
		return nil
	}

	var names []string
	for k := range call.req.Headers.Items() {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ShardKey returns the shard key for this request.
func ShardKey(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.ShardKey
	}
	return ""
}

// RoutingKey returns the routing key for this request.
func RoutingKey(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.RoutingKey
	}
	return ""
}

// RoutingDelegate returns the routing delegate for this request.
func RoutingDelegate(ctx context.Context) string {
	if call, ok := getInboundCall(ctx); ok {
		return call.req.RoutingDelegate
	}
	return ""
}
