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
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpctest"
)

func TestProtobufEncoding(t *testing.T) {
	client, cleanup := initalizeClientAndServer(t, &encodingIntegrationTestServer{})

	defer cleanup()
	for _, test := range []struct {
		desc            string
		request         *yarpctest.Call
		expectedHeaders map[string]string
	}{
		{
			desc: "test1",
			request: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expectedHeaders: map[string]string{
				"CallerProcedure": "ABC1",
				"Procedure":       "uber.yarpc.encoding.protobuf.Test::Unary",
			},
		},
		{
			desc:    "test2",
			request: &yarpctest.Call{},
			expectedHeaders: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "uber.yarpc.encoding.protobuf.Test::Unary",
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			ctx = yarpctest.ContextWithCall(ctx, test.request)
			defer cancel()

			headersStr := mapToStr(test.expectedHeaders)
			_, err := client.Unary(ctx, &testpb.TestMessage{Value: headersStr})
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

func initalizeClientAndServer(t *testing.T, server testpb.TestYARPCServer) (client testpb.TestYARPCClient, cleanup func()) {
	serverAddr, cleanupServer := createServer(t, server)
	client, cleanupClient := createClient(t, serverAddr)
	return client, func() {
		cleanupServer()
		cleanupClient()
	}
}

func createServer(t *testing.T, server testpb.TestYARPCServer) (addr string, cleanup func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
	})

	dispatcher.Register(testpb.BuildTestYARPCProcedures(server))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr = inbound.Addr().String()
	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func createClient(t *testing.T, serverAddr string) (client testpb.TestYARPCClient, cleanup func()) {
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
	})

	client = testpb.NewTestYARPCClient(dispatcher.ClientConfig(_serverName))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return client, cleanup
}

func validateHeader(headerName string, expectedValue string, call *yarpc.Call) (err string) {
	err = ""
	switch headerName {
	case "CallerProcedure":
		value := call.CallerProcedure()
		if value != expectedValue {
			err = fmt.Sprintf("CallerProcedure validation failed - expected('%s'), received('%s')", expectedValue, value)
		}
	case "Procedure":
		value := call.Procedure()
		if value != expectedValue {
			err = fmt.Sprintf("Procedure validation failed - expected('%s'), received('%s')", expectedValue, value)
		}
	default:
		err = fmt.Sprintf("Invalid input header: '%s'", headerName)
	}
	return err
}

type encodingIntegrationTestServer struct{}

func (encodingIntegrationTestServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	call := yarpc.CallFromContext(ctx)
	if call == nil {
		return nil, errors.New("Invalid call context")
	}

	for _, pair := range strings.Split(msg.Value, " ") {
		arr := strings.Split(pair, "|")
		header := arr[0]
		value := arr[1]
		if err := validateHeader(header, value, call); err != "" {
			return nil, errors.New(err)
		}
	}

	return msg, nil
}

func (encodingIntegrationTestServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
	return errors.New("stream handler is not implemented")
}
