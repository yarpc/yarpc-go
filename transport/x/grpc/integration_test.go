// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestBasicYarpc(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYarpc(context.Background(), "foo", "bar"))
		value, err := e.GetValueYarpc(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestBasicGRPC(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueGRPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueGRPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestYarpcMetadata(t *testing.T) {
	t.Parallel()
	var md metadata.MD
	doWithTestEnv(t, []InboundOption{withInboundUnaryInterceptor(newMetadataUnaryServerInterceptor(&md))}, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYarpc(context.Background(), "foo", "bar"))
		assert.Len(t, md["user-agent"], 1)
		assert.True(t, strings.Contains(md["user-agent"][0], UserAgent))
	})
}

func doWithTestEnv(t *testing.T, inboundOptions []InboundOption, outboundOptions []OutboundOption, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv(inboundOptions, outboundOptions)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

type testEnv struct {
	Inbound             *Inbound
	Outbound            *Outbound
	ClientConn          *grpc.ClientConn
	ContextWrapper      *grpcheader.ContextWrapper
	ClientConfig        transport.ClientConfig
	Procedures          []transport.Procedure
	KeyValueGRPCClient  examplepb.KeyValueClient
	KeyValueYarpcClient examplepb.KeyValueYarpcClient
	KeyValueYarpcServer *example.KeyValueYarpcServer
}

func newTestEnv(inboundOptions []InboundOption, outboundOptions []OutboundOption) (_ *testEnv, err error) {
	keyValueYarpcServer := example.NewKeyValueYarpcServer()
	procedures := examplepb.BuildKeyValueYarpcProcedures(keyValueYarpcServer)
	testRouter := newTestRouter(procedures)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	inbound := NewInbound(listener, inboundOptions...)
	inbound.SetRouter(testRouter)
	if err := inbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, inbound.Stop())
		}
	}()

	clientConn, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, clientConn.Close())
		}
	}()
	keyValueClient := examplepb.NewKeyValueClient(clientConn)

	outbound := NewSingleOutbound(listener.Addr().String(), outboundOptions...)
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

	contextWrapper := grpcheader.NewContextWrapper("example-client", "example")

	return &testEnv{
		inbound,
		outbound,
		clientConn,
		contextWrapper,
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
	response, err := e.KeyValueGRPCClient.GetValue(e.ContextWrapper.Wrap(ctx), &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (e *testEnv) SetValueGRPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	_, err := e.KeyValueGRPCClient.SetValue(e.ContextWrapper.Wrap(ctx), &examplepb.SetValueRequest{key, value})
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
