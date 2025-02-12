// Copyright (c) 2025 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/internal/testutils"
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
		wantCounters := []testutils.CounterAssertion{
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

		testutils.AssertCounters(t, wantCounters, clientMetricsRoot.Snapshot().Counters)
		testutils.AssertCounters(t, wantCounters, serverMetricsRoot.Snapshot().Counters)
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
		wantCounters := []testutils.CounterAssertion{
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "successes", Value: 1},
		}

		testutils.AssertCounters(t, wantCounters, clientMetricsRoot.Snapshot().Counters)
		testutils.AssertCounters(t, wantCounters, serverMetricsRoot.Snapshot().Counters)
	})
	t.Run("inbound histograms", func(t *testing.T) {
		wantHistograms := []testutils.HistogramAssertion{
			{Name: "caller_failure_latency_ms"},
			{Name: "request_payload_size_bytes", Value: []int64{16}},
			{Name: "response_payload_size_bytes", Value: []int64{16}},
			{Name: "server_failure_latency_ms"},
			{Name: "success_latency_ms", IgnoreValueCompare: true, ValueLength: 1},
			{Name: "timeout_ttl_ms"},
			{Name: "ttl_ms", Value: []int64{1000}},
		}
		testutils.AssertClientAndServerHistograms(t, wantHistograms, clientMetricsRoot, serverMetricsRoot)
	})
}

func TestProtobufStreamMetrics(t *testing.T) {
	client, _, clientMetricsRoot, serverMetricsRoot, cleanup := initClientAndServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	stream, err := client.Duplex(ctx)
	require.NoError(t, err, "unexpected error on stream creation")

	msg := &testpb.TestMessage{Value: "echo"}

	err = stream.Send(msg)
	require.NoError(t, err, "unexpected error on stream send")

	reply, err := stream.Recv()
	require.NoError(t, err, "unexpected error on stream receive")
	assert.Equal(t, msg, reply)

	stream.CloseSend()

	_, err = stream.Recv()
	assert.Error(t, err)

	t.Run("counters", func(t *testing.T) {
		clientCounters := []testutils.CounterAssertion{
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "stream_receive_failures", Value: 1},
			{Name: "stream_receive_successes", Value: 1},
			{Name: "stream_receives", Value: 2},
			{Name: "stream_send_successes", Value: 1},
			{Name: "stream_sends", Value: 1},
			{Name: "successes", Value: 1},
		}

		serverCounters := []testutils.CounterAssertion{
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "server_failures", Value: 1},
			{Name: "stream_receive_successes", Value: 2},
			{Name: "stream_receives", Value: 2},
			{Name: "stream_send_successes", Value: 1},
			{Name: "stream_sends", Value: 1},
			{Name: "successes", Value: 1},
		}

		testutils.AssertCounters(t, clientCounters, clientMetricsRoot.Snapshot().Counters)
		testutils.AssertCounters(t, serverCounters, serverMetricsRoot.Snapshot().Counters)
	})
	t.Run("inbound histograms", func(t *testing.T) {
		wantHistograms := []testutils.HistogramAssertion{
			{Name: "stream_duration_ms", IgnoreValueCompare: true, ValueLength: 1},
			{Name: "stream_request_payload_size_bytes", Value: []int64{8}},
			{Name: "stream_response_payload_size_bytes", Value: []int64{8}},
		}
		testutils.AssertHistograms(t, wantHistograms, serverMetricsRoot.Snapshot().Histograms)
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
	outbound := grpc.NewTransport().NewSingleOutbound(serverAddr)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       outbound,
				Stream:      outbound,
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

func (observabilityTestServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		err = stream.Send(msg)
		if err != nil {
			return err
		}
	}
}
