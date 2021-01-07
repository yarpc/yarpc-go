// Copyright (c) 2021 Uber Technologies, Inc.
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

// CallOption defines options that may be passed in at call sites to other
// services.
//
// Encoding authors should accept yarpc.CallOptions and convert them to
// encoding.CallOptions to use with NewOutboundCall. This will keep the
// API for service authors simple.
type CallOption struct{ apply func(*OutboundCall) }

// ResponseHeaders specifies that headers received in response to this request
// should replace the given map.
func ResponseHeaders(h *map[string]string) CallOption {
	return CallOption{func(o *OutboundCall) { o.responseHeaders = h }}
}

// WithHeader adds a new header to the request.
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
