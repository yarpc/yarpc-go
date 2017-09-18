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
	"math"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestYARPCBasic(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		_, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, yarpcerrors.NotFoundErrorf("foo"), err)
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueYARPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestGRPCBasic(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.NotFound, "foo"), err)
		assert.NoError(t, e.SetValueGRPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueGRPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestYARPCMetadata(t *testing.T) {
	t.Parallel()
	var md metadata.MD
	doWithTestEnv(t, nil, []InboundOption{withInboundUnaryInterceptor(newMetadataUnaryServerInterceptor(&md))}, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", "bar"))
		assert.Len(t, md["user-agent"], 1)
		assert.True(t, strings.Contains(md["user-agent"][0], UserAgent))
	})
}

func TestYARPCWellKnownError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		_, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, yarpcerrors.FailedPreconditionErrorf("bar 1"), err)
	})
}

func TestYARPCNamedError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(yarpcerrors.NamedErrorf("bar", "baz 1"))
		_, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, yarpcerrors.NamedErrorf("bar", "baz 1"), err)
	})
}

func TestYARPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(yarpcerrors.NamedErrorf("bar", ""))
		_, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, yarpcerrors.NamedErrorf("bar", ""), err)
	})
}

func TestGRPCWellKnownError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.FailedPrecondition, "bar 1"), err)
	})
}

func TestGRPCNamedError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(yarpcerrors.NamedErrorf("bar", "baz 1"))
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.Unknown, "bar: baz 1"), err)
	})
}

func TestGRPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(yarpcerrors.NamedErrorf("bar", ""))
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.Unknown, "bar"), err)
	})
}

func TestYARPCMaxMsgSize(t *testing.T) {
	t.Parallel()
	value := strings.Repeat("a", defaultServerMaxRecvMsgSize*2)
	doWithTestEnv(t, nil, nil, nil, func(t *testing.T, e *testEnv) {
		assert.Equal(t, yarpcerrors.CodeResourceExhausted, yarpcerrors.ErrorCode(e.SetValueYARPC(context.Background(), "foo", value)))
	})
	doWithTestEnv(t, []TransportOption{
		ClientMaxRecvMsgSize(math.MaxInt32),
		ClientMaxSendMsgSize(math.MaxInt32),
		ServerMaxRecvMsgSize(math.MaxInt32),
		ServerMaxSendMsgSize(math.MaxInt32),
	}, nil, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", value))
		getValue, err := e.GetValueYARPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, value, getValue)
	})
}

func doWithTestEnv(t *testing.T, transportOptions []TransportOption, inboundOptions []InboundOption, outboundOptions []OutboundOption, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv(transportOptions, inboundOptions, outboundOptions)
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
	ContextWrapper      *grpcctx.ContextWrapper
	ClientConfig        transport.ClientConfig
	Procedures          []transport.Procedure
	KeyValueGRPCClient  examplepb.KeyValueClient
	KeyValueYARPCClient examplepb.KeyValueYARPCClient
	KeyValueYARPCServer *example.KeyValueYARPCServer
}

func newTestEnv(transportOptions []TransportOption, inboundOptions []InboundOption, outboundOptions []OutboundOption) (_ *testEnv, err error) {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)
	testRouter := newTestRouter(procedures)

	t := NewTransport(transportOptions...)
	if err := t.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, t.Stop())
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	inbound := t.NewInbound(listener, inboundOptions...)
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

	outbound := t.NewSingleOutbound(listener.Addr().String(), outboundOptions...)
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
	keyValueYARPCClient := examplepb.NewKeyValueYARPCClient(clientConfig)

	contextWrapper := grpcctx.NewContextWrapper().
		WithCaller("example-client").
		WithService("example").
		WithEncoding(string(protobuf.Encoding))

	return &testEnv{
		inbound,
		outbound,
		clientConn,
		contextWrapper,
		clientConfig,
		procedures,
		keyValueClient,
		keyValueYARPCClient,
		keyValueYARPCServer,
	}, nil
}

func (e *testEnv) GetValueYARPC(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	response, err := e.KeyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (e *testEnv) SetValueYARPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	_, err := e.KeyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{key, value})
	return err
}

func (e *testEnv) GetValueGRPC(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	response, err := e.KeyValueGRPCClient.GetValue(e.ContextWrapper.Wrap(ctx), &examplepb.GetValueRequest{key})
	if err != nil {
		return "", err
	}
	return response.Value, nil
}

func (e *testEnv) SetValueGRPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
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
