// Copyright (c) 2026 Uber Technologies, Inc.
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
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func loggedStringFields(e observer.LoggedEntry) map[string]string {
	m := make(map[string]string)
	for _, f := range e.Context {
		if f.Type == zapcore.StringType {
			m[f.Key] = f.String
		}
	}
	return m
}

func TestRegister_thriftUnannotatedExceptionWarnLogs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	const procName = "mysvc::ping"
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	require.Nil(t, d.meter)

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	assert.NotPanics(t, func() {
		d.Register([]transport.Procedure{{
			Name:        procName,
			Encoding:    transport.ThriftEncoding,
			HandlerSpec: transport.NewUnaryHandlerSpec(h),
			Exceptions: map[string]string{
				"error.InvalidArgumentError": transport.RPCCodeNotSetLiteral,
			},
		}})
	})

	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 1, w.Len())
	e := w.All()[0]
	fields := loggedStringFields(e)
	assert.Equal(t, procName, fields["procedure"])
	assert.Equal(t, "error.InvalidArgumentError", fields["exception"])
}

func TestRegister_thriftUnannotatedExceptionWithMetricsScope(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	const procName = "mysvc::ping"
	d := NewDispatcher(Config{
		Name: "mysvc",
		Metrics: MetricsConfig{
			Metrics: metrics.New().Scope(),
		},
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	require.NotNil(t, d.meter)

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	assert.NotPanics(t, func() {
		d.Register([]transport.Procedure{{
			Name:        procName,
			Encoding:    transport.ThriftEncoding,
			HandlerSpec: transport.NewUnaryHandlerSpec(h),
			Exceptions: map[string]string{
				"error.InvalidArgumentError": transport.RPCCodeNotSetLiteral,
			},
		}})
	})

	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 1, w.Len())
	e := w.All()[0]
	fields := loggedStringFields(e)
	assert.Equal(t, procName, fields["procedure"])
	assert.Equal(t, "error.InvalidArgumentError", fields["exception"])
}

func TestRegister_thriftRpcCodeAnnotationSkipsUnannotatedWarn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"SomeErr": "INVALID_ARGUMENT",
		},
	}})
	assert.Equal(t, 0, logs.Len())
}

func TestRegister_nonThriftEncodingSkipsExceptionInstrumentation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    "json",
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"SomeErr": transport.RPCCodeNotSetLiteral,
		},
	}})
	assert.Equal(t, 0, logs.Len())
}

func TestRegister_thriftNilExceptionsNoWarn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
	}})
	assert.Equal(t, 0, logs.Len())
}

// recordingLifecycle is a transport.Lifecycle that records the order in which
// its Stop method is invoked, so tests can assert teardown ordering.
type recordingLifecycle struct {
	name      string
	stopErr   error
	mu        *sync.Mutex
	stopOrder *[]string
}

func (l *recordingLifecycle) Start() error { return nil }

func (l *recordingLifecycle) Stop() error {
	l.mu.Lock()
	*l.stopOrder = append(*l.stopOrder, l.name)
	l.mu.Unlock()
	return l.stopErr
}

func (l *recordingLifecycle) IsRunning() bool { return false }

// TestPhasedStarterAbortStopsInReverseOrder verifies that a failed startup
// tears down the already-started lifecycles in the reverse of start order
// (inbounds, then outbounds, then transports). This guarantees that an
// outbound is always stopped before the transport it depends on, preventing a
// data race between an outbound's Stop and its transport's Stop.
func TestPhasedStarterAbortStopsInReverseOrder(t *testing.T) {
	var (
		mu        sync.Mutex
		stopOrder []string
	)
	newLifecycle := func(name string) *recordingLifecycle {
		return &recordingLifecycle{name: name, mu: &mu, stopOrder: &stopOrder}
	}

	s := &PhasedStarter{
		log: zap.NewNop(),
		// started reflects start order: transports, then outbounds, then
		// inbounds.
		started: []transport.Lifecycle{
			newLifecycle("transport"),
			newLifecycle("outbound"),
			newLifecycle("inbound"),
		},
	}

	err := s.abort(nil)
	require.NoError(t, err, "abort should not report an error when all Stops succeed")
	assert.Equal(t, []string{"inbound", "outbound", "transport"}, stopOrder,
		"lifecycles must be stopped in reverse of start order")
}

// TestPhasedStarterAbortAggregatesErrors verifies that abort still tears down
// every started lifecycle even if some Stops fail, and that it combines the
// original startup error with any Stop errors.
func TestPhasedStarterAbortAggregatesErrors(t *testing.T) {
	var (
		mu        sync.Mutex
		stopOrder []string
	)
	newLifecycle := func(name string, stopErr error) *recordingLifecycle {
		return &recordingLifecycle{name: name, stopErr: stopErr, mu: &mu, stopOrder: &stopOrder}
	}

	s := &PhasedStarter{
		log: zap.NewNop(),
		started: []transport.Lifecycle{
			newLifecycle("transport", errors.New("transport stop failed")),
			newLifecycle("outbound", nil),
			newLifecycle("inbound", errors.New("inbound stop failed")),
		},
	}

	err := s.abort([]error{errors.New("startup failed")})
	require.Error(t, err)
	assert.ErrorContains(t, err, "startup failed", "should retain the original startup error")
	assert.ErrorContains(t, err, "inbound stop failed", "should include inbound Stop error")
	assert.ErrorContains(t, err, "transport stop failed", "should include transport Stop error")
	assert.Equal(t, []string{"inbound", "outbound", "transport"}, stopOrder,
		"all lifecycles must be stopped even when some Stops fail")
}

func TestRegister_thriftMultipleUnannotatedExceptionsWarnEach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"AErr": transport.RPCCodeNotSetLiteral,
			"BErr": transport.RPCCodeNotSetLiteral,
		},
	}})
	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 2, w.Len())
	got := make(map[string]struct{})
	for _, e := range w.All() {
		got[loggedStringFields(e)["exception"]] = struct{}{}
	}
	assert.Contains(t, got, "AErr")
	assert.Contains(t, got, "BErr")
}

func TestMetricsConfigReachesObservabilityMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	root := metrics.New()
	cfg := addObservingMiddleware(
		Config{
			Name: "mysvc",
			Metrics: MetricsConfig{
				TagsBlocklist:   []string{"routing_delegate"},
				DisabledMetrics: []string{"ttl_ms", "request_payload_size_bytes"},
				MaxEdges:        1,
			},
		},
		root.Scope(), zap.NewNop(), observability.NewNopContextExtractor())

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	h.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, rd := range []string{"rd1", "rd2"} {
		err := cfg.InboundMiddleware.Unary.Handle(ctx, &transport.Request{
			Caller:          "caller",
			Service:         "mysvc",
			Procedure:       "p",
			Encoding:        "raw",
			RoutingDelegate: rd,
			Body:            strings.NewReader("body"),
		}, &transporttest.FakeResponseWriter{}, h)
		require.NoError(t, err)
	}

	var histograms []string
	for _, hist := range root.Snapshot().Histograms {
		histograms = append(histograms, hist.Name)
	}
	assert.Contains(t, histograms, "success_latency_ms")
	assert.Contains(t, histograms, "response_payload_size_bytes")
	assert.NotContains(t, histograms, "ttl_ms", "disabled in the config")
	assert.NotContains(t, histograms, "request_payload_size_bytes", "disabled in the config")

	// Both calls share one edge because of the blocklist.
	counters := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		counters[c.Name] += c.Value
		assert.Equal(t, "__dropped__", c.Tags["routing_delegate"])
	}
	assert.Equal(t, map[string]int64{"calls": 2, "successes": 2, "panics": 0}, counters)

	// The limit of one edge is full, so a different edge is recorded on the
	// overflow edge.
	require.NoError(t, cfg.InboundMiddleware.Unary.Handle(ctx, &transport.Request{
		Caller:    "other-caller",
		Service:   "mysvc",
		Procedure: "other",
		Encoding:  "raw",
		Body:      strings.NewReader("body"),
	}, &transporttest.FakeResponseWriter{}, h))
	overflowed := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		if c.Tags["source"] == "__overflow__" {
			overflowed[c.Name] += c.Value
		}
	}
	assert.Equal(t, map[string]int64{"calls": 1, "successes": 1, "panics": 0}, overflowed)
}
