// Copyright (c) 2026 Uber Technologies, Inc.
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

package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/prototest/example"
	"go.uber.org/yarpc/internal/prototest/examplepb"
	yarpcpeer "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
	"go.uber.org/zap"
)

func TestStreamingWithNoCtxDeadline(t *testing.T) {
	// This test ensures that we can use gRPC streaming without a context deadline
	// set. For long-lived streams, it should be unnecesary for users to set a
	// deadline; instead they should use context.WithCancel to cancel the stream.

	const serviceName = "service-name"

	// init YARPC transport / inbound / outbound
	grpcTransport := grpc.NewTransport()
	peerList := roundrobin.New(grpcTransport)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "could not start listener")
	inbound := grpcTransport.NewInbound(listener)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     serviceName,
		Inbounds: yarpc.Inbounds{inbound},
		Outbounds: yarpc.Outbounds{
			serviceName: {
				ServiceName: serviceName,
				Stream:      grpcTransport.NewOutbound(peerList),
			},
		},
	})
	dispatcher.Register(
		examplepb.BuildFooYARPCProcedures(
			example.NewFooYARPCServer(transport.NewHeaders())))

	require.NoError(t, dispatcher.Start(), "could not start dispatcher")
	defer func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }()

	// add streaming peer so we can call ourself
	err = peerList.Update(peer.ListUpdates{Additions: []peer.Identifier{
		hostport.PeerIdentifier(listener.Addr().String()),
	}})
	require.NoError(t, err, "could not add peer to peer list")

	waitForPeerAvailable(t, peerList, time.Second)

	// init streaming client
	client := examplepb.NewFooYARPCClient(dispatcher.ClientConfig(serviceName))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamClient, err := client.EchoBoth(ctx)
	require.NoError(t, err, "could not create client stream")

	// veryify we can send a request
	err = streamClient.Send(&examplepb.EchoBothRequest{
		Message:      "test message!",
		NumResponses: 0,
	})
	require.NoError(t, err, "could not send message")
	assert.NoError(t, streamClient.CloseSend(), "could not close stream")
}

// waitForPeerAvailable ensures that the peer becomes available before
// proceeding, and that we do not wait forever.
func waitForPeerAvailable(t *testing.T, peerList *roundrobin.List, wait time.Duration) {
	peerAvailable := make(chan struct{})
	go func() {
		for {
			if peerList.Peers()[0].Status().ConnectionStatus == peer.Available {
				close(peerAvailable)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-time.After(wait):
		t.Fatal("failed waiting to connect to peer")
	case <-peerAvailable:
		return
	}
}

func TestFoo(t *testing.T) {
	const (
		serviceName   = "test-service"
		procedureName = "test-procedure"

		appErrName    = "ProtoAppErrName"
		appErrDetails = " this is an app error detail string!"

		portName = "port"
	)

	handler := &types.UnaryHandler{
		Handler: api.UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
			// simulate Protobuf encoding setting `transport.ApplicationErrorMeta`
			metaSetter, ok := resw.(transport.ApplicationErrorMetaSetter)
			if !ok {
				return errors.New("missing transport.ApplicationErrorMetaSetter")
			}
			metaSetter.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
				Name:    appErrName,
				Details: appErrDetails,
			})
			return nil
		})}

	outboundMwAssertion := middleware.UnaryOutboundFunc(
		func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
			res, err := next.Call(ctx, req)

			// verify gRPC propagating `transport.ApplicationErrorMeta`
			require.NotNil(t, res.ApplicationErrorMeta, "missing transport.ApplicationErrorMeta")
			assert.Equal(t, appErrName, res.ApplicationErrorMeta.Name, "incorrect app error name")
			assert.Equal(t, appErrDetails, res.ApplicationErrorMeta.Details, "incorrect app error message")
			assert.Nil(t, res.ApplicationErrorMeta.Code, "unexpected code")

			return res, err
		})

	portProvider := yarpctest.NewPortProvider(t)
	service := yarpctest.GRPCService(
		yarpctest.Name(serviceName),
		portProvider.NamedPort(portName),
		yarpctest.Proc(yarpctest.Name(procedureName), handler),
	)
	require.NoError(t, service.Start(t))
	defer func() { assert.NoError(t, service.Stop(t)) }()

	request := yarpctest.GRPCRequest(
		yarpctest.Service(serviceName),
		portProvider.NamedPort(portName),
		yarpctest.Procedure(procedureName),
		yarpctest.GiveTimeout(time.Second),
		api.RequestOptionFunc(func(opts *api.RequestOpts) {
			opts.UnaryMiddleware = []middleware.UnaryOutbound{outboundMwAssertion}
		}),
	)
	request.Run(t)
}

// --- connection pool integration tests (public API only) ---

// withPoolTestEnv starts a gRPC server+client pair with the given transport
// options using only the public API, runs f, then tears everything down.
func withPoolTestEnv(t *testing.T, opts []grpc.TransportOption, f func(set, get func(ctx context.Context, key, value string) error)) {
	t.Helper()

	const svcName = "example"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Use a nop logger so goroutines that outlive the test (e.g. monitorConnWrapper
	// draining after dispatcher.Stop()) cannot trigger zaptest's after-test panic.
	opts = append(opts, grpc.Logger(zap.NewNop()))
	trans := grpc.NewTransport(opts...)

	chooser := yarpcpeer.NewSingle(hostport.PeerIdentifier(listener.Addr().String()), trans.NewDialer())
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     svcName,
		Inbounds: yarpc.Inbounds{trans.NewInbound(listener)},
		Outbounds: yarpc.Outbounds{
			svcName: {ServiceName: svcName, Unary: trans.NewOutbound(chooser)},
		},
	})
	dispatcher.Register(examplepb.BuildKeyValueYARPCProcedures(example.NewKeyValueYARPCServer()))
	require.NoError(t, dispatcher.Start())
	defer func() { assert.NoError(t, dispatcher.Stop()) }()

	client := examplepb.NewKeyValueYARPCClient(dispatcher.ClientConfig(svcName))

	set := func(ctx context.Context, key, value string) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err := client.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: value})
		return err
	}
	get := func(ctx context.Context, key, _ string) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err := client.GetValue(ctx, &examplepb.GetValueRequest{Key: key})
		return err
	}
	f(set, get)
}

// extPoolMetricSnapshot reads gauges and counters from a metrics root into maps.
func extPoolMetricSnapshot(root *metrics.Root) (gauges, counters map[string]int64) {
	snap := root.Snapshot()
	gauges = make(map[string]int64, len(snap.Gauges))
	for _, g := range snap.Gauges {
		gauges[g.Name] = g.Value
	}
	counters = make(map[string]int64, len(snap.Counters))
	for _, c := range snap.Counters {
		counters[c.Name] = c.Value
	}
	return
}

func TestConnectionPoolBasicRequest(t *testing.T) {
	t.Parallel()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(2),
		grpc.MaxConnections(5),
	}, func(set, get func(ctx context.Context, key, value string) error) {
		ctx := context.Background()
		require.NoError(t, set(ctx, "foo", "bar"))
		require.NoError(t, get(ctx, "foo", ""))
	})
}

func TestConnectionPoolMinConnectionsAtStartup(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(2),
		grpc.MaxConnections(5),
		grpc.Meter(root.Scope()),
	}, func(_, _ func(ctx context.Context, key, value string) error) {
		gauges, _ := extPoolMetricSnapshot(root)
		assert.Equal(t, int64(2), gauges["conn_pool_active_connections"],
			"pool should be pre-warmed to minConnections")
	})
}

func TestConnectionPoolScaleUpOnLoad(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(1),
		grpc.MaxConnections(5),
		grpc.MaxConcurrentStreams(2),
		grpc.ScaleUpThreshold(0.5), // threshold = 1
		grpc.Meter(root.Scope()),
	}, func(set, _ func(ctx context.Context, key, value string) error) {
		require.NoError(t, set(context.Background(), "foo", "bar"))

		assert.Eventually(t, func() bool {
			_, counters := extPoolMetricSnapshot(root)
			return counters["conn_pool_scale_up_total"] >= 1
		}, 3*time.Second, 10*time.Millisecond,
			"conn_pool_scale_up_total should increment after load exceeds threshold")
	})
}

func TestConnectionPoolConcurrentRequests(t *testing.T) {
	t.Parallel()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(2),
		grpc.MaxConnections(5),
	}, func(set, _ func(ctx context.Context, key, value string) error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		const concurrent = 20
		errs := make([]error, concurrent)
		var wg sync.WaitGroup
		wg.Add(concurrent)
		for i := range concurrent {
			go func() {
				defer wg.Done()
				errs[i] = set(ctx, fmt.Sprintf("key-%d", i), "value")
			}()
		}
		wg.Wait()
		for i, err := range errs {
			assert.NoError(t, err, "request %d should succeed", i)
		}
	})
}

func TestConnectionPoolDisabledFallback(t *testing.T) {
	t.Parallel()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(false),
	}, func(set, get func(ctx context.Context, key, value string) error) {
		ctx := context.Background()
		require.NoError(t, set(ctx, "foo", "bar"))
		require.NoError(t, get(ctx, "foo", ""))
	})
}

func TestConnectionPoolMinimalSingleConnection(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(1),
		grpc.MaxConnections(3),
		grpc.Meter(root.Scope()),
	}, func(set, get func(ctx context.Context, key, value string) error) {
		ctx := context.Background()
		require.NoError(t, set(ctx, "foo", "bar"))
		require.NoError(t, get(ctx, "foo", ""))

		gauges, _ := extPoolMetricSnapshot(root)
		assert.GreaterOrEqual(t, gauges["conn_pool_active_connections"], int64(1))
	})
}

func TestConnectionPoolMaxConnectionsCapRespected(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	withPoolTestEnv(t, []grpc.TransportOption{
		grpc.WithDynamicConnectionScaling(true),
		grpc.MinConnections(1),
		grpc.MaxConnections(1),
		grpc.MaxConcurrentStreams(2),
		grpc.ScaleUpThreshold(0.5),
		grpc.Meter(root.Scope()),
	}, func(set, get func(ctx context.Context, key, value string) error) {
		ctx := context.Background()
		require.NoError(t, set(ctx, "foo", "bar"))
		require.NoError(t, get(ctx, "foo", ""))

		assert.Eventually(t, func() bool {
			_, counters := extPoolMetricSnapshot(root)
			return counters["conn_pool_scale_up_total"] == 0
		}, 2*time.Second, 10*time.Millisecond,
			"scale-up must not dial when MaxConnections is already reached")

		gauges, _ := extPoolMetricSnapshot(root)
		assert.Equal(t, int64(1), gauges["conn_pool_active_connections"])
	})
}
