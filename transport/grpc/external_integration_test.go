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
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/prototest/example"
	"go.uber.org/yarpc/internal/prototest/examplepb"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
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
