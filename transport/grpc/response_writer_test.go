package grpc

import (
	"testing"

	"go.uber.org/yarpc/api/transport"
)

func Benchmark_ResponseWriter_AddHeaders(b *testing.B) {
	b.Run("lowercase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"abc", "",
			))
		}
	})

	b.Run("titlecase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"Abc", "",
			))
		}
	})
}

func Benchmark_ResponseWriter_AddHeader(b *testing.B) {
	b.Run("lowercase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("abc", "")
		}
	})

	b.Run("titlecase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("Abc", "")
		}
	})
}
