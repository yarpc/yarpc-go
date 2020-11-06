package protobuf_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/yarpcerrors"
)

type errorServer struct{}

func (errorServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	testDetails := []proto.Message{
		&types.StringValue{Value: "string value"},
		&types.Int32Value{Value: 100},
	}
	return nil,
		protobuf.NewError(yarpcerrors.CodeInvalidArgument, msg.Value,
			protobuf.WithErrorDetails(testDetails...))
}

func (errorServer) Duplex(stream testpb.TestServiceDuplexYARPCServer) error {
	return nil
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

	_, err = client.Unary(ctx, &testpb.TestMessage{Value: "error msg"})
	assert.NotNil(t, err, "unexpected nil error")
	st := yarpcerrors.FromError(err)
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, st.Code(), "unexpected error code")
	assert.Equal(t, "error msg", st.Message(), "unexpected error message")
	expectedDetails := []interface{}{
		&types.StringValue{Value: "string value"},
		&types.Int32Value{Value: 100},
	}
	actualDetails := protobuf.GetErrorDetails(err)
	assert.Equal(t, expectedDetails, actualDetails, "unexpected error details")
}

type errorRawServer struct{}

func (errorRawServer) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	testDetails := []proto.Message{
		&types.StringValue{Value: "string value"},
		&types.Int32Value{Value: 100},
	}
	return protobuf.NewError(yarpcerrors.CodeInvalidArgument, "error message",
		protobuf.WithErrorDetails(testDetails...))
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

	var rpcStatus rpc.Status
	proto.Unmarshal(yarpcStatus.Details(), &rpcStatus)
	status := status.FromProto(&rpcStatus)
	expectedDetails := []interface{}{
		&types.StringValue{Value: "string value"},
		&types.Int32Value{Value: 100},
	}
	assert.Equal(t, expectedDetails, status.Details(), "unexpected error details")

}
