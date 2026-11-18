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

package connpool

import "testing"

var sinkResolved Config

// BenchmarkConfigResolverGet measures the Config call that TryScaleUp makes on
// every RPC. The live-provider cases are the ones a service with an
// object-config or flipr provider pays.
func BenchmarkConfigResolverGet(b *testing.B) {
	startup := baseConfig()
	enabled := boolPtr(true)

	b.Run("NoProvider", func(b *testing.B) {
		r := NewConfigResolver(startup, nil, nil, "bench", "peer")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sinkResolved = r.Get()
		}
	})
	b.Run("ProviderUnchanged", func(b *testing.B) {
		r := NewConfigResolver(startup, func() Override {
			return Override{MaxConnections: 9, DynamicScalingEnabled: enabled}
		}, nil, "bench", "peer")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sinkResolved = r.Get()
		}
	})
	b.Run("ProviderUnchangedParallel", func(b *testing.B) {
		r := NewConfigResolver(startup, func() Override {
			return Override{MaxConnections: 9, DynamicScalingEnabled: enabled}
		}, nil, "bench", "peer")
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var local Config
			for pb.Next() {
				local = r.Get()
			}
			_ = local
		})
	})
	b.Run("ProviderChangingEveryCall", func(b *testing.B) {
		n := 0
		r := NewConfigResolver(startup, func() Override {
			n++
			return Override{MaxConnections: 5 + n%2}
		}, nil, "bench", "peer")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sinkResolved = r.Get()
		}
	})
}
