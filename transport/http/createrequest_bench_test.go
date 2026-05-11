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

package http

// Benchmarks for the createRequest hot path — opt #3: URL string caching.
//
// createRequest() is called on every outbound HTTP RPC. It does:
//   newURL := *o.urlTemplate          // copies url.URL struct (4 strings + int64 + ...)
//   http.NewRequest("POST", newURL.String(), body)  // String() allocates
//
// The URL template is immutable after Outbound construction, so we can cache
// the string and avoid the copy + allocation per call.
//
// Run baseline:
//   go test -bench=BenchmarkCreateRequest -benchmem -count=6 -run='^$' ./transport/http/ > before.txt
//
// After caching urlTemplate.String() in Outbound.urlStr:
//   go test -bench=BenchmarkCreateRequest -benchmem -count=6 -run='^$' ./transport/http/ > after.txt
//
// Compare:
//   benchstat before.txt after.txt

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"go.uber.org/yarpc/api/transport"
)

var _benchBody = bytes.Repeat([]byte("x"), 1<<10)

// BenchmarkCreateRequest_Current benchmarks the current createRequest()
// implementation: copies url.URL and calls String() on every call.
// This outbound is constructed directly (not via setURLTemplate) to simulate
// the pre-optimization path — urlStr is empty, so the fallback runs.
func BenchmarkCreateRequest_Current(b *testing.B) {
	urlTemplate, _ := url.Parse("http://localhost:8080")
	out := &Outbound{urlTemplate: urlTemplate}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hreq, err := out.createRequest(makeReq(), make(http.Header))
		if err != nil {
			b.Fatal(err)
		}
		// drain so GC doesn't hold body refs
		_, _ = io.Copy(io.Discard, hreq.Body)
	}
}

// BenchmarkCreateRequest_WithCache benchmarks createRequest() when urlStr is
// pre-populated (the post-optimization path via setURLTemplate/NewOutbound).
func BenchmarkCreateRequest_WithCache(b *testing.B) {
	tr := NewTransport()
	out := tr.NewSingleOutbound("http://localhost:8080")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hreq, err := out.createRequest(makeReq(), make(http.Header))
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, hreq.Body)
	}
}

// BenchmarkURLCopy isolates just the url.URL copy + String() cost —
// the part we want to eliminate.
func BenchmarkURLCopy(b *testing.B) {
	urlTemplate, _ := url.Parse("http://my-service.prod.uber.internal:8080/v1")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newURL := *urlTemplate
		_ = newURL.String()
	}
}

// BenchmarkURLStringCached benchmarks the target implementation:
// using a pre-computed URL string avoids the url.URL copy and String() alloc.
func BenchmarkURLStringCached(b *testing.B) {
	urlTemplate, _ := url.Parse("http://my-service.prod.uber.internal:8080/v1")
	cachedStr := urlTemplate.String() // computed once at construction
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cachedStr // just a string reference, zero cost
	}
}

func makeReq() *transport.Request {
	return &transport.Request{
		Caller:    "myservice",
		Service:   "downstream",
		Encoding:  "proto",
		Procedure: "MyService/MyMethod",
		Body:      bytes.NewReader(_benchBody),
	}
}
