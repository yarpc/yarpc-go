package observabilitylogger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFromContext_NoLogger(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, FromContext(ctx))
}
func TestWithLogger_RoundTrip(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := WithLogger(context.Background(), logger)
	got := FromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, logger, got)
}
func TestFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), loggerKey{}, "not a logger")
	assert.Nil(t, FromContext(ctx))
}
