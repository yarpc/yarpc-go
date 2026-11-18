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

import "testing"

var (
	_benchHeaderKey = "x-uber-source"
	_benchHeaderVal = "my-service"
	_benchShardKey  = "shard-42"
	_benchRouteKey  = "route-7"
	_benchRouteDel  = "delegate-3"
)

// BenchmarkNewOutboundCallHeaders measures a realistic header-only call: build
// the options and apply them onto an OutboundCall.
func BenchmarkNewOutboundCallHeaders(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(
			WithHeader(_benchHeaderKey, _benchHeaderVal),
			WithHeader(_benchRouteKey, _benchRouteDel),
			WithHeader(_benchShardKey, _benchHeaderVal),
		)
	}
}

// BenchmarkNewOutboundCallMixed measures a mix of option kinds on one call.
func BenchmarkNewOutboundCallMixed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(
			WithHeader(_benchHeaderKey, _benchHeaderVal),
			WithShardKey(_benchShardKey),
			WithRoutingKey(_benchRouteKey),
			WithRoutingDelegate(_benchRouteDel),
		)
	}
}

func BenchmarkNewOutboundCallWithHeader(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(WithHeader(_benchHeaderKey, _benchHeaderVal))
	}
}

func BenchmarkNewOutboundCallWithShardKey(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(WithShardKey(_benchShardKey))
	}
}

func BenchmarkNewOutboundCallWithRoutingKey(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(WithRoutingKey(_benchRouteKey))
	}
}

func BenchmarkNewOutboundCallWithRoutingDelegate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewOutboundCall(WithRoutingDelegate(_benchRouteDel))
	}
}

func BenchmarkNewOutboundCallResponseHeaders(b *testing.B) {
	b.ReportAllocs()
	var resHeaders map[string]string
	for i := 0; i < b.N; i++ {
		NewOutboundCall(ResponseHeaders(&resHeaders))
	}
}
