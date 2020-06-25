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

package thrift_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceserver"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	_clientName = "caller"
	_serverName = "callee"

	_wantSuccess              = "success"
	_wantExceptionWithCode    = "exception with code"
	_wantExceptionWithoutCode = "exception with no code"
)

func TestThriftExceptionObservability(t *testing.T) {
	// TODO(apeatsbond): add HTTP test when feature complete.

	t.Run("exception with annotation", func(t *testing.T) {
		client, cleanup := initClientAndServer(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := client.Call(ctx, _wantExceptionWithCode)
		require.Error(t, err, "expected call error")

		ex, ok := err.(*test.ExceptionWithCode)
		require.True(t, ok, "unexpected Thrift exception")
		assert.Equal(t, _wantExceptionWithCode, ex.Val, "unexpected response")

	})

	t.Run("exception without annotation ", func(t *testing.T) {
		client, cleanup := initClientAndServer(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := client.Call(ctx, _wantExceptionWithoutCode)
		require.Error(t, err, "expected call error")

		ex, ok := err.(*test.ExceptionWithoutCode)
		require.True(t, ok, "unexpected Thrift exception")
		assert.Equal(t, _wantExceptionWithoutCode, ex.Val, "unexpected response")

	})
}

func initClientAndServer(
	t *testing.T,
) (
	client testserviceclient.Interface,
	cleanup func(),
) {
	loggerCore, observedLogs := observer.New(zapcore.DebugLevel)
	metricsRoot := metrics.New()

	serverAddr, cleanupServer := newServer(t, loggerCore, metricsRoot)
	client, cleanupClient := newClient(t, serverAddr, loggerCore, metricsRoot)

	_ = observedLogs.TakeAll() // ignore all start up logs

	return client, func() {
		cleanupServer()
		cleanupClient()
	}
}

func newServer(t *testing.T, loggerCore zapcore.Core, metricsRoot *metrics.Root) (addr string, cleanup func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	trans, err := tchannel.NewTransport(
		tchannel.ServiceName(_serverName),
		tchannel.Listener(listener))
	require.NoError(t, err)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{trans.NewInbound()},
		Logging: yarpc.LoggingConfig{
			Zap: zap.New(loggerCore),
		},
		Metrics: yarpc.MetricsConfig{
			Metrics: metricsRoot.Scope(),
		},
	})

	dispatcher.Register(testserviceserver.New(&testServer{}))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr = listener.Addr().String()
	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func newClient(t *testing.T, serverAddr string, loggerCore zapcore.Core, metricsRoot *metrics.Root) (client testserviceclient.Interface, cleanup func()) {
	trans, err := tchannel.NewTransport(tchannel.ServiceName(_clientName))
	require.NoError(t, err)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       trans.NewSingleOutbound(serverAddr),
			},
		},
		Logging: yarpc.LoggingConfig{
			Zap: zap.New(loggerCore),
		},
		Metrics: yarpc.MetricsConfig{
			Metrics: metricsRoot.Scope(),
		},
	})

	client = testserviceclient.New(dispatcher.ClientConfig(_serverName))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return client, cleanup
}

type testServer struct{}

func (testServer) Call(ctx context.Context, val string) (string, error) {
	switch val {
	case _wantExceptionWithoutCode:
		return "", &test.ExceptionWithoutCode{Val: val}
	case _wantExceptionWithCode:
		return "", &test.ExceptionWithCode{Val: val}
	default: // success
		return val, nil
	}
}
