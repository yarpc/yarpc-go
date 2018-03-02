package relay

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

func TestScopeOption(t *testing.T) {
	scope := tally.NoopScope
	option := Scope(scope)
	opts := applyOptions(option)
	assert.Equal(t, scope, opts.scope)
}

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
