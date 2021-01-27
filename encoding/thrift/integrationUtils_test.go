package thrift_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceserver"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func initClientAndServer(
	t *testing.T,
	trans string,
	server testserviceserver.Interface,
) (
	client testserviceclient.Interface,
	observedLogs *observer.ObservedLogs,
	clientMetricsRoot *metrics.Root,
	serverMetricsRoot *metrics.Root,
	cleanup func(),
) {
	loggerCore, observedLogs := observer.New(zapcore.DebugLevel)
	clientMetricsRoot, serverMetricsRoot = metrics.New(), metrics.New()

	serverAddr, cleanupServer := newServer(t, trans, loggerCore, serverMetricsRoot, server)
	client, cleanupClient := newClient(t, trans, serverAddr, loggerCore, clientMetricsRoot)

	_ = observedLogs.TakeAll() // ignore all start up logs

	return client, observedLogs, clientMetricsRoot, serverMetricsRoot, func() {
		cleanupServer()
		cleanupClient()
	}
}

func newServer(t *testing.T, transportType string, loggerCore zapcore.Core, metricsRoot *metrics.Root, server testserviceserver.Interface) (addr string, cleanup func()) {
	var inbound transport.Inbound

	switch transportType {
	case tchannel.TransportName:
		listen, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		trans, err := tchannel.NewTransport(
			tchannel.ServiceName(_serverName),
			tchannel.Listener(listen))
		require.NoError(t, err)

		inbound = trans.NewInbound()
		addr = listen.Addr().String()

	case http.TransportName:
		hInbound := http.NewTransport().NewInbound("127.0.0.1:0")
		defer func() { addr = "http://" + hInbound.Addr().String() }() // can only get addr after dispatcher has started
		inbound = hInbound

	default:
		t.Fatal("unknown transport")
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging: yarpc.LoggingConfig{
			Zap: zap.New(loggerCore),
		},
		Metrics: yarpc.MetricsConfig{
			Metrics: metricsRoot.Scope(),
		},
	})

	//dispatcher.Register(testserviceserver.New(&testServer{}))
	dispatcher.Register(testserviceserver.New(server))

	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func newClient(t *testing.T, transportType string, serverAddr string, loggerCore zapcore.Core, metricsRoot *metrics.Root) (client testserviceclient.Interface, cleanup func()) {
	var out transport.UnaryOutbound

	switch transportType {
	case tchannel.TransportName:
		trans, err := tchannel.NewTransport(tchannel.ServiceName(_clientName))
		require.NoError(t, err)
		out = trans.NewSingleOutbound(serverAddr)

	case http.TransportName:
		out = http.NewTransport().NewSingleOutbound(serverAddr)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       out,
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
