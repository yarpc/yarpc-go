// Copyright (c) 2018 Uber Technologies, Inc.
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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
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
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/procedure"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestYARPCBasic(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		_, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeNotFound, "foo"), err)
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueYARPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestGRPCBasic(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.NotFound, "foo"), err)
		assert.NoError(t, e.SetValueGRPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueGRPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func TestTLSWithYARPC(t *testing.T) {
	for _, test := range []struct {
		clientValidity      time.Duration
		serverValidity      time.Duration
		expectedErrContains string
		name                string
	}{
		{
			clientValidity: 1 * time.Minute,
			serverValidity: 1 * time.Minute,
			name:           "valid certs both sides",
		},
		{
			clientValidity:      1 * time.Minute,
			serverValidity:      -1 * time.Minute,
			expectedErrContains: "transport: authentication handshake failed: x509: certificate has expired or is not yet valid",
			name:                "invalid server cert",
		},
		{
			clientValidity:      -1 * time.Minute,
			serverValidity:      1 * time.Minute,
			expectedErrContains: "remote error: tls: bad certificate",
			name:                "invalid client cert",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			scenario := createTLSScenario(t, test.clientValidity, test.serverValidity)

			serverCreds := credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{{
					Certificate: [][]byte{scenario.ServerCert.Raw},
					Leaf:        scenario.ServerCert,
					PrivateKey:  scenario.ServerKey,
				}},
				ClientAuth: tls.RequireAndVerifyClientCert,
				ClientCAs:  scenario.CAs,
			})
			inboundTLSOpt := Creds(serverCreds)

			clientCreds := credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{{
					Certificate: [][]byte{scenario.ClientCert.Raw},
					Leaf:        scenario.ClientCert,
					PrivateKey:  scenario.ClientKey,
				}},
				RootCAs: scenario.CAs,
			})
			dialTLSOpt := WithTransportCredentials(clientCreds)

			doWithTestEnv(t, nil, []InboundOption{inboundTLSOpt}, nil, []DialOption{dialTLSOpt}, func(t *testing.T, e *testEnv) {
				err := e.SetValueYARPC(context.Background(), "foo", "bar")
				if test.expectedErrContains == "" {
					assert.NoError(t, err)
				} else {
					assert.Contains(t, err.Error(), test.expectedErrContains)
				}

				err = e.SetValueGRPC(context.Background(), "foo", "bar")
				if test.expectedErrContains == "" {
					assert.NoError(t, err)
				} else {
					assert.Contains(t, err.Error(), test.expectedErrContains)
				}
			})
		})

	}
}

func TestYARPCWellKnownError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "bar 1"), err)
	})
}

func TestYARPCNamedError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"), err)
	})
}

func TestYARPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""), err)
	})
}

func TestGRPCWellKnownError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.FailedPrecondition, "bar 1"), err)
	})
}

func TestGRPCNamedError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.Unknown, "bar: baz 1"), err)
	})
}

func TestGRPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.Unknown, "bar"), err)
	})
}

func TestYARPCResponseAndError(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.NoError(t, err)
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		value, err := e.GetValueYARPC(context.Background(), "foo")
		assert.Equal(t, "bar", value)
		assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "bar 1"), err)
	})
}

func TestGRPCResponseAndError(t *testing.T) {
	t.Skip("grpc-go clients do not support returning both a response and error as of now")
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.NoError(t, err)
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		value, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, "bar", value)
		assert.Equal(t, status.Error(codes.FailedPrecondition, "bar 1"), err)
	})
}

func TestYARPCMaxMsgSize(t *testing.T) {
	t.Parallel()
	value := strings.Repeat("a", defaultServerMaxRecvMsgSize*2)
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		assert.Equal(t, yarpcerrors.CodeResourceExhausted, yarpcerrors.FromError(e.SetValueYARPC(context.Background(), "foo", value)).Code())
	})
	doWithTestEnv(t, []TransportOption{
		ClientMaxRecvMsgSize(math.MaxInt32),
		ClientMaxSendMsgSize(math.MaxInt32),
		ServerMaxRecvMsgSize(math.MaxInt32),
		ServerMaxSendMsgSize(math.MaxInt32),
	}, nil, nil, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", value))
		getValue, err := e.GetValueYARPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, value, getValue)
	})
}

func TestLargeEcho(t *testing.T) {
	t.Parallel()
	value := strings.Repeat("a", 32768)
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", value))
		getValue, err := e.GetValueYARPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, value, getValue)
	})
}

func TestApplicationErrorPropagation(t *testing.T) {
	t.Parallel()
	doWithTestEnv(t, nil, nil, nil, nil, func(t *testing.T, e *testEnv) {
		response, err := e.Call(
			context.Background(),
			"GetValue",
			&examplepb.GetValueRequest{Key: "foo"},
			protobuf.Encoding,
			transport.Headers{},
		)
		require.Equal(t, yarpcerrors.NotFoundErrorf("foo"), err)
		require.True(t, response.ApplicationError)

		response, err = e.Call(
			context.Background(),
			"SetValue",
			&examplepb.SetValueRequest{Key: "foo", Value: "hello"},
			protobuf.Encoding,
			transport.Headers{},
		)
		require.NoError(t, err)
		require.False(t, response.ApplicationError)

		response, err = e.Call(
			context.Background(),
			"GetValue",
			&examplepb.GetValueRequest{Key: "foo"},
			"bad_encoding",
			transport.Headers{},
		)
		require.True(t, yarpcerrors.IsInvalidArgument(err))
		require.False(t, response.ApplicationError)
	})
}

func doWithTestEnv(t *testing.T, transportOptions []TransportOption, inboundOptions []InboundOption, outboundOptions []OutboundOption, dialOptions []DialOption, f func(*testing.T, *testEnv), opts ...testEnvOption) {
	testEnv, err := newTestEnv(transportOptions, inboundOptions, outboundOptions, dialOptions, opts...)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

type testEnvOptions struct {
	tlsScenario *tlsScenario
}

type testEnvOption func(*testEnvOptions)

type testEnv struct {
	Caller              string
	Service             string
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

func newTestEnv(
	transportOptions []TransportOption,
	inboundOptions []InboundOption,
	outboundOptions []OutboundOption,
	dialOptions []DialOption,
	opts ...testEnvOption,
) (_ *testEnv, err error) {
	options := &testEnvOptions{}
	for _, opt := range opts {
		opt(options)
	}

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)
	testRouter := newTestRouter(procedures)

	t := NewTransport(transportOptions...)
	if err := t.Start(); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	inbound := t.NewInbound(listener, inboundOptions...)
	inbound.SetRouter(testRouter)
	if err := inbound.Start(); err != nil {
		return nil, err
	}

	var clientConn *grpc.ClientConn

	clientConn, err = grpc.Dial(listener.Addr().String(), newDialOptions(dialOptions).grpcOptions()...)
	if err != nil {
		return nil, err
	}
	keyValueClient := examplepb.NewKeyValueClient(clientConn)

	chooser := peer.NewSingle(hostport.Identify(listener.Addr().String()), t.NewDialer(dialOptions...))
	outbound := t.NewOutbound(chooser, outboundOptions...)

	if err := outbound.Start(); err != nil {
		return nil, err
	}

	caller := "example-client"
	service := "example"
	clientConfig := clientconfig.MultiOutbound(
		caller,
		service,
		transport.Outbounds{
			ServiceName: caller,
			Unary:       outbound,
		},
	)
	keyValueYARPCClient := examplepb.NewKeyValueYARPCClient(clientConfig)

	contextWrapper := grpcctx.NewContextWrapper().
		WithCaller("example-client").
		WithService("example").
		WithEncoding(string(protobuf.Encoding))

	return &testEnv{
		caller,
		service,
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

func (e *testEnv) Call(
	ctx context.Context,
	methodName string,
	message proto.Message,
	encoding transport.Encoding,
	headers transport.Headers,
) (*transport.Response, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	return e.Outbound.Call(
		ctx,
		&transport.Request{
			Caller:   e.Caller,
			Service:  e.Service,
			Encoding: encoding,
			Procedure: procedure.ToName(
				"uber.yarpc.internal.examples.protobuf.example.KeyValue",
				methodName,
			),
			Headers: headers,
			Body:    bytes.NewReader(data),
		},
	)
}

func (e *testEnv) GetValueYARPC(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	response, err := e.KeyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{Key: key})
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func (e *testEnv) SetValueYARPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	_, err := e.KeyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: value})
	return err
}

func (e *testEnv) GetValueGRPC(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	response, err := e.KeyValueGRPCClient.GetValue(e.ContextWrapper.Wrap(ctx), &examplepb.GetValueRequest{Key: key})
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func (e *testEnv) SetValueGRPC(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	_, err := e.KeyValueGRPCClient.SetValue(e.ContextWrapper.Wrap(ctx), &examplepb.SetValueRequest{Key: key, Value: value})
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

type tlsScenario struct {
	CAs        *x509.CertPool
	ServerCert *x509.Certificate
	ServerKey  *ecdsa.PrivateKey
	ClientCert *x509.Certificate
	ClientKey  *ecdsa.PrivateKey
}

func createTLSScenario(t *testing.T, clientValidity time.Duration, serverValidity time.Duration) tlsScenario {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test ca",
			},
			SerialNumber:          big.NewInt(1),
			BasicConstraintsValid: true,
			IsCA:      true,
			KeyUsage:  x509.KeyUsageCertSign,
			NotBefore: time.Now(),
			NotAfter:  time.Now().Add(10 * time.Minute),
		},
		&x509.Certificate{},
		caKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caBytes)
	require.NoError(t, err)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serverCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "server",
			},
			NotAfter:     time.Now().Add(serverValidity),
			SerialNumber: big.NewInt(2),
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		serverKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverCertBytes)
	require.NoError(t, err)

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	clientCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "client",
			},
			NotAfter:     time.Now().Add(clientValidity),
			SerialNumber: big.NewInt(3),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		clientKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	clientCert, err := x509.ParseCertificate(clientCertBytes)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(ca)

	return tlsScenario{
		CAs:        pool,
		ServerCert: serverCert,
		ServerKey:  serverKey,
		ClientCert: clientCert,
		ClientKey:  clientKey,
	}
}
