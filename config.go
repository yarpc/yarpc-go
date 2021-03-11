// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package yarpc

import (
	"context"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"go.uber.org/net/metrics"
	"go.uber.org/net/metrics/tallypush"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Sleep between pushes to Tally metrics. At some point, we may want this
	// to be configurable.
	_tallyPushInterval = 500 * time.Millisecond
	_packageName       = "yarpc"
)

// DirectionalLogLevelConfig may override the log levels for any combination of
// successes, failures, and application errors.
type DirectionalLogLevelConfig struct {
	// Level at which successful requests are logged.
	// Defaults to DebugLevel.
	Success *zapcore.Level
	// Level at which errors are logged.
	// Thrift exceptions are application errors, which we log as a separate
	// class from success and failure.
	// Deprecated in favor of ServerError and ClientError.
	Failure *zapcore.Level
	// Level at which application errors are logged.
	// All Thrift exceptions are considered application errors.
	// All errors from Protobuf handlers are application errors.
	// Defaults to ErrorLevel.
	// Deprecated in favor of ServerError and ClientError.
	ApplicationError *zapcore.Level
	// Level at which client errors are logged.
	// All Thrift exceptions are considered application errors if
	// they are not annotated with the option rpc.code.
	// Defaults to ErrorLevel.
	ClientError *zapcore.Level
	// Level at which server errors are logged.
	// Defaults to ErrorLevel.
	ServerError *zapcore.Level
}

// LogLevelConfig configures the levels at which YARPC logs various things.
type LogLevelConfig struct {
	// Level at which successful requests are logged.
	// Defaults to DebugLevel.
	// Can be overridden by Inbound.Success or Outbound.Success for inbound or
	// outbound requests.
	Success *zapcore.Level
	// Level at which errors are logged.
	// Thrift exceptions are application errors, which we log as a separate
	// class from success and failure.
	// Can be overridden by Inbound.Failure or Outbound.Failure for inbound or
	// outbound requests.
	// Deprecated in favor of ServerError and ClientError.
	Failure *zapcore.Level
	// Level at which application errors are logged.
	// All Thrift exceptions are considered application errors.
	// Defaults to ErrorLevel.
	// Can be overridden by Inbound.ApplicationError or
	// Outbound.ApplicationError for inbound or outbound requests.
	// Deprecated in favor of ServerError and ClientError.
	ApplicationError *zapcore.Level
	// Level at which client errors are logged.
	// All Thrift exceptions are considered application errors if
	// they are not annotated with the option rpc.code.
	// Defaults to ErrorLevel.
	// Can be overridden by Inbound.ApplicationError or
	// Outbound.ApplicationError for inbound or outbound requests.
	ClientError *zapcore.Level
	// Level at which server errors are logged.
	// Defaults to ErrorLevel.
	// Can be overridden by Inbound.ApplicationError or
	// Outbound.ApplicationError for inbound or outbound requests.
	ServerError *zapcore.Level
	// Specific overrides for inbound and outbound requests.
	Inbound, Outbound DirectionalLogLevelConfig
}

// LoggingConfig describes how logging should be configured.
type LoggingConfig struct {
	// Supplies a logger for the dispatcher. By default, no logs are
	// emitted.
	Zap *zap.Logger

	// If supplied, ExtractContext is used to log request-scoped
	// information carried on the context (e.g., trace and span IDs).
	ContextExtractor func(context.Context) zapcore.Field

	// Levels configures the levels at which YARPC logs various messages.
	Levels LogLevelConfig
}

func (c LoggingConfig) logger(name string) *zap.Logger {
	if c.Zap == nil {
		return zap.NewNop()
	}
	return c.Zap.Named(_packageName).With(
		// Use a namespace to prevent key collisions with other libraries.
		zap.Namespace(_packageName),
		zap.String("dispatcher", name),
	)
}

func (c LoggingConfig) extractor() observability.ContextExtractor {
	if c.ContextExtractor == nil {
		return observability.NewNopContextExtractor()
	}
	return observability.ContextExtractor(c.ContextExtractor)
}

// MetricsConfig describes how telemetry should be configured.
// Scope and Tally are exclusive; choose one.
// If neither is present, metrics are not recorded, all instrumentation becomes
// no-ops.
// If both are present, we emit a warning and ignore Tally.
// If a metrics scope is preseent, we use that scope to record metrics and they
// are not pushed to Tally.
// If Tally is present, we use its metrics scope and push them periodically.
type MetricsConfig struct {
	// Metrics is a *"go.uber.org/net/metrics".Scope for recording stats.
	// YARPC does not push these metrics; pushing metrics from the root is an
	// external concern.
	Metrics *metrics.Scope
	// Tally scope used for pushing to M3 or StatsD-based systems. By
	// default, metrics are collected in memory but not pushed.
	// TODO deprecate this option for metrics configuration.
	Tally tally.Scope
	// TagsBlocklist enlists tags' keys that should be suppressed from all the metrics
	// emitted from w/in YARPC middleware.
	TagsBlocklist []string
}

func (c MetricsConfig) scope(name string, logger *zap.Logger) (*metrics.Scope, context.CancelFunc) {
	// Neither: no-op metrics, not pushed
	if c.Metrics == nil && c.Tally == nil {
		return nil, func() {}
	}

	// Both: ignore Tally and warn.
	if c.Metrics != nil && c.Tally != nil {
		logger.Warn("yarpc.NewDispatcher expects only one of Metrics.Tally or Metrics.Scope. " +
			"To push to Tally, either use a Metrics.Scope and use tallypush, or just pass a Tally Scope")
		c.Tally = nil
	}

	// Hereafter: We have one of either c.Metrics or c.Tally exclusively.

	var root *metrics.Root    // For pushing, if present
	var parent *metrics.Scope // For measuring

	if c.Metrics != nil {
		// root remains nil
		parent = c.Metrics
	} else { // c.Tally != nil
		root = metrics.New()
		parent = root.Scope()
	}

	meter := parent.Tagged(metrics.Tags{
		"component":  _packageName,
		"dispatcher": name,
	})

	// When we have c.Metrics, we do not push
	if root == nil {
		return meter, func() {}
	}

	// When we have c.Tally, we measure *and* push
	stopMeter, err := root.Push(tallypush.New(c.Tally), _tallyPushInterval)
	if err != nil {
		logger.Error("Failed to start pushing metrics to Tally.", zap.Error(err))
		return meter, func() {}
	}
	return meter, stopMeter
}

// Config specifies the parameters of a new Dispatcher constructed via
// NewDispatcher.
type Config struct {
	// Name of the service. This is the name used by other services when
	// making requests to this service.
	Name string

	// Inbounds define how this service receives incoming requests from other
	// services.
	//
	// This may be nil if this service does not receive any requests.
	Inbounds Inbounds

	// Outbounds defines how this service makes requests to other services.
	//
	// This may be nil if this service does not send any requests.
	Outbounds Outbounds

	// Inbound and Outbound Middleware that will be applied to all incoming
	// and outgoing requests respectively.
	//
	// These may be nil if there is no middleware to apply.
	InboundMiddleware  InboundMiddleware
	OutboundMiddleware OutboundMiddleware

	// Tracer is meant to add/record tracing information to a request.
	//
	// Deprecated: The dispatcher does nothing with this property.  Set the
	// tracer directly on the transports used to build inbounds and outbounds.
	Tracer opentracing.Tracer

	// RouterMiddleware is middleware to control how requests are routed.
	RouterMiddleware middleware.Router

	// Configures logging.
	Logging LoggingConfig

	// Configures telemetry.
	Metrics MetricsConfig

	// DisableAutoObservabilityMiddleware is used to stop the dispatcher from
	// automatically attaching observability middleware to all inbounds and
	// outbounds.  It is the assumption that if if this option is disabled the
	// observability middleware is being inserted in the Inbound/Outbound
	// Middleware.
	DisableAutoObservabilityMiddleware bool
}
