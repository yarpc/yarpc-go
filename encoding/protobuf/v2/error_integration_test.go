// Copyright (c) 2024 Uber Technologies, Inc.
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

package v2_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	v2 "go.uber.org/yarpc/encoding/protobuf/v2"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpcerrors"
	rpc_status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	_clientName = "caller"
	_serverName = "callee"
)

type errorServer struct{}

func (errorServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	testDetails := []proto.Message{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	return nil,
		v2.NewError(yarpcerrors.CodeInvalidArgument, msg.Value,
			v2.WithErrorDetails(testDetails...))
}

func (errorServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
	testDetails := []proto.Message{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	return v2.NewError(yarpcerrors.CodeInvalidArgument, msg.Value,
		v2.WithErrorDetails(testDetails...))
}

func TestProtoGrpcServerErrorDetails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{},
		Metrics:  yarpc.MetricsConfig{},
	})

	dispatcher.Register(testpb.BuildTestYARPCProcedures(&errorServer{}))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr := inbound.Addr().String()
	outbound := grpc.NewTransport().NewSingleOutbound(addr)
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       outbound,
				Stream:      outbound,
			},
		},
		Logging: yarpc.LoggingConfig{},
		Metrics: yarpc.MetricsConfig{},
	})

	client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig(_serverName))
	require.NoError(t, clientDispatcher.Start(), "could not start client dispatcher")

	defer func() {
		assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher")
		assert.NoError(t, clientDispatcher.Stop(), "could not stop client dispatcher")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	const errorMsg = "error msg"

	_, err = client.Unary(ctx, &testpb.TestMessage{Value: errorMsg})
	assert.NotNil(t, err, "unexpected nil error")
	st := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, st.Code(), "unexpected error code")
	assert.Equal(t, errorMsg, st.Message(), "unexpected error message")
	expectedDetails := []interface{}{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	actualDetails := v2.GetErrorDetails(err)
	for i := 0; i < len(expectedDetails); i++ {
		assert.True(t, proto.Equal(expectedDetails[i].(proto.Message), actualDetails[i].(proto.Message)), "unexpected error details")
	}
}

func TestProtoGrpcStreamServerErrorDetails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{},
		Metrics:  yarpc.MetricsConfig{},
	})

	dispatcher.Register(testpb.BuildTestYARPCProcedures(&errorServer{}))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr := inbound.Addr().String()
	outbound := grpc.NewTransport().NewSingleOutbound(addr)
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       outbound,
				Stream:      outbound,
			},
		},
		Logging: yarpc.LoggingConfig{},
		Metrics: yarpc.MetricsConfig{},
	})

	client := testpb.NewTestYARPCClient(clientDispatcher.ClientConfig(_serverName))
	require.NoError(t, clientDispatcher.Start(), "could not start client dispatcher")

	defer func() {
		assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher")
		assert.NoError(t, clientDispatcher.Stop(), "could not stop client dispatcher")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	const errorMsg = "stream error msg"
	expectedDetails := []interface{}{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}

	streamHandle, err := client.Duplex(ctx)
	assert.NoError(t, err, "unexpected error")

	err = streamHandle.Send(&testpb.TestMessage{Value: errorMsg})
	assert.NoError(t, err, "unexpected error")

	msg, err := streamHandle.Recv()
	assert.Nil(t, msg, "unexpected non-nil reply")
	assert.Error(t, err, "unexpected nil error")

	st := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, st.Code(), "unexpected error code")
	assert.Equal(t, errorMsg, st.Message(), "unexpected error message")

	actualDetails := v2.GetErrorDetails(err)
	for i := 0; i < len(expectedDetails); i++ {
		assert.True(t, proto.Equal(expectedDetails[i].(proto.Message), actualDetails[i].(proto.Message)), "unexpected error details")
	}
}

type errorRawServer struct{}

func (errorRawServer) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	testDetails := []proto.Message{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	return v2.NewError(yarpcerrors.CodeInvalidArgument, "error message",
		v2.WithErrorDetails(testDetails...))
}

func TestRawGrpcServerErrorDetails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{},
		Metrics:  yarpc.MetricsConfig{},
	})

	dispatcher.Register([]transport.Procedure{{
		Name:        "test::unary",
		HandlerSpec: transport.NewUnaryHandlerSpec(&errorRawServer{}),
		Encoding:    "raw",
	}})
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr := inbound.Addr().String()
	outbound := grpc.NewTransport().NewSingleOutbound(addr)
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       outbound,
				Stream:      outbound,
			},
		},
		Logging: yarpc.LoggingConfig{},
		Metrics: yarpc.MetricsConfig{},
	})

	client := raw.New(clientDispatcher.ClientConfig(_serverName))
	require.NoError(t, clientDispatcher.Start(), "could not start client dispatcher")

	defer func() {
		assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher")
		assert.NoError(t, clientDispatcher.Stop(), "could not stop client dispatcher")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Call(ctx, "test::unary", nil)
	assert.NotNil(t, err, "unexpected nil error")
	yarpcStatus := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcStatus.Code(), "unexpected error code")
	assert.Equal(t, "error message", yarpcStatus.Message(), "unexpected error message")

	var rpcStatus rpc_status.Status
	proto.Unmarshal(yarpcStatus.Details(), &rpcStatus)
	status := status.FromProto(&rpcStatus)
	expectedDetails := []interface{}{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	for i := 0; i < len(expectedDetails); i++ {
		assert.True(t, proto.Equal(expectedDetails[i].(proto.Message), status.Details()[i].(proto.Message)), "unexpected error details")
	}
}

func TestJSONGrpcServerErrorDetails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	inbound := grpc.NewTransport().NewInbound(listener)
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{},
		Metrics:  yarpc.MetricsConfig{},
	})

	dispatcher.Register(json.Procedure("test", func(ctx context.Context, req *struct{}) (*struct{}, error) {
		testDetails := []proto.Message{
			&wrappers.StringValue{Value: "string value"},
			&wrappers.Int32Value{Value: 100},
		}
		return nil, v2.NewError(yarpcerrors.CodeInvalidArgument, "error message",
			v2.WithErrorDetails(testDetails...))
	}))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	addr := inbound.Addr().String()
	outbound := grpc.NewTransport().NewSingleOutbound(addr)
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       outbound,
				Stream:      outbound,
			},
		},
		Logging: yarpc.LoggingConfig{},
		Metrics: yarpc.MetricsConfig{},
	})

	client := json.New(clientDispatcher.ClientConfig(_serverName))
	require.NoError(t, clientDispatcher.Start(), "could not start client dispatcher")

	defer func() {
		assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher")
		assert.NoError(t, clientDispatcher.Stop(), "could not stop client dispatcher")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = client.Call(ctx, "test", nil, nil)
	assert.NotNil(t, err, "unexpected nil error")
	yarpcStatus := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcStatus.Code(), "unexpected error code")
	assert.Equal(t, "error message", yarpcStatus.Message(), "unexpected error message")

	var rpcStatus rpc_status.Status
	proto.Unmarshal(yarpcStatus.Details(), &rpcStatus)
	status := status.FromProto(&rpcStatus)
	expectedDetails := []interface{}{
		&wrappers.StringValue{Value: "string value"},
		&wrappers.Int32Value{Value: 100},
	}
	for i := 0; i < len(expectedDetails); i++ {
		assert.True(t, proto.Equal(expectedDetails[i].(proto.Message), status.Details()[i].(proto.Message)), "unexpected error details")
	}
}
