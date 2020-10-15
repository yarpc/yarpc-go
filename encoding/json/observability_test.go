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

package json_test

import (
	"context"
	"fmt"
	nethttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_serverName = "json-server"
	_clientName = "json-client"
)

func TestJsonMetrics(t *testing.T) {
	client, clientMetricsRoot, serverMetricsRoot, cleanup := initClientAndServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var resp requestResponse
	err := client.Call(ctx, "test", &requestResponse{Val: "test body"}, &resp)
	assert.NoError(t, err, "unexpected call error")

	t.Run("counters", func(t *testing.T) {
		wantCounters := []counterAssertion{
			{Name: "calls", Value: 1},
			{Name: "panics"},
			{Name: "successes", Value: 1},
		}

		assertClientAndServerMetrics(t, wantCounters, clientMetricsRoot, serverMetricsRoot)
	})
	t.Run("inbound histograms", func(t *testing.T) {
		wantHistograms := []histogramAssertion{
			{Name: "caller_failure_latency_ms"},
			{Name: "request_payload_size_bytes", Value: []int64{32}},
			{Name: "response_payload_size_bytes", Value: []int64{32}},
			{Name: "server_failure_latency_ms"},
			{Name: "success_latency_ms", Value: []int64{1}},
			{Name: "timeout_ttl_ms"},
			{Name: "ttl_ms", Value: []int64{1000}},
		}
		assertHistogram(t, wantHistograms, serverMetricsRoot.Snapshot().Histograms)
	})
}

type counterAssertion struct {
	Name  string
	Tags  map[string]string
	Value int
}

type histogramAssertion struct {
	Name  string
	Tags  map[string]string
	Value []int64
}

func assertClientAndServerMetrics(t *testing.T, counterAssertions []counterAssertion, clientSnapshot, serverSnapshot *metrics.Root) {
	t.Run("inbound", func(t *testing.T) {
		assertMetrics(t, counterAssertions, serverSnapshot.Snapshot().Counters)
	})
	t.Run("outbound", func(t *testing.T) {
		assertMetrics(t, counterAssertions, clientSnapshot.Snapshot().Counters)
	})
}

func assertMetrics(t *testing.T, counterAssertions []counterAssertion, snapshot []metrics.Snapshot) {
	require.Len(t, counterAssertions, len(snapshot), "unexpected number of counters")

	for i, wantCounter := range counterAssertions {
		require.Equal(t, wantCounter.Name, snapshot[i].Name, "unexpected counter")
		assert.EqualValues(t, wantCounter.Value, snapshot[i].Value, "unexpected counter value")
		for wantTagKey, wantTagVal := range wantCounter.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

func assertHistogram(t *testing.T, histogramAssertions []histogramAssertion, snapshot []metrics.HistogramSnapshot) {
	require.Len(t, histogramAssertions, len(snapshot), "unexpected number of histograms")

	for i, wantCounter := range histogramAssertions {
		require.Equal(t, wantCounter.Name, snapshot[i].Name, "unexpected histogram")
		assert.EqualValues(t, wantCounter.Value, snapshot[i].Values, "unexpected histogram value")
		for wantTagKey, wantTagVal := range wantCounter.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

func initClientAndServer(t *testing.T) (json.Client, *metrics.Root, *metrics.Root, func()) {
	clientMetricsRoot, serverMetricsRoot := metrics.New(), metrics.New()

	serverAddr, cleanupServer := newServer(t, zapcore.NewNopCore(), serverMetricsRoot)
	client, cleanupClient := newClient(t, serverAddr, zapcore.NewNopCore(), clientMetricsRoot)

	return client, clientMetricsRoot, serverMetricsRoot, func() {
		cleanupServer()
		cleanupClient()
	}
}

type requestResponse struct {
	Val string
}

func newServer(t *testing.T, loggerCore zapcore.Core, metricsRoot *metrics.Root) (addr string, cleanup func()) {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("/test", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Write([]byte("healthy"))
	})
	inbound := http.NewTransport().NewInbound("127.0.0.1:0")
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _serverName,
		Inbounds: yarpc.Inbounds{inbound},
		Logging:  yarpc.LoggingConfig{Zap: zap.New(loggerCore)},
		Metrics:  yarpc.MetricsConfig{Metrics: metricsRoot.Scope()},
	})
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")
	dispatcher.Register(json.Procedure("test", func(ctx context.Context, req *requestResponse) (*requestResponse, error) {
		return &requestResponse{Val: req.Val}, nil
	}))
	addr = inbound.Addr().String()
	cleanup = func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return addr, cleanup
}

func newClient(t *testing.T, serverAddr string, loggerCore zapcore.Core, metricsRoot *metrics.Root) (json.Client, func()) {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _clientName,
		Outbounds: map[string]transport.Outbounds{
			_serverName: {
				ServiceName: _serverName,
				Unary:       http.NewTransport().NewSingleOutbound(fmt.Sprintf("http://%s", serverAddr)),
			},
		},
		Logging: yarpc.LoggingConfig{Zap: zap.New(loggerCore)},
		Metrics: yarpc.MetricsConfig{Metrics: metricsRoot.Scope()},
	})

	client := json.New(dispatcher.ClientConfig(_serverName))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	cleanup := func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") }
	return client, cleanup
}
