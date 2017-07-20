package debug

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerOption(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	option := Logger(logger)
	opts := applyOptions(option)
	assert.Equal(t, logger, opts.logger)
}

func TestNilLoggerOption(t *testing.T) {
	opts := applyOptions()
	assert.NotNil(t, opts.logger)
}
