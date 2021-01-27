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

package protobuf_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/transport/grpc"
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

func initClientAndServer(t *testing.T, server testpb.TestYARPCServer) (
	client testpb.TestYARPCClient,
	observedLogs *observer.ObservedLogs,
	clientMetricsRoot *metrics.Root,
	serverMetricsRoot *metrics.Root,
	cleanup func(),
) {
	loggerCore, observedLogs := observer.New(zapcore.DebugLevel)
	clientMetricsRoot, serverMetricsRoot = metrics.New(), metrics.New()

	serverAddr, cleanupServer := newServer(t, loggerCore, serverMetricsRoot, server)
	client, cleanupClient := newClient(t, serverAddr, loggerCore, clientMetricsRoot)

	_ = observedLogs.TakeAll() // ignore all start up logs

	return client, observedLogs, clientMetricsRoot, serverMetricsRoot, func() {
		cleanupServer()
		cleanupClient()
	}
}

func newServer(t *testing.T, loggerCore zapcore.Core, metricsRoot *metrics.Root, server testpb.TestYARPCServer) (addr string, cleanup func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{Zap: zap.New(loggerCore)},
		Metrics:  yarpc.MetricsConfig{Metrics: metricsRoot.Scope()},
	})

	dispatcher.Register(testpb.BuildTestYARPCProcedures(server))
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
