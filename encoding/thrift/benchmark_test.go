package thrift

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/golang/mock/gomock"
	"go.uber.org/thriftrw/wire"
)

func BenchmarkThrift(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	proto := NewMockProtocol(mockCtrl)
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(wire.Envelope{
		Name:  "someMethod",
		SeqID: 42,
		Type:  wire.Exception,
		Value: wire.NewValueStruct(wire.Struct{}),
	}, nil).AnyTimes()

	handler := func(ctx context.Context, w wire.Value) (Response,
		error) {
		return Response{
			Body: fakeEnveloper(wire.Call),
		}, nil
	}
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler, Enveloping: true}

	for i := 0; i < b.N; i++ {
		rw := new(transporttest.FakeResponseWriter)
		h.Handle(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  Encoding,
			Procedure: "MyService::someMethod",
			Body:      bytes.NewReader([]byte("irrelevant")),
		}, rw)
	}
}
