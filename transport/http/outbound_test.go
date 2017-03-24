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

package http

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			ttl := req.Header.Get(TTLMSHeader)
			ttlms, err := strconv.Atoi(ttl)
			assert.NoError(t, err, "can parse TTL header")
			assert.InDelta(t, ttlms, 1000, 5, "ttl header within tolerance")

			assert.Equal(t, "caller", req.Header.Get(CallerHeader))
			assert.Equal(t, "service", req.Header.Get(ServiceHeader))
			assert.Equal(t, "raw", req.Header.Get(EncodingHeader))
			assert.Equal(t, "hello", req.Header.Get(ProcedureHeader))

			body, err := ioutil.ReadAll(req.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("world"), body)
			}

			w.Header().Set("rpc-header-foo", "bar")
			_, err = w.Write([]byte("great success"))
			assert.NoError(t, err)
		},
	))
	defer successServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(successServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := out.Call(ctx, &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte("world")),
	})
	require.NoError(t, err)
	defer res.Body.Close()

	foo, ok := res.Headers.Get("foo")
	assert.True(t, ok, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("great success"), body)
	}
}

func TestOutboundHeaders(t *testing.T) {
	tests := []struct {
		desc    string
		context context.Context
		headers transport.Headers

		wantHeaders map[string]string
	}{
		{
			desc:    "application headers",
			headers: transport.NewHeaders().With("foo", "bar").With("baz", "Qux"),
			wantHeaders: map[string]string{
				"Rpc-Header-Foo": "bar",
				"Rpc-Header-Baz": "Qux",
			},
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				for k, v := range tt.wantHeaders {
					assert.Equal(
						t, v, r.Header.Get(k), "%v: header %v did not match", tt.desc, k)
				}
			},
		))
		defer server.Close()

		ctx := tt.context
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()
		}

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		res, err := out.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Headers:   tt.headers,
			Procedure: "hello",
			Body:      bytes.NewReader([]byte("world")),
		})

		if !assert.NoError(t, err, "%v: call failed", tt.desc) {
			continue
		}

		if !assert.NoError(t, res.Body.Close(), "%v: failed to close response body") {
			continue
		}
	}
}

func TestOutboundApplicationError(t *testing.T) {
	tests := []struct {
		desc     string
		status   string
		appError bool
	}{
		{
			desc:     "ok",
			status:   "success",
			appError: false,
		},
		{
			desc:     "error",
			status:   "error",
			appError: true,
		},
		{
			desc:     "not an error",
			status:   "lolwut",
			appError: false,
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Rpc-Status", tt.status)
				defer r.Body.Close()
			},
		))
		defer server.Close()

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		res, err := out.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "hello",
			Body:      bytes.NewReader([]byte("world")),
		})

		assert.Equal(t, res.ApplicationError, tt.appError, "%v: application status", tt.desc)

		if !assert.NoError(t, err, "%v: call failed", tt.desc) {
			continue
		}

		if !assert.NoError(t, res.Body.Close(), "%v: failed to close response body") {
			continue
		}
	}
}

func TestCallFailures(t *testing.T) {
	notFoundServer := httptest.NewServer(http.NotFoundHandler())
	defer notFoundServer.Close()

	internalErrorServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "great sadness", http.StatusInternalServerError)
		}))
	defer internalErrorServer.Close()

	httpTransport := NewTransport()

	tests := []struct {
		url      string
		messages []string
	}{
		{"not a URL", []string{"protocol scheme"}},
		{notFoundServer.URL, []string{"404", "page not found"}},
		{internalErrorServer.URL, []string{"great sadness"}},
	}

	for _, tt := range tests {
		out := httpTransport.NewSingleOutbound(tt.url)
		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := out.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "wat",
			Body:      bytes.NewReader([]byte("huh")),
		})
		assert.Error(t, err, "expected failure")
		for _, msg := range tt.messages {
			assert.Contains(t, err.Error(), msg)
		}
	}
}

func TestStartMultiple(t *testing.T) {
	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound("http://localhost:9999")

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Start()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestStopMultiple(t *testing.T) {
	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound("http://127.0.0.1:9999")

	err := out.Start()
	require.NoError(t, err)

	var wg sync.WaitGroup
	signal := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-signal

			err := out.Stop()
			assert.NoError(t, err)
		}()
	}
	close(signal)
	wg.Wait()
}

func TestCallWithoutStarting(t *testing.T) {
	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound("http://127.0.0.1:9999")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := out.Call(
		ctx,
		&transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "foo",
			Body:      bytes.NewReader([]byte("sup")),
		},
	)

	assert.Equal(t, context.DeadlineExceeded, err)
}
