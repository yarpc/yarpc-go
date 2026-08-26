// Copyright (c) 2026 Uber Technologies, Inc.
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

package encoding

const (
	// RoutingDelegateCrosszoneHeaderKey is one of the cross-zone header for the routing delegate
	RoutingDelegateCrosszoneHeaderKey = "crosszone"

	// RoutingDelegateCrossregionHeaderKey is one of the cross-region header for the routing delegate
	RoutingDelegateCrossregionHeaderKey = "crossregion"

	// RoutingZoneGRPCHeaderKey is one of the cross-zone header for the zone of the routing key for gRPC.
	RoutingZoneGRPCHeaderKey = "x-uber-rpc-routing-zone"

	// RoutingRegionGRPCHeaderKey is one of the cross-region header for the region of the routing key for gRPC.
	RoutingRegionGRPCHeaderKey = "x-uber-rpc-routing-region"
)

// CallOption defines options that may be passed in at call sites to other
// services.
//
// Encoding authors should accept yarpc.CallOptions and convert them to
// encoding.CallOptions to use with NewOutboundCall. This will keep the
// API for service authors simple.

type CallOption struct {
	t               callOptionType
	key             string
	value           string
	responseHeaders *map[string]string
}

// callOptionType identifies which option a CallOption carries.
type callOptionType uint8

const (
	callOptionTypeUnknown callOptionType = iota
	callOptionTypeHeader
	callOptionTypeShardKey
	callOptionTypeRoutingKey
	callOptionTypeRoutingDelegate
	callOptionTypeResponseHeaders
)

// apply writes the option onto the given OutboundCall.
func (o CallOption) apply(call *OutboundCall) {
	switch o.t {
	case callOptionTypeHeader:
		call.headers = append(call.headers, keyValuePair{k: o.key, v: o.value})
	case callOptionTypeShardKey:
		v := o.value
		call.shardKey = &v
	case callOptionTypeRoutingKey:
		v := o.value
		call.routingKey = &v
	case callOptionTypeRoutingDelegate:
		v := o.value
		call.routingDelegate = &v
	case callOptionTypeResponseHeaders:
		call.responseHeaders = o.responseHeaders
	}
}

// ResponseHeaders specifies that headers received in response to this request
// should replace the given map.
func ResponseHeaders(h *map[string]string) CallOption {
	return CallOption{t: callOptionTypeResponseHeaders, responseHeaders: h}
}

// WithHeader adds a new header to the request.
func WithHeader(k, v string) CallOption {
	return CallOption{t: callOptionTypeHeader, key: k, value: v}
}

// WithShardKey sets the shard key for the request.
func WithShardKey(sk string) CallOption {
	return CallOption{t: callOptionTypeShardKey, value: sk}
}

// WithRoutingKey sets the routing key for the request.
func WithRoutingKey(rk string) CallOption {
	return CallOption{t: callOptionTypeRoutingKey, value: rk}
}

// WithRoutingDelegate sets the routing delegate for the request.
func WithRoutingDelegate(rd string) CallOption {
	return CallOption{t: callOptionTypeRoutingDelegate, value: rd}
}

// WithCrossZoneRoutingGRPC sets the cross zone routing header for the gRPC request.
func WithCrossZoneRoutingGRPC(routingZone string) CallOption {
	return WithHeader(RoutingZoneGRPCHeaderKey, routingZone)
}

// WithCrossRegionRoutingGRPC sets the cross region routing header for the gRPC request.
func WithCrossRegionRoutingGRPC(routingRegion string) CallOption {
	return WithHeader(RoutingRegionGRPCHeaderKey, routingRegion)
}
