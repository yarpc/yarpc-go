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
	"net"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpcerrors"
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
				err := "CallerProcedure '" + call.CallerProcedure() + "' does match with expected value '" + value + "'"
				return false, err
			}
		case "Procedure":
			if call.Procedure() != value {
				err := "Procedure '" + call.Procedure() + "' does match with expected value '" + value + "'"
				return false, err
			}
		}
	}
	//fmt.Println("Entered in validateReq :  ", testname)
	return true, ""
}

func runTest(t *testing.T, test testStructure, client testpb.TestYARPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = yarpctest.ContextWithCall(ctx, test.req)
	defer cancel()

	_, err := client.Unary(ctx, &testpb.TestMessage{Value: test.name})
	require.NoError(t, err, "unexpected call error")
}

func TestProtobufEncoding(t *testing.T) {
	client, cleanup := initalizeClientAndServer(t, &encodingIntegrationTestServer{})

	defer cleanup()

	tests := []testStructure{
		{
			name: "test1",
			req: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expReq: map[string]string{
				"CallerProcedure": "ABC1",
				"Procedure":       "uber.yarpc.encoding.protobuf.Test::Unary",
			},
		},
		{
			name: "test2",
			req:  &yarpctest.Call{},
			expReq: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "uber.yarpc.encoding.protobuf.Test::Unary",
			},
		},
	}
	allTests = make(map[string]testStructure)
	for _, test := range tests {
		allTests[test.name] = test
		runTest(t, test, client)
	}

}

func initalizeClientAndServer(t *testing.T, server testpb.TestYARPCServer) (
	client testpb.TestYARPCClient,
	cleanup func(),
) {
	serverAddr, cleanupServer := CreateServer(t, server)
	client, cleanupClient := CreateClient(t, serverAddr)
	return client, func() {
		cleanupServer()
		cleanupClient()
	}
}

func CreateServer(t *testing.T, server testpb.TestYARPCServer) (addr string, cleanup func()) {
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

func CreateClient(t *testing.T, serverAddr string) (client testpb.TestYARPCClient, cleanup func()) {
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

type encodingIntegrationTestServer struct{}

func (encodingIntegrationTestServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	ok, err := validateReq(msg.Value, ctx)
	if ok == true {
		return &testpb.TestMessage{Value: msg.Value}, nil
	}

	details := []proto.Message{
		&types.StringValue{Value: err},
	}

	return nil, protobuf.NewError(yarpcerrors.CodeInvalidArgument, err, protobuf.WithErrorDetails(details...))
}

func (encodingIntegrationTestServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
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
