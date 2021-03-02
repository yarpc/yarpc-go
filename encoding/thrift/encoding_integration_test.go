package thrift_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceserver"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpctest"
)

func TestThriftEncoding(t *testing.T) {
	for _, test := range []struct {
		desc            string
		transport       string
		request         *yarpctest.Call
		expectedHeaders map[string]string
	}{
		{
			desc:      "test1",
			transport: tchannel.TransportName,
			request: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expectedHeaders: map[string]string{
				"CallerProcedure": "ABC1",
				"Procedure":       "TestService::Call",
			},
		},
		{
			desc:      "test2",
			transport: tchannel.TransportName,
			request:   &yarpctest.Call{},
			expectedHeaders: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "TestService::Call",
			},
		},
		{
			desc:      "test3",
			transport: http.TransportName,
			request: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expectedHeaders: map[string]string{
				"CallerProcedure": "ABC1",
				"Procedure":       "TestService::Call",
			},
		},
		{
			desc:      "test4",
			transport: http.TransportName,
			request:   &yarpctest.Call{},
			expectedHeaders: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "TestService::Call",
			},
		},
	} {

		t.Run(test.desc, func(t *testing.T) {
			client, cleanup := CreateClientAndServer(t, test.transport, testServer1{})
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			ctx = yarpctest.ContextWithCall(ctx, test.request)
			defer cancel()

			headersStr := mapToStr(test.expectedHeaders)
			_, err := client.Call(ctx, headersStr)
			require.NoError(t, err, "unexpected call error")
		})
	}

}

func mapToStr(m map[string]string) string {
	var arr []string
	for name, value := range m {
		arr = append(arr, fmt.Sprintf("%s|%s", name, value))
	}
	return strings.Join(arr, " ")
}

func CreateClientAndServer(
	t *testing.T,
	trans string,
	server testserviceserver.Interface,
) (
	client testserviceclient.Interface,
	cleanup func(),
) {
	serverAddr, cleanupServer := CreateServer(t, trans, server)
	client, cleanupClient := CreateClient(t, trans, serverAddr)

	return client, func() {
		cleanupServer()
		cleanupClient()
	}
}

func CreateServer(t *testing.T, transportType string, server testserviceserver.Interface) (addr string, cleanup func()) {
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
	})

	dispatcher.Register(testserviceserver.New(server))

	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func CreateClient(t *testing.T, transportType string, serverAddr string) (client testserviceclient.Interface, cleanup func()) {
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
	})

	client = testserviceclient.New(dispatcher.ClientConfig(_serverName))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return client, cleanup
}

func validateHeader(headerName string, expectedValue string, call *yarpc.Call) (result bool, err string) {
	result = true
	err = ""
	switch headerName {
	case "CallerProcedure":
		value := call.CallerProcedure()
		if value != expectedValue {
			result = false
			err = fmt.Sprintf("CallerProcedure validation failed - expected('%s'), received('%s')", expectedValue, value)
		}
	case "Procedure":
		value := call.Procedure()
		if value != expectedValue {
			result = false
			err = fmt.Sprintf("Procedure validation failed - expected('%s'), received('%s')", expectedValue, value)
		}
	default:
		result = false
		err = fmt.Sprintf("Invalid input header: '%s'", headerName)
	}
	return result, err
}

type testServer1 struct{}

func (testServer1) Call(ctx context.Context, val string) (string, error) {
	call := yarpc.CallFromContext(ctx)
	if call == nil {
		return "", errors.New("Invalid call context")
	}

	for _, pair := range strings.Split(val, " ") {
		arr := strings.Split(pair, "|")
		header := arr[0]
		value := arr[1]
		if _, err := validateHeader(header, value, call); err != "" {
			return "", errors.New(err)
		}
	}

	return val, nil
}
