// Copyright (c) 2020 Uber Technologies, Inc.
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
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gogostatus "github.com/gogo/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/grpcctx"
	"go.uber.org/yarpc/internal/prototest/example"
	"go.uber.org/yarpc/internal/prototest/examplepb"
	"go.uber.org/yarpc/internal/testtime"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/procedure"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
)

func TestYARPCBasic(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
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
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		_, err := e.GetValueGRPC(context.Background(), "foo")
		assert.Equal(t, status.Error(codes.NotFound, "foo"), err)
		assert.NoError(t, e.SetValueGRPC(context.Background(), "foo", "bar"))
		value, err := e.GetValueGRPC(context.Background(), "foo")
		assert.NoError(t, err)
		assert.Equal(t, "bar", value)
	})
}

func expectErrorContains(t *testing.T, err error, contains string) {
	if contains == "" {
		assert.NoError(t, err)
	} else {
		require.Error(t, err)
		assert.Contains(t, err.Error(), contains)
	}
}

func TestYARPCWellKnownError(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeFailedPrecondition, "bar 1"), err)
	})
}

func TestYARPCNamedError(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"), err)
	})
}

func TestYARPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""), err)
	})
}

func TestYARPCErrorWithDetails(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(protobuf.NewError(yarpcerrors.CodeNotFound, "hello world", protobuf.WithErrorDetails(&examplepb.SetValueResponse{})))
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		assert.Equal(t, protobuf.NewError(yarpcerrors.CodeNotFound, "hello world", protobuf.WithErrorDetails(&examplepb.SetValueResponse{})), err)
	})
}

func TestGRPCWellKnownError(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(status.Error(codes.FailedPrecondition, "bar 1"))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.FailedPrecondition, "bar 1"), err)
	})
}

func TestGRPCNamedError(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", "baz 1"))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.Unknown, "bar: baz 1"), err)
	})
}

func TestGRPCNamedErrorNoMessage(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(intyarpcerrors.NewWithNamef(yarpcerrors.CodeUnknown, "bar", ""))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		assert.Equal(t, status.Error(codes.Unknown, "bar"), err)
	})
}

func TestGRPCErrorWithDetails(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		e.KeyValueYARPCServer.SetNextError(protobuf.NewError(yarpcerrors.CodeNotFound, "hello world", protobuf.WithErrorDetails(&examplepb.SetValueResponse{})))
		err := e.SetValueGRPC(context.Background(), "foo", "bar")
		st := gogostatus.Convert(err)
		assert.Equal(t, st.Code(), codes.NotFound)
		assert.Equal(t, st.Message(), "hello world")
		assert.Equal(t, st.Details(), []interface{}{&examplepb.SetValueResponse{}})
	})
}

func TestYARPCResponseAndError(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
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
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
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
	value := strings.Repeat("a", defaultServerMaxRecvMsgSize+1)
	t.Run("too big", func(t *testing.T) {
		te := testEnvOptions{}
		te.do(t, func(t *testing.T, e *testEnv) {
			assert.Equal(t, yarpcerrors.CodeResourceExhausted, yarpcerrors.FromError(e.SetValueYARPC(context.Background(), "foo", value)).Code())
		})
	})
	t.Run("just right", func(t *testing.T) {
		te := testEnvOptions{
			TransportOptions: []TransportOption{
				ClientMaxRecvMsgSize(math.MaxInt32),
				ClientMaxSendMsgSize(math.MaxInt32),
				ServerMaxRecvMsgSize(math.MaxInt32),
				ServerMaxSendMsgSize(math.MaxInt32),
			},
		}
		te.do(t, func(t *testing.T, e *testEnv) {
			if assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", value)) {
				getValue, err := e.GetValueYARPC(context.Background(), "foo")
				assert.NoError(t, err)
				assert.Equal(t, value, getValue)
			}
		})
	})
}

func TestLargeEcho(t *testing.T) {
	t.Parallel()
	value := strings.Repeat("a", 32768)
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
		if assert.NoError(t, e.SetValueYARPC(context.Background(), "foo", value)) {
			getValue, err := e.GetValueYARPC(context.Background(), "foo")
			assert.NoError(t, err)
			assert.Equal(t, value, getValue)
		}
	})
}

func TestApplicationErrorPropagation(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{}
	te.do(t, func(t *testing.T, e *testEnv) {
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

func TestCustomContextDial(t *testing.T) {
	t.Parallel()
	errMsg := "my custom dialer error"
	contextDial := func(context.Context, string) (net.Conn, error) {
		return nil, errors.New(errMsg)
	}

	te := testEnvOptions{
		DialOptions: []DialOption{ContextDialer(contextDial)},
	}
	te.do(t, func(t *testing.T, e *testEnv) {
		err := e.SetValueYARPC(context.Background(), "foo", "bar")
		require.Error(t, err)
		assert.Contains(t, err.Error(), errMsg)
	})
}

// TestGRPCCompression aims to test the compression when both, the client and
// the server has the same compressors registered and have the same compressor
// enabled.
func TestGRPCCompression(t *testing.T) {
	t.Parallel()
	compressors := []*testCompressor{
		newCompressor("test-good"),
		newCompressor("test-fail-comp"),
		newCompressor("test-fail-decomp"),
		newCompressor("test-gzip"),
	}
	for _, comp := range compressors {
		encoding.RegisterCompressor(comp)
	}

	tests := []struct {
		testEnvOptions

		msg         string
		compressor  string
		wantErr     string
		wantMetrics []compressionMetric
	}{
		{
			msg: "no compression",
		},
		{
			// we need 2 separate processes to test the case where we have the
			// content encoding negotiation when the client side sends the
			// compression request, which the server does not support
			msg:        "unknown compressor on the client-side",
			compressor: "test-unknown",
			wantErr:    "code:internal message:grpc: Compressor is not installed for requested grpc-encoding \"test-unknown\"",
		},
		{
			msg:        "fail compression",
			compressor: "test-fail-comp",
			wantErr:    "code:internal message:grpc: error while compressing: assert.AnError general error for testing",
			wantMetrics: []compressionMetric{
				{true, 0}, // failed compression of request
			},
		},
		{
			msg:        "fail decompression",
			compressor: "test-fail-decomp",
			wantErr:    "code:internal message:grpc: failed to decompress the received message assert.AnError general error for testing",
			wantMetrics: []compressionMetric{
				{true, 32777},
				{false, 0}, // failed decompression of request
			},
		},
		{
			msg:        "ok",
			compressor: "test-good",
			wantMetrics: []compressionMetric{
				{true, 32777},  // client compression of request
				{false, 32777}, // server decompression of request
				{true, 0},      // compression
				{true, 5},      // compression
				{false, 5},     // decompression
				{true, 32772},  // server compression of response
				{false, 32772}, // client decompression of response
			},
		},
		{
			msg:        "gzip",
			compressor: "test-gzip",
			wantMetrics: []compressionMetric{
				{true, 82},
				{false, 82},
				{true, 23},
				{false, 23},
				{true, 29},
				{false, 29},
				{true, 75},
				{false, 75},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			for _, comp := range compressors {
				comp.resetMetrics()
			}

			if tt.compressor != "" {
				tt.testEnvOptions.DialOptions = []DialOption{Compressor(tt.compressor)}
			}
			tt.do(t, func(t *testing.T, e *testEnv) {
				value := strings.Repeat("a", 32*1024)
				err := e.SetValueYARPC(context.Background(), "foo", value)
				if tt.wantErr != "" {
					assert.Error(t, err)
					assert.EqualError(t, err, tt.wantErr)
				} else if assert.NoError(t, err) {
					getValue, err := e.GetValueYARPC(context.Background(), "foo")
					assert.NoError(t, err)
					assert.Equal(t, value, getValue)
				}
			})

			for _, comp := range compressors {
				if comp.Name() == tt.compressor {
					assert.Equal(t, tt.wantMetrics, comp.metrics)
				} else {
					assert.Empty(t, comp.metrics, "compressor %s was called whilst not expected", comp.Name())
				}
			}
		})
	}
}

type testCompressor struct {
	name       string
	metrics    []compressionMetric
	comperr    error
	decomperr  error
	enableGZip bool
}

func newCompressor(name string) *testCompressor {
	comp := testCompressor{
		name: name,
	}

	comp.resetMetrics()
	switch strings.TrimPrefix(name, "test-") {
	case "fail-comp":
		comp.comperr = assert.AnError
	case "fail-decomp":
		comp.decomperr = assert.AnError
	case "gzip":
		comp.enableGZip = true
	}
	return &comp
}

type compressionMetric struct {
	isComp bool
	bytes  int
}

func (c *testCompressor) resetMetrics() {
	c.metrics = make([]compressionMetric, 0)
}

// newMetrics creates a new metrics data point and passes returns it as one element slice
func (c *testCompressor) newMetrics(isCompression bool) []compressionMetric {
	l := len(c.metrics)
	c.metrics = append(c.metrics, compressionMetric{
		isComp: isCompression,
		bytes:  0,
	})
	return c.metrics[l : l+1]
}

func (c *testCompressor) Name() string { return c.name }

func (c *testCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	var wc io.WriteCloser = &byteMeter{
		Writer:  w,
		metrics: c.newMetrics(true),
	}

	if c.enableGZip {
		wc = gzip.NewWriter(wc)
	}

	return wc, c.comperr
}

// Decompress maybe should return io.ReadCloser? because it is rather weird why you have this.
func (c *testCompressor) Decompress(r io.Reader) (io.Reader, error) {
	r = &byteMeter{
		Reader:  r,
		metrics: c.newMetrics(false),
	}

	if c.enableGZip {
		return gzip.NewReader(r)
	}

	return r, c.decomperr
}

type byteMeter struct {
	io.Writer
	io.Reader
	metrics []compressionMetric
}

func (w *byteMeter) Write(p []byte) (int, error) {
	w.metrics[0].bytes += len(p)
	return w.Writer.Write(p)
}

func (r *byteMeter) Read(p []byte) (int, error) {
	l, err := r.Reader.Read(p)
	r.metrics[0].bytes += l
	return l, err
}

func (w *byteMeter) Close() error { return nil }

type testEnv struct {
	Caller              string
	Service             string
	Transport           *Transport
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

type testEnvOptions struct {
	TransportOptions []TransportOption
	InboundOptions   []InboundOption
	OutboundOptions  []OutboundOption
	DialOptions      []DialOption
}

func (te *testEnvOptions) do(t *testing.T, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv(
		t,
		te.TransportOptions,
		te.InboundOptions,
		te.OutboundOptions,
		te.DialOptions,
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

func newTestEnv(
	t *testing.T,
	transportOptions []TransportOption,
	inboundOptions []InboundOption,
	outboundOptions []OutboundOption,
	dialOptions []DialOption,
) (_ *testEnv, err error) {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)
	testRouter := newTestRouter(procedures)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	logger := zaptest.NewLogger(t)
	transportOptions = append(transportOptions, Logger(logger))
	trans := NewTransport(transportOptions...)
	inbound := trans.NewInbound(listener, inboundOptions...)
	inbound.SetRouter(testRouter)
	chooser := peer.NewSingle(hostport.Identify(listener.Addr().String()), trans.NewDialer(dialOptions...))
	outbound := trans.NewOutbound(chooser, outboundOptions...)

	if err := trans.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, trans.Stop())
		}
	}()

	if err := inbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, inbound.Stop())
		}
	}()

	if err := outbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, outbound.Stop())
		}
	}()

	var clientConn *grpc.ClientConn

	clientConn, err = grpc.Dial(listener.Addr().String(), newDialOptions(dialOptions).grpcOptions()...)
	if err != nil {
		return nil, err
	}
	keyValueClient := examplepb.NewKeyValueClient(clientConn)

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
		Caller:              caller,
		Service:             service,
		Transport:           trans,
		Inbound:             inbound,
		Outbound:            outbound,
		ClientConn:          clientConn,
		ContextWrapper:      contextWrapper,
		ClientConfig:        clientConfig,
		Procedures:          procedures,
		KeyValueGRPCClient:  keyValueClient,
		KeyValueYARPCClient: keyValueYARPCClient,
		KeyValueYARPCServer: keyValueYARPCServer,
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
		e.Transport.Stop(),
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
	now := time.Now()

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
			NotBefore: now,
			NotAfter:  now.Add(10 * time.Minute),
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
			NotAfter:     now.Add(serverValidity),
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
			NotAfter:     now.Add(clientValidity),
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
