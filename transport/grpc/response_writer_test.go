package grpc

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc/metadata"
)

func Test_AddHeader(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		value  string
		start  metadata.MD
		result metadata.MD
		err    bool
	}{
		{
			name:   "empty key",
			key:    "foo",
			result: metadata.MD{},
		},
		{
			name:   "lowercase",
			key:    "foo",
			value:  "bar",
			result: metadata.New(map[string]string{"foo": "bar"}),
		},
		{
			name:   "titlecase",
			key:    "Foo",
			value:  "Bar",
			result: metadata.New(map[string]string{"foo": "Bar"}),
		},
		{
			name:   "restricted key",
			key:    "rpc-foo",
			result: metadata.MD{},
			err:    true,
		},
		{
			name:  "duplicate key",
			key:   "foo",
			value: "bar",
			start: metadata.New(map[string]string{"foo": "bar"}),
			err:   true,
		},
	}

	t.Run("AddHeader", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				rw := newResponseWriter()
				rw.md = test.start
				rw.AddHeader(test.key, test.value)
				if test.err {
					assert.Error(t, rw.headerErr)
				} else {
					assert.NoError(t, rw.headerErr)
					assert.Equal(t, test.result, rw.md)
				}
			})
		}
	})

	t.Run("AddHeaders", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				rw := newResponseWriter()
				rw.md = test.start
				rw.AddHeaders(transport.HeadersFromMap(map[string]string{test.key: test.value}))
				if test.err {
					assert.Error(t, rw.headerErr)
				} else {
					assert.NoError(t, rw.headerErr)
					assert.Equal(t, test.result, rw.md)
				}
			})
		}
	})
}

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
