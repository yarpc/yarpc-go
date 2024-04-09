package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCallEndLogs(t *testing.T) {
	/** Default values **/
	// Extractor produces at most one field.
	extractor := func(context.Context) zapcore.Field {
		return zap.String("extractor_key", "extractor_value")
	}

	callStartedAt := time.Now().Add(-time.Second)
	contextDeadline := time.Now().Add(time.Hour)
	contextWithDeadline, cancel := context.WithDeadline(context.Background(), contextDeadline)
	defer cancel()

	/** Tests **/

	dp := map[string]struct {
		/** Call **/
		callDirection directionName
		callCtx       context.Context

		/** Request result **/
		reqElapsed              time.Duration
		reqErr                  error
		reqIsApplicationError   bool
		reqApplicationErrorMeta *transport.ApplicationErrorMeta

		/** Expected log output **/
		expSuccessful bool
		expLogMessage string
		expLogType logLevelType
		expLogLevel   zapcore.Level
		expLogFields  []zap.Field // Except those added always
	}{
		"outbound_success": {
			callDirection: _directionOutbound,

			reqElapsed: 1 * time.Second,

			expSuccessful: true,
			expLogMessage: _successfulOutbound,
			expLogType: levelTypeSuccess,
			expLogLevel:   zapcore.DebugLevel,
		},

		"outbound_success_with_context_with_deadline": {
			callDirection: _directionOutbound,
			callCtx:       contextWithDeadline,

			reqElapsed: 1 * time.Second,

			expSuccessful: true,
			expLogMessage: _successfulOutbound,
			expLogType: levelTypeSuccess,
			expLogLevel:   zapcore.InfoLevel,
			expLogFields: []zap.Field{
				zap.Duration("timeout", contextDeadline.Sub(callStartedAt)),
			},
		},

		"inbound_success": {
			callDirection: _directionInbound,

			reqElapsed: 1 * time.Second,

			expSuccessful: true,
			expLogMessage: _successfulInbound,
			expLogType: levelTypeSuccess,
			expLogLevel:   zapcore.WarnLevel,
		},
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	for name, p := range dp {
		t.Run(name, func(t *testing.T) {
			if p.callCtx == nil {
				p.callCtx = context.Background()
			}

			call := call{
				edge:      &edge{logger: logger},
				extract:   extractor,
				started:   callStartedAt,
				ctx:       p.callCtx,
				rpcType:   transport.Unary,
				direction: p.callDirection,
				levels:    levelsFor(p.expLogType, p.expLogLevel),
			}

			entity, fields := call.endLogs(
				p.reqElapsed,
				p.reqErr,
				p.reqIsApplicationError,
				p.reqApplicationErrorMeta,
				[]zap.Field{
					zap.String("extra_key_1", "extra_value_1"),
					zap.String("extra_key_2", "extra_value_2"),
				}...,
			)

			require.NotNil(t, entity)

			alwaysExpFields := []zap.Field{
				zap.String("rpcType", transport.Unary.String()),
				zap.Duration("latency", p.reqElapsed),
				zap.Bool("successful", p.expSuccessful),
				zap.String("extractor_key", "extractor_value"),
				zap.String("extra_key_1", "extra_value_1"),
				zap.String("extra_key_2", "extra_value_2"),
			}

			assert.Equal(t, p.expLogMessage, entity.Message)
			assert.Equal(t, p.expLogLevel, entity.Level)
			assert.ElementsMatch(t, append(alwaysExpFields, p.expLogFields...), fields)
		})
	}
}

type logLevelType int

const (
	levelTypeSuccess logLevelType = iota+1
	levelTypeFailure
	levelTypeApplicationError
	levelTypeServerError
	levelTypeClientError
)

func levelsFor(levelType logLevelType, lvl zapcore.Level) *levels {
	res := &levels{}

	switch levelType {
	case levelTypeSuccess:
		res.success = lvl
	case levelTypeFailure:
		res.failure = lvl
	case levelTypeApplicationError:
		res.applicationError = lvl
	case levelTypeServerError:
		res.serverError = lvl
	case levelTypeClientError:
		res.clientError = lvl
	}

	return res
}
