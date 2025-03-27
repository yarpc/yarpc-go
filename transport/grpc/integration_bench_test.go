package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/opentracing/opentracing-go"
)

func BenchmarkYARPCBasic(b *testing.B) {
	te := testEnvOptions{
		TransportOptions: []TransportOption{
			Tracer(opentracing.NoopTracer{}),
		},
	}
	te.do(b, func(t testing.TB, e *testEnv) {
		b := t.(*testing.B)
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", "bar"))
			value, err := e.GetValueYARPC(context.Background(), "foo")
			assert.NoError(t, err)
			assert.Equal(t, "bar", value)
		}
	})
}