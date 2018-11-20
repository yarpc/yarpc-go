package internaldrainmiddleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

type nop struct{}

func (nop) Handle(_ context.Context, _ *transport.Request, _ transport.ResponseWriter) error {
	return nil
}

func (nop) HandleOneway(_ context.Context, _ *transport.Request) error {
	return nil
}

func TestMiddleware(t *testing.T) {
	mw := New()
	assert.NoError(t, mw.Handle(context.Background(), nil, nil, nop{}), "unary error")
	assert.NoError(t, mw.HandleOneway(context.Background(), nil, nop{}), "oneway error")
	assert.NoError(t, mw.Drain(), "error on first drain")
	assert.Error(t, mw.Drain(), "success on second drain")
}
