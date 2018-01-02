// Copyright (c) 2018 Uber Technologies, Inc.
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Sleep between pushes to Tally metrics. At some point, we may want this
	// to be configurable.
	_tallyPushInterval = 500 * time.Millisecond
	_packageName       = "yarpc"
)

// LoggingConfig describes how logging should be configured.
type LoggingConfig struct {
	// Supplies a logger for the dispatcher. By default, no logs are
	// emitted.
	Zap *zap.Logger
	// If supplied, ExtractContext is used to log request-scoped
	// information carried on the context (e.g., trace and span IDs).
	ContextExtractor func(context.Context) zapcore.Field
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
type MetricsConfig struct {
	// Tally scope used for pushing to M3 or StatsD-based systems. By
	// default, metrics are collected in memory but not pushed.
	Tally tally.Scope
}

func (c MetricsConfig) registry(name string, logger *zap.Logger) (*pally.Registry, context.CancelFunc) {
	r := pally.NewRegistry(
		pally.Labeled(pally.Labels{
			"component":  _packageName,
			"dispatcher": pally.ScrubLabelValue(name),
		}),
		// Also expose all YARPC metrics via the default Prometheus registry.
		pally.Federated(prometheus.DefaultRegisterer),
	)

	if c.Tally == nil {
		return r, func() {}
	}

	stop, err := r.Push(c.Tally, _tallyPushInterval)
	if err != nil {
		logger.Error("Failed to start pushing metrics to Tally.", zap.Error(err))
		return r, func() {}
	}
	return r, stop
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
}
