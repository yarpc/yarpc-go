// isolating panic tests in separate package to avoid cyclic imports
package transport_panic_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
)

func TestDispatchUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *transport.Request, transport.ResponseWriter) error {
		panic(msg)
	}

	err := transport.DispatchUnaryHandler(
		context.Background(),
		transport.UnaryHandlerFunc(handler),
		time.Now(),
		&transport.Request{},
		nil,
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestDispatchOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *transport.Request) error {
		panic(msg)
	}

	err := transport.DispatchOnewayHandler(
		context.Background(),
		transport.OnewayHandlerFunc(handler),
		&transport.Request{},
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestDispatchStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"

	handler := func(_ *transport.ServerStream) error {
		panic(msg)
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{},
		}).Times(1)
	mockServerStream, _ := transport.NewServerStream(mockStream)
	err := transport.DispatchStreamHandler(
		transport.StreamHandlerFunc(handler),
		mockServerStream,
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}
