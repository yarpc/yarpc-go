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

package grpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gogostatus "github.com/gogo/status"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
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
	"go.uber.org/yarpc/transport/internal/tls/testscenario"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestYARPCBasic(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{
		TransportOptions: []TransportOption{
			Tracer(opentracing.NoopTracer{}),
		},
	}
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
		require.Len(t, protobuf.GetErrorDetails(err), 1)
		assert.Equal(t, protobuf.GetErrorDetails(err)[0], &examplepb.SetValueResponse{})
		assert.Equal(t, yarpcerrors.FromError(err).Code(), yarpcerrors.CodeNotFound)
		assert.Equal(t, yarpcerrors.FromError(err).Message(), "hello world")
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
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second*5)
			defer cancel()

			err := e.SetValueYARPC(ctx, "foo", value)

			assert.Equal(t, yarpcerrors.CodeResourceExhausted.String(), yarpcerrors.FromError(err).Code().String())
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
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second*5)
			defer cancel()

			if assert.NoError(t, e.SetValueYARPC(ctx, "foo", value)) {
				getValue, err := e.GetValueYARPC(ctx, "foo")
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
	tagsCompression := map[string]string{"stage": "compress"}
	tagsDecompression := map[string]string{"stage": "decompress"}

	tests := []struct {
		testEnvOptions

		msg         string
		compressor  transport.Compressor
		wantErr     string
		wantMetrics []metric
	}{
		{
			msg: "no compression",
		},
		{
			msg:        "fail compression of request",
			compressor: _badCompressor,
			wantErr:    "code:internal message:grpc: error while compressing: assert.AnError general error for testing",
			wantMetrics: []metric{
				{0, tagsCompression},
			},
		},
		{
			msg:        "fail decompression of request",
			compressor: _badDecompressor,
			wantErr:    "code:internal message:grpc: failed to decompress the received message: assert.AnError general error for testing",
			wantMetrics: []metric{
				{32777, tagsCompression},
				{0, tagsDecompression},
			},
		},
		{
			msg:        "ok, dummy compression",
			compressor: _goodCompressor,
			wantMetrics: []metric{
				{32777, tagsCompression},
				{32777, tagsDecompression},
				{0, tagsCompression},
				{5, tagsCompression},
				{5, tagsDecompression},
				{32772, tagsCompression},
				{32772, tagsDecompression},
			},
		},
		{
			msg:        "ok, gzip compression",
			compressor: _gzipCompressor,
			wantMetrics: []metric{
				{82, tagsCompression},
				{82, tagsDecompression},
				{23, tagsCompression},
				{23, tagsDecompression},
				{29, tagsCompression},
				{29, tagsDecompression},
				{75, tagsCompression},
				{75, tagsDecompression},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			_metrics.reset()

			tt.testEnvOptions.DialOptions = []DialOption{Compressor(tt.compressor)}
			tt.do(t, func(t *testing.T, e *testEnv) {
				value := strings.Repeat("a", 32*1024)
				err := e.SetValueYARPC(context.Background(), "foo", value)
				if tt.wantErr != "" {
					assert.Error(t, err)
					assert.EqualError(t, err, tt.wantErr)
				} else if assert.NoError(t, err) {
					getValue, err := e.GetValueYARPC(context.Background(), "foo")
					require.NoError(t, err)
					assert.Equal(t, value, getValue)
				}
			})

			compressor := ""
			if tt.compressor != nil {
				compressor = tt.compressor.Name()
			}
			assert.Equal(t, newMetrics(tt.wantMetrics, map[string]string{
				"compressor": compressor,
			}), _metrics)
		})
	}
}

func TestTLSWithYARPCAndGRPC(t *testing.T) {
	tests := []struct {
		name           string
		clientValidity time.Duration
		serverValidity time.Duration
		wantErr        bool
	}{
		{
			name:           "valid certs both sides",
			clientValidity: time.Minute,
			serverValidity: time.Minute,
		},
		{
			name:           "invalid server cert",
			clientValidity: time.Minute,
			serverValidity: -1,
			wantErr:        true,
		},
		{
			name:           "invalid client cert",
			clientValidity: -1,
			serverValidity: time.Minute,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := testscenario.Create(t, tt.clientValidity, tt.serverValidity)
			te := testEnvOptions{
				InboundOptions: []InboundOption{InboundCredentials(credentials.NewTLS(scenario.ServerTLSConfig()))},
				DialOptions:    []DialOption{DialerCredentials(credentials.NewTLS(scenario.ClientTLSConfig()))},
			}
			te.do(t, func(t *testing.T, e *testEnv) {
				err := e.SetValueYARPC(context.Background(), "foo", "bar")
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				err = e.SetValueGRPC(context.Background(), "foo", "bar")
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

// TestCompressionWithMultipleOutbounds creates multiple outbound for the
// same hostport where one outbound has compression enabled.
// Validates compression is applied for the outbound with compression enabled
// and rest of the outbounds are still uncompressed.
func TestCompressionWithMultipleOutbounds(t *testing.T) {
	env, err := newTestEnv(t, nil, nil, nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, env.Close()) }()

	chooser := peer.NewSingle(hostport.Identify(env.Inbound.Addr().String()), env.Transport.NewDialer())
	compressedOutbound := env.Transport.NewOutbound(chooser, OutboundCompressor(_goodCompressor))
	require.NoError(t, compressedOutbound.Start())
	defer compressedOutbound.Stop()

	caller := "example-client"
	service := "example"
	clientConfig := clientconfig.MultiOutbound(
		caller,
		service,
		transport.Outbounds{
			ServiceName: caller,
			Unary:       compressedOutbound,
		},
	)
	compressedClient := examplepb.NewKeyValueYARPCClient(clientConfig)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second*5)
	defer cancel()

	// Send request over uncompressed outbound and assert compression metric
	// is empty.
	_metrics.reset()
	require.NoError(t, env.SetValueYARPC(ctx, "foo", strings.Repeat("a", 32*1024)))
	assert.Equal(t, &metricCollection{metrics: []metric{}}, _metrics)

	// Send request over compressed outbound and assert compression metric
	// is seen.
	_metrics.reset()
	_, err = compressedClient.SetValue(ctx, &examplepb.SetValueRequest{Key: "foo", Value: strings.Repeat("a", 32*1024)})
	require.NoError(t, err)
	wantMetric := []metric{
		{32777, map[string]string{"stage": "compress"}},
		{32777, map[string]string{"stage": "decompress"}},
		{0, map[string]string{"stage": "compress"}},
	}
	assert.Equal(t, newMetrics(wantMetric, map[string]string{
		"compressor": _goodCompressor.name,
	}), _metrics)
}

func TestGRPCHeaderListSize(t *testing.T) {
	tests := []struct {
		desc       string
		options    []TransportOption
		headerSize int
		errorMsg   string
	}{
		{
			desc:       "default_setting",
			headerSize: 1024,
		},
		{
			desc:       "limit_server_header_size",
			headerSize: 1024,
			options:    []TransportOption{ServerMaxHeaderListSize(1000)},
			errorMsg:   "header list size to send violates the maximum size (1000 bytes) set by server",
		},
		{
			desc:       "limit_client_header_size",
			headerSize: 1024,
			options:    []TransportOption{ClientMaxHeaderListSize(1000)},
			errorMsg:   "stream terminated",
		},
		{
			desc:       "allow_large_header_size",
			headerSize: 1024 * 1024 * 1, // 1MB
			options:    []TransportOption{ServerMaxHeaderListSize(1024 * 1024 * 2), ClientMaxHeaderListSize(1024 * 1024 * 2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			headerVal := make([]byte, tt.headerSize)
			// Set valid ASCII as grpc header cannot be a 0 byte slice.
			for i := 0; i < tt.headerSize; i++ {
				headerVal[i] = 'a'
			}
			te := testEnvOptions{
				TransportOptions: tt.options,
			}
			te.do(t, func(t *testing.T, e *testEnv) {
				var resHeaders map[string]string
				// Setting longer timeout as CI timesout on large payloads.
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
				defer cancel()

				err := e.SetValueYARPC(ctx, "foo", "bar", yarpc.ResponseHeaders(&resHeaders), yarpc.WithHeader("test-header", string(headerVal)))
				if tt.errorMsg != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errorMsg)
					return
				}
				assert.NoError(t, err)
				assert.Equal(t, resHeaders["test-header"], string(headerVal))
			})
		})
	}
}

func TestMuxTLS(t *testing.T) {
	defer goleak.VerifyNone(t)
	tests := []struct {
		name        string
		isClientTLS bool
	}{
		{
			name:        "plaintext_client",
			isClientTLS: false,
		},
		{
			name:        "tls_client",
			isClientTLS: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := testscenario.Create(t, time.Minute, time.Minute)
			var dialOptions []DialOption
			if tt.isClientTLS {
				dialOptions = append(dialOptions, DialerCredentials(credentials.NewTLS(scenario.ClientTLSConfig())))
			}

			te := testEnvOptions{
				InboundOptions: []InboundOption{InboundTLSConfiguration(scenario.ServerTLSConfig()), InboundTLSMode(yarpctls.Permissive)},
				DialOptions:    dialOptions,
			}
			te.do(t, func(t *testing.T, e *testEnv) {
				err := e.SetValueYARPC(context.Background(), "foo", "bar")
				assert.NoError(t, err)

				err = e.SetValueGRPC(context.Background(), "foo", "bar")
				assert.NoError(t, err)
			})
		})
	}
}

func TestOutboundTLS(t *testing.T) {
	defer goleak.VerifyNone(t)
	scenario := testscenario.Create(t, time.Minute, time.Minute)

	tests := []struct {
		desc             string
		withCustomDialer bool
	}{
		{desc: "without_custom_dialer", withCustomDialer: false},
		{desc: "with_custom_dialer", withCustomDialer: true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dialOpts := []DialOption{
				DialerTLSConfig(scenario.ClientTLSConfig()),
			}
			// This is used for asserting if custom dialer is invoked.
			var invokedCustomDialer int32
			if tt.withCustomDialer {
				dialOpts = append(dialOpts, ContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
					// Avoid write race warning as concurrent dialers will be
					// invoked as two gRPC clients are created below.
					atomic.AddInt32(&invokedCustomDialer, 1)
					return (&net.Dialer{}).DialContext(ctx, "tcp", s)
				}))
			}
			te := testEnvOptions{
				InboundOptions: []InboundOption{InboundTLSConfiguration(scenario.ServerTLSConfig()), InboundTLSMode(yarpctls.Permissive)},
				DialOptions:    dialOpts,
			}
			te.do(t, func(t *testing.T, e *testEnv) {
				err := e.SetValueYARPC(context.Background(), "foo", "bar")
				assert.NoError(t, err)

				err = e.SetValueGRPC(context.Background(), "foo", "bar")
				assert.NoError(t, err)
			})
			if tt.withCustomDialer {
				assert.True(t, invokedCustomDialer > 0)
			}
		})
	}
}

type metricCollection struct {
	metrics []metric
}

func (c *metricCollection) reset() {
	c.metrics = c.metrics[:0]
}

func newMetrics(metrics []metric, tags map[string]string) *metricCollection {
	c := metricCollection{
		metrics: make([]metric, len(metrics)),
	}
	for i, m := range metrics {
		c.metrics[i] = metric{
			bytes: m.bytes,
			tags:  map[string]string{},
		}
		for key, value := range m.tags {
			c.metrics[i].tags[key] = value
		}
		for key, value := range tags {
			c.metrics[i].tags[key] = value
		}
	}
	return &c
}

type metric struct {
	bytes int
	tags  map[string]string
}

func (m *metric) Increment(value int) {
	m.bytes += value
}

// new creates a new metrics data point and passes returns it as one element slice
func (c *metricCollection) new(stage, compressor string) *metric {
	l := len(c.metrics)
	c.metrics = append(c.metrics, metric{
		bytes: 0,
		tags: map[string]string{
			"compressor": compressor,
			"stage":      stage,
		},
	})
	return &c.metrics[l]
}

type counter interface {
	Increment(value int)
}

type testCompressor struct {
	name       string
	metrics    *metricCollection
	comperr    error
	decomperr  error
	enableGZip bool
}

type testCompressorBehavior int

const (
	testCompressorOk = 1 << iota
	testCompressorFailToCompress
	testCompressorFailToDecompress
	testCompressorGzip
)

func newCompressor(name string, behavior testCompressorBehavior, metrics *metricCollection) *testCompressor {
	comp := testCompressor{
		name:    name,
		metrics: metrics,
	}

	if behavior&testCompressorFailToCompress != 0 {
		comp.comperr = assert.AnError
	}

	if behavior&testCompressorFailToDecompress != 0 {
		comp.decomperr = assert.AnError
	}

	if behavior&testCompressorGzip != 0 {
		comp.enableGZip = true
	}

	return &comp
}

func (c *testCompressor) Name() string { return c.name }

func (c *testCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	metered := byteMeter{
		Writer:  w,
		counter: c.metrics.new("compress", c.name),
	}

	if c.enableGZip {
		return gzip.NewWriter(&metered), nil
	}
	return &metered, c.comperr
}

func (c *testCompressor) Decompress(r io.Reader) (io.ReadCloser, error) {
	metered := byteMeter{
		Reader:  r,
		counter: c.metrics.new("decompress", c.name),
	}

	if c.enableGZip {
		return gzip.NewReader(&metered)
	}

	return &metered, c.decomperr
}

// byteMeter is a test type wrapper that counts the number of bytes transferred within the compressors.
type byteMeter struct {
	io.Writer
	io.Reader
	counter counter
}

func (m *byteMeter) Write(p []byte) (int, error) {
	m.counter.Increment(len(p))
	return m.Writer.Write(p)
}

func (m *byteMeter) Read(p []byte) (int, error) {
	l, err := m.Reader.Read(p)
	m.counter.Increment(l)
	return l, err
}

func (m *byteMeter) Close() error { return nil }

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

	clientConn, err = grpc.Dial(listener.Addr().String(), newDialOptions(dialOptions).grpcOptions(trans)...)
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

func (e *testEnv) GetValueYARPC(ctx context.Context, key string, options ...yarpc.CallOption) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)
	defer cancel()
	response, err := e.KeyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{Key: key}, options...)
	if response != nil {
		return response.Value, err
	}
	return "", err
}

func (e *testEnv) SetValueYARPC(ctx context.Context, key string, value string, options ...yarpc.CallOption) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, testtime.Second)
		defer cancel()
	}
	_, err := e.KeyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: value}, options...)
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
	return transport.HandlerSpec{}, yarpcerrors.UnimplementedErrorf("no procedure for name %s", request.Procedure)
}

func TestYARPCErrorsConverted(t *testing.T) {
	// Ensures that all returned errors are gRPC errors and not YARPC errors

	trans := NewTransport()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	inbound := trans.NewInbound(listener)

	outbound := trans.NewSingleOutbound(listener.Addr().String())

	router := &testRouter{}
	inbound.SetRouter(router)

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()

	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()

	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()

	t.Run("no procedure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := outbound.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  "encoding",
			Procedure: "no procedure",
			Body:      strings.NewReader("foo-body"),
		})

		require.Error(t, err)
		assert.True(t, yarpcerrors.IsUnimplemented(err))
	})
}
