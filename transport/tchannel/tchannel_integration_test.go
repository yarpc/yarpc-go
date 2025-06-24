// Copyright (c) 2025 Uber Technologies, Inc.
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

package tchannel_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/transport/internal/tls/testscenario"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestHandleResourceExhausted(t *testing.T) {
	serviceName := "test-service"
	procedureName := "test-procedure"
	port := uint16(8000)

	resourceExhaustedHandler := &types.UnaryHandler{
		Handler: api.UnaryHandlerFunc(func(context.Context, *transport.Request, transport.ResponseWriter) error {
			// eg: simulating a rate limiter that's reached its limit
			return yarpcerrors.Newf(yarpcerrors.CodeResourceExhausted, "resource exhausted: rate limit exceeded")
		})}

	service := yarpctest.TChannelService(
		yarpctest.Name(serviceName),
		yarpctest.Port(port),
		yarpctest.Proc(yarpctest.Name(procedureName), resourceExhaustedHandler),
	)
	require.NoError(t, service.Start(t))
	defer func() { require.NoError(t, service.Stop(t)) }()

	requests := yarpctest.ConcurrentAction(
		yarpctest.TChannelRequest(
			yarpctest.Service(serviceName),
			yarpctest.Port(port),
			yarpctest.Procedure(procedureName),
			yarpctest.GiveTimeout(time.Millisecond*100),

			// resource exhausted error should be returned
			yarpctest.WantError("resource exhausted: rate limit exceeded"),
		),
		10,
	)
	requests.Run(t)
}

func TestDialerOption(t *testing.T) {
	customDialerErr := errors.New("error from custom dialer function")

	trans, err := tchannel.NewTransport(
		tchannel.ServiceName("foo-service"),
		tchannel.Dialer(
			func(ctx context.Context, network, hostPort string) (net.Conn, error) {
				return nil, customDialerErr
			}))
	require.NoError(t, err)
	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()

	out := trans.NewOutbound(peer.NewSingle(peertest.MockPeerIdentifier("bar-peer"), trans))
	require.NoError(t, out.Start())
	defer func() { assert.NoError(t, out.Stop()) }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = out.Call(ctx, &transport.Request{Service: "bar-service"})
	require.Error(t, err, "expected dialer error")
	assert.Contains(t, err.Error(), customDialerErr.Error())
}

func TestInboundTLS(t *testing.T) {
	scenario := testscenario.Create(t, time.Minute, time.Minute)

	tests := []struct {
		desc        string
		isClientTLS bool
	}{
		{desc: "plaintext_client_permissive_tls_inbound"},
		{desc: "tls_client_permissive_tls_inbound", isClientTLS: true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			options := []tchannel.TransportOption{
				tchannel.InboundTLSConfiguration(scenario.ServerTLSConfig()),
				tchannel.InboundTLSMode(yarpctls.Permissive),
				tchannel.ServiceName("test-svc"),
				tchannel.ListenAddr("127.0.0.1:0"),
			}
			if tt.isClientTLS {
				tchannel.Dialer(func(ctx context.Context, network, hostPort string) (net.Conn, error) {
					return tls.Dial(network, hostPort, scenario.ClientTLSConfig())
				})
			}
			tr, err := tchannel.NewTransport(options...)
			require.NoError(t, err)
			inbound := tr.NewInbound()
			inbound.SetRouter(testRouter{proc: transport.Procedure{HandlerSpec: transport.NewUnaryHandlerSpec(testServer{})}})

			require.NoError(t, tr.Start())
			defer tr.Stop()
			require.NoError(t, inbound.Start())
			defer inbound.Stop()

			outbound := tr.NewSingleOutbound(tr.ListenAddr())
			require.NoError(t, outbound.Start())
			defer outbound.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			res, err := outbound.Call(ctx, &transport.Request{
				Service:   "test-svc-1",
				Procedure: "test-proc",
				Body:      strings.NewReader("hello"),
			})
			require.NoError(t, err)

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(resBody))
		})
	}
}

func TestTLSOutbound(t *testing.T) {
	scenario := testscenario.Create(t, time.Minute, time.Minute)
	serverTransport, err := tchannel.NewTransport(
		tchannel.InboundTLSConfiguration(scenario.ServerTLSConfig()),
		tchannel.InboundTLSMode(yarpctls.Enforced), // reject plaintext connections.
		tchannel.ServiceName("test-svc"),
		tchannel.ListenAddr("127.0.0.1:0"),
	)
	require.NoError(t, err)

	inbound := serverTransport.NewInbound()
	inbound.SetRouter(testRouter{proc: transport.Procedure{HandlerSpec: transport.NewUnaryHandlerSpec(testServer{})}})
	require.NoError(t, serverTransport.Start())
	defer serverTransport.Stop()
	require.NoError(t, inbound.Start())
	defer inbound.Stop()

	clientTransport, err := tchannel.NewTransport(tchannel.ServiceName("test-client-svc"))
	require.NoError(t, err)
	// Create outbound tchannel with client tls config.
	peerTransport, err := clientTransport.CreateTLSOutboundChannel(scenario.ClientTLSConfig(), "test-svc")
	require.NoError(t, err)
	outbound := serverTransport.NewOutbound(peer.NewSingle(hostport.Identify(serverTransport.ListenAddr()), peerTransport))
	require.NoError(t, clientTransport.Start())
	defer clientTransport.Stop()
	require.NoError(t, outbound.Start())
	defer outbound.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	res, err := outbound.Call(ctx, &transport.Request{
		Service:   "test-svc-1",
		Procedure: "test-proc",
		Body:      strings.NewReader("hello"),
	})
	require.NoError(t, err)

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(resBody))
}

type testRouter struct {
	proc transport.Procedure
}

func (t testRouter) Procedures() []transport.Procedure {
	return []transport.Procedure{t.proc}
}

func (t testRouter) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return t.proc.HandlerSpec, nil
}

type testServer struct{}

func (testServer) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	resw.Write(data)
	return nil
}
