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

package encoding

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

func TestNilCall(t *testing.T) {
	call := CallFromContext(context.Background())
	require.Nil(t, call)

	assert.Equal(t, "", call.Caller())
	assert.Equal(t, "", call.Service())
	assert.Equal(t, "", call.Transport())
	assert.Equal(t, "", string(call.Encoding()))
	assert.Equal(t, "", call.Procedure())
	assert.Equal(t, "", call.ShardKey())
	assert.Equal(t, "", call.RoutingKey())
	assert.Equal(t, "", call.RoutingDelegate())
	assert.Equal(t, "", call.CallerProcedure())
	assert.Equal(t, "", call.Header("foo"))
	assert.Equal(t, "", call.OriginalHeader("foo"))
	assert.Empty(t, call.HeaderNames())
	assert.Nil(t, call.OriginalHeaders())
	assert.Equal(t, 0, call.HeadersLen())
	assert.Equal(t, 0, call.OriginalHeadersLen())
	assert.Len(t, slices.Collect(call.HeaderNamesAll()), 0, "nil call should yield no headers")
	assert.Empty(t, maps.Collect(call.HeadersAll()), "nil call should yield no header key-value pairs")
	assert.Empty(t, maps.Collect(call.OriginalHeadersAll()), "nil call should yield no original header key-value pairs")

	assert.Error(t, call.WriteResponseHeader("foo", "bar"))
}

func TestReadFromRequest(t *testing.T) {
	ctx, icall := NewInboundCall(context.Background())
	icall.ReadFromRequest(&transport.Request{
		Service:         "service",
		Transport:       "transport",
		Caller:          "caller",
		Encoding:        transport.Encoding("raw"),
		Procedure:       "proc",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
		CallerProcedure: "cp",
		// later header's key/value takes precedence
		Headers: transport.NewHeaders().With("Foo", "Bar").With("foo", "bar"),
	})
	call := CallFromContext(ctx)
	require.NotNil(t, call)

	assert.Equal(t, "caller", call.Caller())
	assert.Equal(t, "service", call.Service())
	assert.Equal(t, "transport", call.Transport())
	assert.Equal(t, "raw", string(call.Encoding()))
	assert.Equal(t, "proc", call.Procedure())
	assert.Equal(t, "sk", call.ShardKey())
	assert.Equal(t, "rk", call.RoutingKey())
	assert.Equal(t, "rd", call.RoutingDelegate())
	assert.Equal(t, "bar", call.Header("foo"))
	assert.Equal(t, "bar", call.OriginalHeader("foo"))
	assert.Equal(t, "Bar", call.OriginalHeader("Foo"))
	assert.Equal(t, map[string]string{"Foo": "Bar", "foo": "bar"}, call.OriginalHeaders())
	assert.Equal(t, call.OriginalHeaders(), maps.Collect(call.OriginalHeadersAll()), "OriginalHeadersAll should match OriginalHeaders")
	assert.Equal(t, map[string]string{"foo": "bar"}, call.Headers())
	assert.Equal(t, call.Headers(), maps.Collect(call.HeadersAll()), "HeadersAll should match Headers")

	assert.Equal(t, "cp", call.CallerProcedure())
	assert.Len(t, call.HeaderNames(), 1)
	assert.Equal(t, 1, call.HeadersLen())
	assert.Equal(t, 2, call.OriginalHeadersLen())

	headerNames := call.HeaderNames()
	headerNamesFromIterator := slices.Collect(call.HeaderNamesAll())
	slices.Sort(headerNames)
	slices.Sort(headerNamesFromIterator)
	assert.Equal(t, headerNames, headerNamesFromIterator)
	assert.Equal(t, map[string]string{"foo": "bar"}, maps.Collect(call.HeadersAll()), "HeadersAll should return canonicalized headers")

	// Verify early break from iterators.
	for range call.HeaderNamesAll() {
		break
	}
	for range call.HeadersAll() {
		break
	}
	for range call.OriginalHeadersAll() {
		break
	}

	assert.NoError(t, call.WriteResponseHeader("foo2", "bar2"))
	assert.Equal(t, icall.resHeaders[0].k, "foo2")
	assert.Equal(t, icall.resHeaders[0].v, "bar2")
}

func TestReadFromRequestMeta(t *testing.T) {
	ctx, icall := NewInboundCall(context.Background())
	icall.ReadFromRequestMeta(&transport.RequestMeta{
		Service:         "service",
		Caller:          "caller",
		Transport:       "transport",
		Encoding:        transport.Encoding("raw"),
		Procedure:       "proc",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
		CallerProcedure: "cp",
		// later header's key/value takes precedence
		Headers: transport.NewHeaders().With("Foo", "Bar").With("foo", "bar"),
	})
	call := CallFromContext(ctx)
	require.NotNil(t, call)

	assert.Equal(t, "caller", call.Caller())
	assert.Equal(t, "service", call.Service())
	assert.Equal(t, "transport", call.Transport())
	assert.Equal(t, "raw", string(call.Encoding()))
	assert.Equal(t, "proc", call.Procedure())
	assert.Equal(t, "sk", call.ShardKey())
	assert.Equal(t, "rk", call.RoutingKey())
	assert.Equal(t, "rd", call.RoutingDelegate())
	assert.Equal(t, "cp", call.CallerProcedure())
	assert.Equal(t, "bar", call.Header("foo"))
	assert.Equal(t, "bar", call.OriginalHeader("foo"))
	assert.Equal(t, "Bar", call.OriginalHeader("Foo"))
	assert.Equal(t, map[string]string{"Foo": "Bar", "foo": "bar"}, call.OriginalHeaders())
	assert.Equal(t, call.OriginalHeaders(), maps.Collect(call.OriginalHeadersAll()), "OriginalHeadersAll should match OriginalHeaders")
	assert.Len(t, call.HeaderNames(), 1)

	assert.NoError(t, call.WriteResponseHeader("foo2", "bar2"))
	assert.Equal(t, icall.resHeaders[0].k, "foo2")
	assert.Equal(t, icall.resHeaders[0].v, "bar2")
}

func TestDisabledResponseHeaders(t *testing.T) {
	ctx, icall := NewInboundCallWithOptions(context.Background(), DisableResponseHeaders())
	icall.ReadFromRequest(&transport.Request{
		Service:         "service",
		Transport:       "transport",
		Caller:          "caller",
		Encoding:        transport.Encoding("raw"),
		Procedure:       "proc",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
		CallerProcedure: "cp",
		Headers:         transport.NewHeaders().With("foo", "bar"),
	})
	call := CallFromContext(ctx)
	require.NotNil(t, call)

	assert.Equal(t, "caller", call.Caller())
	assert.Equal(t, "service", call.Service())
	assert.Equal(t, "transport", call.Transport())
	assert.Equal(t, "raw", string(call.Encoding()))
	assert.Equal(t, "proc", call.Procedure())
	assert.Equal(t, "sk", call.ShardKey())
	assert.Equal(t, "rk", call.RoutingKey())
	assert.Equal(t, "rd", call.RoutingDelegate())
	assert.Equal(t, "cp", call.CallerProcedure())
	assert.Equal(t, "bar", call.Header("foo"))
	assert.Len(t, call.HeaderNames(), 1)

	assert.Error(t, call.WriteResponseHeader("foo", "bar"))
	assert.Nil(t, icall.resHeaders)
}

func BenchmarkCallHeaderNames(b *testing.B) {
	benchmarkSizes := []int{1, 2, 3, 4, 5, 6, 8, 10, 25, 50, 100}

	testCalls := make(map[int]*Call)
	for _, size := range benchmarkSizes {
		headers := transport.NewHeadersWithCapacity(size)
		for i := 0; i < size; i++ {
			headers = headers.With("header-"+strconv.Itoa(i), "value-"+strconv.Itoa(i))
		}
		ctx, icall := NewInboundCall(context.Background())
		icall.ReadFromRequest(&transport.Request{Headers: headers})
		testCalls[size] = CallFromContext(ctx)
	}

	// Benchmark HeaderNames (with sorting).
	b.Run("HeaderNames", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					call.HeaderNames()
				}
			})
		}
	})

	// Benchmark HeaderNamesAll (no sorting).
	b.Run("HeaderNamesAll", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Consume the iterator.
					for name := range call.HeaderNamesAll() {
						_ = name
					}
				}
			})
		}
	})

	// Benchmark OriginalHeaders (creates map copy).
	b.Run("OriginalHeaders", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					call.OriginalHeaders()
				}
			})
		}
	})

	// Benchmark OriginalHeadersAll (no copy).
	b.Run("OriginalHeadersAll", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Consume the iterator.
					for k, v := range call.OriginalHeadersAll() {
						_ = k
						_ = v
					}
				}
			})
		}
	})

	// Benchmark HeaderNames + Header.
	b.Run("HeaderNames + Header", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for _, name := range call.HeaderNames() {
						_ = call.Header(name)
					}
				}
			})
		}
	})

	// Benchmark HeadersAll.
	b.Run("HeaderNamesAll", func(b *testing.B) {
		for _, size := range benchmarkSizes {
			call := testCalls[size]
			b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Consume the iterator.
					for k, v := range call.HeadersAll() {
						_, _ = k, v
					}
				}
			})
		}
	})
}
