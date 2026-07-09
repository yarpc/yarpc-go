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

package grpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
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
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	yarpcpeer "go.uber.org/yarpc/api/peer"
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
	"go.uber.org/yarpc/peer/roundrobin"
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
			// The value is ~64 MB; allow extra headroom under race detector and parallel load.
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second*30)
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
	//lint:ignore SA1019 grpc.Dial is deprecated
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

// TestPeerChurnCallsSucceedAfterChurn verifies that after a full peer churn
// cycle (retain → release → re-retain the same address), YARPC calls to the
// downstream service succeed without errors. This is the integration-level
// reproduction of the yarpc-go v1.88.6 incident: same-address peer churn must
// not break the outbound or produce error-level logs.
//
// Peer churn is simulated via roundrobin.List.Update(), which mirrors real
// service discovery: the server is removed from the peer list (transport
// releases the peer, subscriber count → 0) then re-added (peer re-created
// for the same address).
func TestPeerChurnCallsSucceedAfterChurn(t *testing.T) {
	t.Parallel()
	root := metrics.New()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans := NewTransport(Logger(zaptest.NewLogger(t)), Meter(root.Scope()), WithDynamicConnectionScaling(true))
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	serverID := hostport.Identify(listener.Addr().String())
	list := roundrobin.New(trans.NewDialer())
	outbound := trans.NewOutbound(list)

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()

	// Peer is initially present in the peer list.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	clientConfig := clientconfig.MultiOutbound("example-client", "example",
		transport.Outbounds{ServiceName: "example-client", Unary: outbound},
	)
	client := examplepb.NewKeyValueYARPCClient(clientConfig)

	set := func(key, val string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		_, err := client.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: val})
		require.NoError(t, err)
	}
	get := func(key string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		resp, err := client.GetValue(ctx, &examplepb.GetValueRequest{Key: key})
		require.NoError(t, err)
		return resp.Value
	}

	// Baseline: calls succeed before any churn.
	set("key1", "value1")
	assert.Equal(t, "value1", get("key1"))

	// Simulate peer churn: remove the peer (transport releases it, subscriber
	// count drops to 0) then immediately re-add it (peer re-created for the
	// same address). With transport-level metric registration, re-creation
	// cannot produce duplicate registration errors.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	// Calls succeed after churn; data written before churn is preserved.
	set("key2", "value2")
	assert.Equal(t, "value2", get("key2"))
	assert.Equal(t, "value1", get("key1"))

	// Second churn cycle confirms idempotence.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))
	set("key3", "value3")
	assert.Equal(t, "value3", get("key3"))
}

// TestMultiOutboundSamePeerCalls verifies a multi-outbound topology where two
// outbounds route to the same downstream address. Both outbounds must be able
// to make successful YARPC calls independently, and the underlying transport
// must share a single grpcPeer for the address (not create two separate peers
// to the same host).
func TestMultiOutboundSamePeerCalls(t *testing.T) {
	t.Parallel()
	root := metrics.New()

	env, err := newTestEnv(t,
		[]TransportOption{Meter(root.Scope()), WithDynamicConnectionScaling(true)},
		nil, nil, nil,
	)
	require.NoError(t, err)
	defer func() { assert.NoError(t, env.Close()) }()

	serverAddr := env.Inbound.Addr().String()

	// Second outbound to the same address, sharing env.Transport.
	chooser2 := peer.NewSingle(hostport.Identify(serverAddr), env.Transport.NewDialer())
	outbound2 := env.Transport.NewOutbound(chooser2)
	require.NoError(t, outbound2.Start())
	defer func() { assert.NoError(t, outbound2.Stop()) }()

	client2 := examplepb.NewKeyValueYARPCClient(
		clientconfig.MultiOutbound("example-client", "example",
			transport.Outbounds{ServiceName: "example-client", Unary: outbound2},
		),
	)

	ctx := context.Background()

	// Both outbounds write and read successfully.
	require.NoError(t, env.SetValueYARPC(ctx, "from-outbound1", "value1"))

	ctx2, cancel2 := context.WithTimeout(ctx, testtime.Second)
	defer cancel2()
	_, err = client2.SetValue(ctx2, &examplepb.SetValueRequest{Key: "from-outbound2", Value: "value2"})
	require.NoError(t, err)

	// Cross-read: data written via outbound1 is visible via outbound2 (same server).
	ctx3, cancel3 := context.WithTimeout(ctx, testtime.Second)
	defer cancel3()
	resp, err := client2.GetValue(ctx3, &examplepb.GetValueRequest{Key: "from-outbound1"})
	require.NoError(t, err)
	assert.Equal(t, "value1", resp.Value)

	got, err := env.GetValueYARPC(ctx, "from-outbound2")
	require.NoError(t, err)
	assert.Equal(t, "value2", got)

	// Both outbounds share a single grpcPeer: only one entry in addressToPeer.
	env.Transport.lock.Lock()
	peerCount := len(env.Transport.addressToPeer)
	env.Transport.lock.Unlock()
	assert.Equal(t, 1, peerCount,
		"two outbounds to the same address must share one grpcPeer, got %d peers", peerCount)
}

// TestMultiOutboundPeerChurnRemainingOutboundStaysHealthy verifies that in a
// multi-outbound topology, removing the peer from one peer list (releasing one
// subscriber) leaves the peer alive while the other outbound still holds it —
// and after all subscribers release the peer and then re-add it (full churn),
// calls through both outbounds succeed.
func TestMultiOutboundPeerChurnRemainingOutboundStaysHealthy(t *testing.T) {
	t.Parallel()
	root := metrics.New()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans := NewTransport(Logger(zaptest.NewLogger(t)), Meter(root.Scope()), WithDynamicConnectionScaling(true))
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()

	serverAddr := listener.Addr().String()
	serverID := hostport.Identify(serverAddr)

	// Two independent peer lists, each holding the same server address.
	list1 := roundrobin.New(trans.NewDialer())
	list2 := roundrobin.New(trans.NewDialer())
	ob1 := trans.NewOutbound(list1)
	ob2 := trans.NewOutbound(list2)

	require.NoError(t, ob1.Start())
	defer func() { assert.NoError(t, ob1.Stop()) }()
	require.NoError(t, ob2.Start())
	defer func() { assert.NoError(t, ob2.Stop()) }()

	require.NoError(t, list1.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, list2.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	makeClient := func(ob *Outbound) examplepb.KeyValueYARPCClient {
		return examplepb.NewKeyValueYARPCClient(
			clientconfig.MultiOutbound("example-client", "example",
				transport.Outbounds{ServiceName: "example-client", Unary: ob},
			),
		)
	}
	client1 := makeClient(ob1)
	client2 := makeClient(ob2)

	call := func(client examplepb.KeyValueYARPCClient, key, val string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		_, err := client.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: val})
		require.NoError(t, err)
	}

	// Baseline: both clients work.
	call(client1, "k1", "v1")
	call(client2, "k2", "v2")

	// Remove peer from list2 only (subscriber count drops from 2 → 1).
	// The peer must survive because list1 still holds a subscriber.
	require.NoError(t, list2.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))

	trans.lock.Lock()
	_, peerAlive := trans.addressToPeer[serverAddr]
	trans.lock.Unlock()
	assert.True(t, peerAlive, "peer must remain when one of two subscribers releases it")

	// Client1 (list1) still works while client2 (list2) has no peer.
	call(client1, "k3", "v3")

	// Remove from list1 too — peer fully released, deleted from transport map.
	require.NoError(t, list1.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))

	trans.lock.Lock()
	_, peerGone := trans.addressToPeer[serverAddr]
	trans.lock.Unlock()
	assert.False(t, peerGone, "peer must be removed when all subscribers release it")

	// Re-add to both lists — peer is re-created for same address (full churn).
	// With transport-level metrics, no duplicate registration errors occur.
	require.NoError(t, list1.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, list2.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	// Both clients work again after churn.
	call(client1, "k4", "v4")
	call(client2, "k5", "v5")
}

// TestPeerChurnInFlightRequests verifies that requests already in flight when
// peer churn fires are not silently dropped or hung. In-flight requests on the
// connection that is being torn down will receive an error; the test asserts no
// deadlocks or panics, and that new requests succeed once the peer is re-added.
func TestPeerChurnInFlightRequests(t *testing.T) {
	t.Parallel()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans := NewTransport(Logger(zaptest.NewLogger(t)), WithDynamicConnectionScaling(true))
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	serverID := hostport.Identify(listener.Addr().String())
	list := roundrobin.New(trans.NewDialer())
	outbound := trans.NewOutbound(list)

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	client := examplepb.NewKeyValueYARPCClient(
		clientconfig.MultiOutbound("example-client", "example",
			transport.Outbounds{ServiceName: "example-client", Unary: outbound},
		),
	)

	// Seed a value so gets have something to return.
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err = client.SetValue(ctx, &examplepb.SetValueRequest{Key: "k", Value: "v"})
	require.NoError(t, err)

	// Launch concurrent requests and churn the peer simultaneously.
	const workers = 10
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()
			_, err := client.GetValue(ctx, &examplepb.GetValueRequest{Key: "k"})
			errCh <- err
		}()
	}

	// Churn while workers are in flight.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	// Drain all worker results. Each either succeeded or got a transport/context
	// error — neither outcome is a panic or deadlock.
	for i := 0; i < workers; i++ {
		<-errCh // just drain; some may error during churn, that is acceptable
	}

	// After churn settles, new requests must succeed.
	ctx2, cancel2 := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel2()
	resp, err := client.GetValue(ctx2, &examplepb.GetValueRequest{Key: "k"})
	require.NoError(t, err)
	assert.Equal(t, "v", resp.Value)
}

// TestPeerChurnConcurrentCallsNoRace verifies there are no data races when
// multiple goroutines make YARPC calls concurrently while another goroutine
// continuously churns the peer list. Run with -race.
func TestPeerChurnConcurrentCallsNoRace(t *testing.T) {
	t.Parallel()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans := NewTransport(Logger(zaptest.NewLogger(t)), WithDynamicConnectionScaling(true))
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	serverID := hostport.Identify(listener.Addr().String())
	list := roundrobin.New(trans.NewDialer())
	outbound := trans.NewOutbound(list)

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	client := examplepb.NewKeyValueYARPCClient(
		clientconfig.MultiOutbound("example-client", "example",
			transport.Outbounds{ServiceName: "example-client", Unary: outbound},
		),
	)

	const (
		callers     = 5
		churnCycles = 3
	)

	done := make(chan struct{})

	// Churn goroutine: remove and re-add the peer repeatedly.
	go func() {
		defer close(done)
		for i := 0; i < churnCycles; i++ {
			_ = list.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}})
			_ = list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}})
		}
	}()

	// Caller goroutines: make calls concurrently with churn.
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
				key := fmt.Sprintf("k%d-%d", i, j)
				_, _ = client.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: "v"})
				cancel()
			}
		}(i)
	}

	<-done
	wg.Wait()
}

// TestPeerChurnGaugesZeroAfterPeerRemoval verifies that transport-level
// connection pool gauges (active, draining, idle connections) return to zero
// after a peer is removed via churn. This ensures PR #2511's aggregate gauge
// approach leaves no stale residue when a peer is torn down in production.
func TestPeerChurnGaugesZeroAfterPeerRemoval(t *testing.T) {
	t.Parallel()
	root := metrics.New()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans := NewTransport(
		Logger(zaptest.NewLogger(t)),
		Meter(root.Scope()),
		WithDynamicConnectionScaling(true),
		MinConnections(1),
	)
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	serverID := hostport.Identify(listener.Addr().String())
	list := roundrobin.New(trans.NewDialer())
	outbound := trans.NewOutbound(list)

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	client := examplepb.NewKeyValueYARPCClient(
		clientconfig.MultiOutbound("example-client", "example",
			transport.Outbounds{ServiceName: "example-client", Unary: outbound},
		),
	)

	// Make a call to ensure the peer is connected and gauges are populated.
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err = client.SetValue(ctx, &examplepb.SetValueRequest{Key: "k", Value: "v"})
	require.NoError(t, err)

	// Active connections gauge must be non-zero (at least 1 connection up).
	gauges, _ := poolMetricSnapshot(root)
	assert.Greater(t, gauges["conn_pool_active_connections"], int64(0),
		"active connections must be non-zero while peer is retained")

	// Remove peer (subscriber count → 0, peer stopped). The peer's contribution
	// to the aggregate gauges must drain to zero.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))

	// Wait for the async cleanup goroutine and peer stop to complete.
	require.Eventually(t, func() bool {
		gauges, _ := poolMetricSnapshot(root)
		return gauges["conn_pool_active_connections"] == 0 &&
			gauges["conn_pool_draining_connections"] == 0 &&
			gauges["conn_pool_idle_connections"] == 0
	}, 5*time.Second, 10*time.Millisecond,
		"all connection pool gauges must return to zero after peer removal")

	// Re-add peer — gauges must recover.
	require.NoError(t, list.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))

	ctx2, cancel2 := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel2()
	_, err = client.GetValue(ctx2, &examplepb.GetValueRequest{Key: "k"})
	require.NoError(t, err)

	gauges, _ = poolMetricSnapshot(root)
	assert.Greater(t, gauges["conn_pool_active_connections"], int64(0),
		"active connections must recover after peer is re-added")
}

// TestDispatcherRestartSameTransport simulates the production scenario where a
// service's YARPC dispatcher is torn down and a new one is brought up using the
// same underlying transport (same process, same metric registry). The new
// outbound re-retains the same peer addresses that the old one held. With
// transport-level metric registration this must produce no errors.
//
// This covers the uf-load / adaptive-authn-gateway staging failure mode: the
// service's fx app was rebuilt (e.g. during a BITS deploy) with the same metric
// scope, causing the old per-peer metric registration (pre-PR #2511) to fail on
// the second RetainPeer for each address.
func TestDispatcherRestartSameTransport(t *testing.T) {
	t.Parallel()
	root := metrics.New()

	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	procedures := examplepb.BuildKeyValueYARPCProcedures(keyValueYARPCServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Transport lives for the entire test — same object, same metric registry.
	trans := NewTransport(Logger(zaptest.NewLogger(t)), Meter(root.Scope()), WithDynamicConnectionScaling(true))
	inbound := trans.NewInbound(listener)
	inbound.SetRouter(newTestRouter(procedures))

	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()
	require.NoError(t, inbound.Start())
	defer func() { assert.NoError(t, inbound.Stop()) }()

	serverID := hostport.Identify(listener.Addr().String())

	makeClientViaOutbound := func() (examplepb.KeyValueYARPCClient, *roundrobin.List, *Outbound) {
		l := roundrobin.New(trans.NewDialer())
		ob := trans.NewOutbound(l)
		require.NoError(t, ob.Start())
		require.NoError(t, l.Update(yarpcpeer.ListUpdates{Additions: []yarpcpeer.Identifier{serverID}}))
		cc := clientconfig.MultiOutbound("example-client", "example",
			transport.Outbounds{ServiceName: "example-client", Unary: ob},
		)
		return examplepb.NewKeyValueYARPCClient(cc), l, ob
	}

	// First "dispatcher" instance: start, make calls, stop.
	client1, list1, ob1 := makeClientViaOutbound()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err = client1.SetValue(ctx, &examplepb.SetValueRequest{Key: "k", Value: "v"})
	require.NoError(t, err)

	// Tear down the first dispatcher: release all peers from the transport.
	require.NoError(t, list1.Update(yarpcpeer.ListUpdates{Removals: []yarpcpeer.Identifier{serverID}}))
	require.NoError(t, ob1.Stop())

	// Second "dispatcher" instance: same transport re-retains the same addresses.
	// With old per-peer metric registration this would fail (duplicate registration).
	// With transport-level registration this must succeed without any errors.
	client2, _, ob2 := makeClientViaOutbound()
	defer func() { assert.NoError(t, ob2.Stop()) }()

	ctx2, cancel2 := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel2()
	resp, err := client2.GetValue(ctx2, &examplepb.GetValueRequest{Key: "k"})
	require.NoError(t, err)
	assert.Equal(t, "v", resp.Value)
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

// --- connection pool integration tests ---

// poolMetricSnapshot reads all gauges and counters from a RootSnapshot into
// convenient maps keyed by metric name.
func poolMetricSnapshot(root *metrics.Root) (gauges, counters map[string]int64) {
	snap := root.Snapshot()
	gauges = make(map[string]int64, len(snap.Gauges))
	for _, g := range snap.Gauges {
		gauges[g.Name] = g.Value
	}
	counters = make(map[string]int64, len(snap.Counters))
	for _, c := range snap.Counters {
		counters[c.Name] = c.Value
	}
	return gauges, counters
}

// TestConnectionPoolScaleDown verifies that evaluateScaling drains a
// connection when the aggregate stream load is low enough that the pool can
// be reduced.  We call evaluateScaling() directly to bypass the 30-second
// monitor interval.
func TestConnectionPoolScaleDown(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	te := testEnvOptions{
		TransportOptions: []TransportOption{
			WithDynamicConnectionScaling(true),
			MinConnections(1),
			MaxConnections(5),
			MaxConcurrentStreams(100),
			ScaleUpThreshold(0.8), // threshold = 80
			Meter(root.Scope()),
		},
	}
	te.do(t, func(t *testing.T, e *testEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		apiPeer, onFinish, err := e.Outbound.peerChooser.Choose(ctx, &transport.Request{})
		require.NoError(t, err)
		defer onFinish(nil)
		p := apiPeer.(*grpcPeer)

		// Grow the pool to 3 connections (minConnections=1 so scale-down is allowed).
		require.NoError(t, p.addConn())
		require.NoError(t, p.addConn())

		// With 3 active connections, 0 total streams, and
		// capacityAfterDrain = threshold*(3-1) = 80*2 = 160 > 0,
		// maybeScaleDown must drain the most-loaded connection.
		p.evaluateScaling()

		gauges, counters := poolMetricSnapshot(root)
		assert.Equal(t, int64(1), counters["conn_pool_scale_down_total"],
			"scale-down counter should increment")
		assert.Equal(t, int64(2), gauges["conn_pool_active_connections"])
		assert.Equal(t, int64(1), gauges["conn_pool_draining_connections"])
	})
}

// TestConnectionPoolIdleReactivation verifies that tryScaleUp reactivates an
// idle connection instead of dialling a new one when capacity is needed.
func TestConnectionPoolIdleReactivation(t *testing.T) {
	t.Parallel()
	root := metrics.New()
	te := testEnvOptions{
		TransportOptions: []TransportOption{
			WithDynamicConnectionScaling(true),
			MinConnections(1),
			MaxConnections(3),
			MaxConcurrentStreams(2),
			ScaleUpThreshold(0.5), // threshold = 1
			Meter(root.Scope()),
		},
	}
	te.do(t, func(t *testing.T, e *testEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		apiPeer, onFinish, err := e.Outbound.peerChooser.Choose(ctx, &transport.Request{})
		require.NoError(t, err)
		defer onFinish(nil)
		p := apiPeer.(*grpcPeer)

		// Mark the initial connection as idle (live context, so reactivation is allowed).
		p.loadConns()[0].setState(connStateIdle)

		// tryScaleUp should reactivate the idle conn instead of dialling.
		overBudget := makeConn(connStateActive, 85)
		p.tryScaleUp(overBudget)

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond)

		_, counters := poolMetricSnapshot(root)
		assert.Equal(t, int64(1), counters["conn_pool_idle_reactivation_total"],
			"idle reactivation counter should increment")
		assert.Equal(t, int64(0), counters["conn_pool_scale_up_total"],
			"no new dial should happen when an idle conn is available")

		assert.Equal(t, connStateActive, p.loadConns()[0].getState(),
			"formerly idle conn should be active after reactivation")
	})
}

// TestConnectionPoolNoActiveConnectionsReturnsUnavailable verifies that when
// all pool connections are draining (non-active) and the YARPC peer is still
// considered Available by the chooser, invoke() returns UnavailableErrorf
// rather than hanging or panicking.
//
// Marking connections as connStateDraining (our internal state) does not
// cancel their contexts, so monitorConnWrapper stays blocked in
// WaitForStateChange and does not update the peer's YARPC status.  The
// chooser therefore still returns the peer, but pickConn() finds no active
// connections and the outbound returns UnavailableErrorf immediately.
func TestConnectionPoolNoActiveConnectionsReturnsUnavailable(t *testing.T) {
	t.Parallel()
	te := testEnvOptions{
		TransportOptions: []TransportOption{
			WithDynamicConnectionScaling(true),
			MinConnections(1),
			MaxConnections(3),
		},
	}
	te.do(t, func(t *testing.T, e *testEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Obtain the peer reference then immediately release the chooser hold.
		apiPeer, onFinish, err := e.Outbound.peerChooser.Choose(ctx, &transport.Request{})
		require.NoError(t, err)
		onFinish(nil)
		p := apiPeer.(*grpcPeer)

		// Mark every connection as draining without cancelling contexts.
		// monitorConnWrapper goroutines remain blocked in WaitForStateChange,
		// so the YARPC peer status stays Available — Choose will still return
		// this peer, but pickConn() will find no active connections.
		for _, c := range p.loadConns() {
			c.setState(connStateDraining)
		}

		err = e.SetValueYARPC(ctx, "foo", "bar")
		require.Error(t, err)
		assert.True(t, yarpcerrors.IsUnavailable(err),
			"expected UnavailableErrorf when all connections are draining, got: %v", err)
	})
}
