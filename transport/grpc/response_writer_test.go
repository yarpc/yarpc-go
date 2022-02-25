package grpc

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

func Benchmark_ResponseWriter_AddHeaders(b *testing.B) {
	b.Run("lowercase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"foo"+strconv.Itoa(i), "bar",
			))
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("lowercase no-value", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"foo", "",
			))
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("titlecase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"Foo"+strconv.Itoa(i), "bar",
			))
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("titlecase no-value", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
				"Foo", "",
			))
		}
		assert.NoError(b, rw.headerErr)
	})
}

func Test_addHeaders(t *testing.T) {
	// t.Run("No ")
	rw := newResponseWriter()
	for i := 0; i < 10; i++ {
		rw.AddHeaders(transport.NewHeadersWithCapacity(1).With(
			"foo", "bar",
		))
	}
	assert.NoError(t, rw.headerErr)

}

func Benchmark_ResponseWriter_AddHeader(b *testing.B) {
	b.Run("lowercase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("foo"+strconv.Itoa(i), "bar")
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("lowercase no-value", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("foo", "")
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("titlecase", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("Foo"+strconv.Itoa(i), "bar")
		}
		assert.NoError(b, rw.headerErr)
	})

	b.Run("titlecase no-value", func(b *testing.B) {
		rw := newResponseWriter()
		for i := 0; i < b.N; i++ {
			rw.AddHeader("Foo", "")
		}
		assert.NoError(b, rw.headerErr)
	})
}
