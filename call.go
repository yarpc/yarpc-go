// Copyright (c) 2025 Uber Technologies, Inc.
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

package yarpc

import (
	"context"

	"go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
)

// CallOption defines options that may be passed in at call sites to other
// services.
//
// These may be used to add or alter the request.
type CallOption encoding.CallOption

// ResponseHeaders specifies that headers received in response to this request
// should replace the given map.
//
// Header keys in the map are normalized using the CanonicalizeHeaderKey
// function.
//
//	var resHeaders map[string]string
//	resBody, err := client.SetValue(ctx, key, value, yarpc.ResponseHeaders(&resHeaders))
//	value, ok := resHeaders[yarpc.CanonicalizeHeaderKey("foo")]
//
// Note that the map is replaced completely. Entries it had before making the
// call will not be available afterwards.
//
//	headers := map[string]string{"hello": "world"}
//	resBody, err := client.SetValue(ctx, key, value, yarpc.ResponseHeaders(&headers))
//	_, ok := headers["hello"]
//	fmt.Println(ok)  // false
func ResponseHeaders(h *map[string]string) CallOption {
	return CallOption(encoding.ResponseHeaders(h))
}

// WithHeader adds a new header to the request. Header keys are case
// insensitive.
//
//	_, err := client.GetValue(ctx, reqBody, yarpc.WithHeader("Token", "10"))
//	// ==> {"token": "10"}
//
// If multiple entries have the same normalized header name, newer entries
// override older ones.
func WithHeader(k, v string) CallOption {
	return CallOption(encoding.WithHeader(k, v))
}

// WithShardKey sets the shard key for the request.
func WithShardKey(sk string) CallOption {
	return CallOption(encoding.WithShardKey(sk))
}

// WithRoutingKey sets the routing key for the request.
func WithRoutingKey(rk string) CallOption {
	return CallOption(encoding.WithRoutingKey(rk))
}

// WithRoutingDelegate sets the routing delegate for the request.
func WithRoutingDelegate(rd string) CallOption {
	return CallOption(encoding.WithRoutingDelegate(rd))
}

// Call provides information about the current request inside handlers. An
// instance of Call for the current request can be obtained by calling
// CallFromContext on the request context.
//
//	func Get(ctx context.Context, req *GetRequest) (*GetResponse, error) {
//		call := yarpc.CallFromContext(ctx)
//		fmt.Println("Received request from", call.Caller())
//		if err := call.WriteResponseHeader("hello", "world"); err != nil {
//			return nil, err
//		}
//		return response, nil
//	}
type Call encoding.Call

// CallFromContext retrieves information about the current incoming request
// from the given context. Returns nil if the context is not a valid request
// context.
//
// The object is valid only as long as the request is ongoing.
//
// # Testing
//
// To test functions which use CallFromContext, use yarpctest.ContextWithCall
// to build contexts compatible with this function.
func CallFromContext(ctx context.Context) *Call {
	return (*Call)(encoding.CallFromContext(ctx))
}

// WriteResponseHeader writes headers to the response of this call.
// Calling this method may mutate the underlying struct.
// The current implementation is not safe for concurrent use by multiple goroutines.
func (c *Call) WriteResponseHeader(k, v string) error {
	return (*encoding.Call)(c).WriteResponseHeader(k, v)
}

// Caller returns the name of the service making this request.
func (c *Call) Caller() string {
	return (*encoding.Call)(c).Caller()
}

// Service returns the name of the service being called.
func (c *Call) Service() string {
	return (*encoding.Call)(c).Service()
}

// Transport returns the name of the transport being called.
func (c *Call) Transport() string {
	return (*encoding.Call)(c).Transport()
}

// Procedure returns the name of the procedure being called.
func (c *Call) Procedure() string {
	return (*encoding.Call)(c).Procedure()
}

// Encoding returns the encoding for this request.
func (c *Call) Encoding() transport.Encoding {
	return (*encoding.Call)(c).Encoding()
}

// Header returns the value of the given request header provided with the
// request.
func (c *Call) Header(k string) string {
	return (*encoding.Call)(c).Header(k)
}

// OriginalHeader returns the value of the given request header provided with the
// request. The getter is suitable for transport like TChannel that hides
// certain headers by default eg: the ones starting with $
func (c *Call) OriginalHeader(k string) string {
	return (*encoding.Call)(c).OriginalHeader(k)
}

// OriginalHeaders returns a copy of the given request headers provided with the request.
// The header key are not canonicalized and suitable for case-sensitive transport like TChannel.
func (c *Call) OriginalHeaders() map[string]string {
	return (*encoding.Call)(c).OriginalHeaders()
}

// HeaderNames returns a sorted list of the names of user defined headers
// provided with this request.
func (c *Call) HeaderNames() []string {
	return (*encoding.Call)(c).HeaderNames()
}

// ShardKey returns the shard key for this request.
func (c *Call) ShardKey() string {
	return (*encoding.Call)(c).ShardKey()
}

// RoutingKey returns the routing key for this request.
func (c *Call) RoutingKey() string {
	return (*encoding.Call)(c).RoutingKey()
}

// RoutingDelegate returns the routing delegate for this request.
func (c *Call) RoutingDelegate() string {
	return (*encoding.Call)(c).RoutingDelegate()
}

// CallerProcedure returns the name of the procedure from the service making this request.
func (c *Call) CallerProcedure() string {
	return (*encoding.Call)(c).CallerProcedure()
}

// StreamOption defines options that may be passed in at streaming function
// call sites.
//
// These may be used to add or alter individual stream calls.
type StreamOption encoding.StreamOption
