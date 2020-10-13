// Copyright (c) 2020 Uber Technologies, Inc.
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

package protobuf_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	_clientName = "caller"
	_serverName = "callee"

	// from observability middleware
	_errorInbound  = "Error handling inbound request."
	_errorOutbound = "Error making outbound call."
)

func TestProtobufErrorDetailObservability(t *testing.T) {
	client, observedLogs, clientMetricsRoot, serverMetricsRoot, cleanup := initClientAndServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.Unary(ctx, &testpb.TestMessage{})
	require.Error(t, err, "expected call error")

	require.NotEmpty(t, protobuf.GetErrorDetails(err),
		"no error details, found error of type '%T': %v", err, err)

	t.Run("logs", func(t *testing.T) {
		wantFields := []zapcore.Field{
			zap.String("errorCode", "invalid-argument"),
			zap.String("errorName", "StringValue"),
			zap.String("errorDetails", "[]{ StringValue{value:\"string value\" } , Int32Value{value:100 } }"),
		}
		assertLogs(t, wantFields, observedLogs.TakeAll())
	})

	t.Run("metrics", func(t *testing.T) {
		wantCounters := []counterAssertion{
			{
				Name: "caller_failures",
				Tags: map[string]string{
					"error":      "invalid-argument",
					"error_name": "StringValue",
				},
				Value: 1,
			},
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "successes"},
		}

		assertClientAndServerMetrics(t, wantCounters, clientMetricsRoot, serverMetricsRoot)
	})
}

func TestProtobufMetrics(t *testing.T) {
	client, _, clientMetricsRoot, serverMetricsRoot, cleanup := initClientAndServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.Unary(ctx, &testpb.TestMessage{Value: "success"})
	require.NoError(t, err, "unexpected call error")

	t.Run("counters", func(t *testing.T) {
		wantCounters := []counterAssertion{
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "successes", Value: 1},
		}

		assertClientAndServerMetrics(t, wantCounters, clientMetricsRoot, serverMetricsRoot)
	})
	t.Run("inbound histograms", func(t *testing.T) {
		wantHistograms := []histogramAssertion{
			{Name: "caller_failure_latency_ms"},
			{Name: "request_payload_size_bytes", Value: []int64{16}},
			{Name: "response_payload_size_bytes", Value: []int64{16}},
			{Name: "server_failure_latency_ms"},
			{Name: "success_latency_ms", Value: []int64{1}},
			{Name: "timeout_ttl_ms"},
			{Name: "ttl_ms", Value: []int64{1000}},
		}
		assertHistogram(t, wantHistograms, serverMetricsRoot.Snapshot().Histograms)
	})
}

func assertLogs(t *testing.T, wantFields []zapcore.Field, logs []observer.LoggedEntry) {
	require.Len(t, logs, 2, "unexpected number of logs")

	t.Run("inbound", func(t *testing.T) {
		require.Equal(t, _errorInbound, logs[0].Message, "unexpected log")
		assertLogFields(t, wantFields, logs[0].Context)
	})

	t.Run("outbound", func(t *testing.T) {
		require.Equal(t, _errorOutbound, logs[1].Message, "unexpected log")
		assertLogFields(t, wantFields, logs[1].Context)
	})
}

func assertLogFields(t *testing.T, wantFields, gotContext []zapcore.Field) {
	gotFields := make(map[string]zapcore.Field)
	for _, log := range gotContext {
		gotFields[log.Key] = log
	}

	for _, want := range wantFields {
		got, ok := gotFields[want.Key]
		if assert.True(t, ok, "key %q not found", want.Key) {
			assert.Equal(t, want, got, "unexpected log field")
		}
	}
}

type counterAssertion struct {
	Name  string
	Tags  map[string]string
	Value int
}

type histogramAssertion struct {
	Name  string
	Tags  map[string]string
	Value []int64
}

func assertClientAndServerMetrics(t *testing.T, counterAssertions []counterAssertion, clientSnapshot, serverSnapshot *metrics.Root) {
	t.Run("inbound", func(t *testing.T) {
		assertMetrics(t, counterAssertions, serverSnapshot.Snapshot().Counters)
	})
	t.Run("outbound", func(t *testing.T) {
		assertMetrics(t, counterAssertions, clientSnapshot.Snapshot().Counters)
	})
}

func assertMetrics(t *testing.T, counterAssertions []counterAssertion, snapshot []metrics.Snapshot) {
	require.Len(t, counterAssertions, len(snapshot), "unexpected number of counters")

	for i, wantCounter := range counterAssertions {
		require.Equal(t, wantCounter.Name, snapshot[i].Name, "unexpected counter")
		assert.EqualValues(t, wantCounter.Value, snapshot[i].Value, "unexpected counter value")
		for wantTagKey, wantTagVal := range wantCounter.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

func assertHistogram(t *testing.T, histogramAssertions []histogramAssertion, snapshot []metrics.HistogramSnapshot) {
	require.Len(t, histogramAssertions, len(snapshot), "unexpected number of histograms")

	for i, wantCounter := range histogramAssertions {
		require.Equal(t, wantCounter.Name, snapshot[i].Name, "unexpected histogram")
		assert.EqualValues(t, wantCounter.Value, snapshot[i].Values, "unexpected histogram value")
		for wantTagKey, wantTagVal := range wantCounter.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

func initClientAndServer(t *testing.T) (
	client testpb.TestYARPCClient,
	observedLogs *observer.ObservedLogs,
	clientMetricsRoot *metrics.Root,
	serverMetricsRoot *metrics.Root,
	cleanup func(),
) {
	loggerCore, observedLogs := observer.New(zapcore.DebugLevel)
	clientMetricsRoot, serverMetricsRoot = metrics.New(), metrics.New()

	serverAddr, cleanupServer := newServer(t, loggerCore, serverMetricsRoot)
	client, cleanupClient := newClient(t, serverAddr, loggerCore, clientMetricsRoot)

	_ = observedLogs.TakeAll() // ignore all start up logs

	return client, observedLogs, clientMetricsRoot, serverMetricsRoot, func() {
		cleanupServer()
		cleanupClient()
	}
}

func newServer(t *testing.T, loggerCore zapcore.Core, metricsRoot *metrics.Root) (addr string, cleanup func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{Zap: zap.New(loggerCore)},
		Metrics:  yarpc.MetricsConfig{Metrics: metricsRoot.Scope()},
	})

	dispatcher.Register(testpb.BuildTestYARPCProcedures(&observabilityTestServer{}))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr = inbound.Addr().String()
	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func newClient(t *testing.T, serverAddr string, loggerCore zapcore.Core, metricsRoot *metrics.Root) (client testpb.TestYARPCClient, cleanup func()) {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       grpc.NewTransport().NewSingleOutbound(serverAddr),
			},
		},
		Logging: yarpc.LoggingConfig{Zap: zap.New(loggerCore)},
		Metrics: yarpc.MetricsConfig{Metrics: metricsRoot.Scope()},
	})

	client = testpb.NewTestYARPCClient(dispatcher.ClientConfig(_serverName))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return client, cleanup
}

type observabilityTestServer struct{}

func (observabilityTestServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	if msg.Value == "success" {
		return &testpb.TestMessage{Value: msg.Value}, nil
	}
	details := []proto.Message{
		&types.StringValue{Value: "string value"},
		&types.Int32Value{Value: 100},
	}
	return nil, protobuf.NewError(yarpcerrors.CodeInvalidArgument, "my message", protobuf.WithErrorDetails(details...))
}

func (observabilityTestServer) Duplex(testpb.TestServiceDuplexYARPCServer) error { return nil }
