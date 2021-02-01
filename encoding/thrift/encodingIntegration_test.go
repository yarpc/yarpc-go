package thrift_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceserver"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpctest"
)

type testStructure struct {
	name   string
	req    *yarpctest.Call
	expReq map[string]string
}

var allTests map[string]testStructure

func validateReq(testname string, ctx context.Context) (bool, string) {
	test := allTests[testname]
	call := yarpc.CallFromContext(ctx)
	for name, value := range test.expReq {
		switch name {
		case "CallerProcedure":
			if call.CallerProcedure() != value {
				err := "TestName(" + testname + ") - CallerProcedure '" + call.CallerProcedure() + "' does not match with expected value '" + value + "'"
				return false, err
			}
		case "Procedure":
			if call.Procedure() != value {
				err := "TestName(" + testname + ") - Procedure '" + call.Procedure() + "' does not match with expected value '" + value + "'"
				return false, err
			}
		}
	}
	return true, ""
}

func runTest(t *testing.T, test testStructure, client testserviceclient.Interface, testName string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = yarpctest.ContextWithCall(ctx, test.req)
	defer cancel()

	_, err := client.Call(ctx, testName)
	require.NoError(t, err, "unexpected error")
}

func TestThriftMetrics1(t *testing.T) {
	transports := []string{tchannel.TransportName, http.TransportName}

	tests := []testStructure{
		{
			name: "test1",
			req: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expReq: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "TestService::Call",
			},
		},
	}
	allTests = make(map[string]testStructure)

	for _, trans := range transports {
		t.Run(trans+" thift call", func(t *testing.T) {
			client, cleanup := CreateClientAndServer(t, trans, testServer1{})
			defer cleanup()

			for _, test := range tests {
				testName := trans + "_" + test.name
				allTests[testName] = test
				runTest(t, test, client, testName)
			}
		})
	}
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

type testServer1 struct{}

func (testServer1) Call(ctx context.Context, val string) (string, error) {

	ok, err := validateReq(val, ctx)
	if ok == true {
		return val, nil
	}

	return "", &test.ExceptionWithoutCode{Val: err}
}
