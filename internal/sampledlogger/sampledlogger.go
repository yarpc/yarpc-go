package sampledlogger

import (
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SampledLogger struct {
	logger      *zap.Logger
	logInterval time.Duration
	lastLogTime atomic.Value
}

// NewSampledLogger creates a new SampledLogger with custom interval and zap.Logger.
func NewSampledLogger(interval time.Duration, logger *zap.Logger) *SampledLogger {
	return &SampledLogger{
		logger:      logger,
		logInterval: interval,
	}
}

// NewDefaultSampledLogger creates a SampledLogger with zap.NewNop() and 5-minute interval.
func NewDefaultSampledLogger() *SampledLogger {
	return NewSampledLogger(5*time.Minute, zap.NewNop())
}

// log performs rate-limited logging for the given level and message.
func (sl *SampledLogger) log(level zapcore.Level, msg string, fields ...zap.Field) {
	now := time.Now()
	last := sl.lastLogTime.Load().(time.Time)

	if last.IsZero() || now.Sub(last) > sl.logInterval {
		switch level {
		case zapcore.DebugLevel:
			sl.logger.Debug(msg, fields...)
		case zapcore.InfoLevel:
			sl.logger.Info(msg, fields...)
		case zapcore.WarnLevel:
			sl.logger.Warn(msg, fields...)
		case zapcore.ErrorLevel:
			sl.logger.Error(msg, fields...)
		case zapcore.DPanicLevel:
			sl.logger.DPanic(msg, fields...)
		case zapcore.PanicLevel:
			sl.logger.Panic(msg, fields...)
		case zapcore.FatalLevel:
			sl.logger.Fatal(msg, fields...)
		}
		sl.lastLogTime.Store(now)
	}
}

// Debug logs a debug-level message with rate limiting.
func (sl *SampledLogger) Debug(msg string, fields ...zap.Field) {
	sl.log(zapcore.DebugLevel, msg, fields...)
}

// Info logs an info-level message with rate limiting.
func (sl *SampledLogger) Info(msg string, fields ...zap.Field) {
	sl.log(zapcore.InfoLevel, msg, fields...)
}

// Warn logs a warn-level message with rate limiting.
func (sl *SampledLogger) Warn(msg string, fields ...zap.Field) {
	sl.log(zapcore.WarnLevel, msg, fields...)
}

// Error logs an error-level message with rate limiting.
func (sl *SampledLogger) Error(msg string, fields ...zap.Field) {
	sl.log(zapcore.ErrorLevel, msg, fields...)
}

// DPanic logs a DPanic-level message with rate limiting.
func (sl *SampledLogger) DPanic(msg string, fields ...zap.Field) {
	sl.log(zapcore.DPanicLevel, msg, fields...)
}

// Panic logs a panic-level message with rate limiting.
func (sl *SampledLogger) Panic(msg string, fields ...zap.Field) {
	sl.log(zapcore.PanicLevel, msg, fields...)
}

// Fatal logs a fatal-level message with rate limiting.
func (sl *SampledLogger) Fatal(msg string, fields ...zap.Field) {
	sl.log(zapcore.FatalLevel, msg, fields...)
}
