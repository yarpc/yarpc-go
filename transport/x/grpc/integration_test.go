package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	ggrpc "google.golang.org/grpc"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYarpc(context.Background(), "foo", "bar"))
		value, err := e.GetValueYarpc(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func doWithTestEnv(t *testing.T, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv()
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

type testEnv struct {
	Inbound             *Inbound
	Outbound            *Outbound
	ClientConn          *ggrpc.ClientConn
	ClientConfig        transport.ClientConfig
	Procedures          []transport.Procedure
	KeyValueGRPCClient  examplepb.KeyValueClient
	KeyValueYarpcClient examplepb.KeyValueYarpcClient
	KeyValueYarpcServer *example.KeyValueYarpcServer
}

func newTestEnv() (_ *testEnv, err error) {
	keyValueYarpcServer := example.NewKeyValueYarpcServer()
	procedures := examplepb.BuildKeyValueYarpcProcedures(keyValueYarpcServer)
	testRouter := newTestRouter(procedures)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	inbound := NewInbound(listener)
	inbound.SetRouter(testRouter)
	if err := inbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, inbound.Stop())
		}
	}()

	clientConn, err := ggrpc.Dial(listener.Addr().String(), ggrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, clientConn.Close())
		}
	}()
	keyValueClient := examplepb.NewKeyValueClient(clientConn)

	outbound := NewSingleOutbound(listener.Addr().String())
	if err := outbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, outbound.Stop())
		}
	}()
	clientConfig := clientconfig.MultiOutbound(
		"example-client",
		"example",
		transport.Outbounds{
			ServiceName: "example-client",
			Unary:       outbound,
		},
	)
	keyValueYarpcClient := examplepb.NewKeyValueYarpcClient(clientConfig)

	return &testEnv{
		inbound,
		outbound,
		clientConn,
		clientConfig,
		procedures,
		keyValueClient,
		keyValueYarpcClient,
		keyValueYarpcServer,
	}, nil
}

func (e *testEnv) GetValueYarpc(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	response, err := e.KeyValueYarpcClient.GetValue(ctx, &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (e *testEnv) SetValueYarpc(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	_, err := e.KeyValueYarpcClient.SetValue(ctx, &examplepb.SetValueRequest{key, value})
	return err
}

func (e *testEnv) GetValueGRPC(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	response, err := e.KeyValueGRPCClient.GetValue(ctx, &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (e *testEnv) SetValueGRPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	_, err := e.KeyValueGRPCClient.SetValue(ctx, &examplepb.SetValueRequest{key, value})
	return err
}

func (e *testEnv) Close() error {
	return multierr.Combine(
		e.ClientConn.Close(),
		e.Outbound.Stop(),
		e.Inbound.Stop(),
	)
}

type testRouter struct {
	procedures []transport.Procedure
}

func newTestRouter(procedures []transport.Procedure) *testRouter {
	return &testRouter{procedures}
}

func (r *testRouter) Procedures() []transport.Procedure {
	return r.procedures
}

func (r *testRouter) Choose(_ context.Context, request *transport.Request) (transport.HandlerSpec, error) {
	for _, procedure := range r.procedures {
		if procedure.Name == request.Procedure {
			return procedure.HandlerSpec, nil
		}
	}
	return transport.HandlerSpec{}, fmt.Errorf("no procedure for name %s", request.Procedure)
}
