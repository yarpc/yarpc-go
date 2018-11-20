package internaldrainmiddleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/v2"
)

type nop struct{}

func (nop) Handle(_ context.Context, _ *yarpc.Request, _ *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	return nil, nil, nil
}

func TestMiddleware(t *testing.T) {
	mw := New()

	_, _, err := mw.Handle(context.Background(), &yarpc.Request{}, &yarpc.Buffer{}, nop{})
	assert.NoError(t, err, "unary error")
	assert.NoError(t, mw.Drain(), "error on first drain")
	assert.Error(t, mw.Drain(), "success on second drain")
}
