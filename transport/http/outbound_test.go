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

package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestNewOutbound(t *testing.T) {
	ctrl := gomock.NewController(t)
	chooser := peertest.NewMockChooser(ctrl)

	out := NewOutbound(chooser)
	require.NotNil(t, out)
	assert.Equal(t, chooser, out.Chooser())
}

func TestNewSingleOutboundPanic(t *testing.T) {
	require.Panics(t, func() {
		// invalid url should cause panic
		NewTransport().NewSingleOutbound(":")
	},
		"expected to panic")
}

func TestCallSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			ttl := req.Header.Get(TTLMSHeader)
			ttlms, err := strconv.Atoi(ttl)
			assert.NoError(t, err, "can parse TTL header")
			assert.InDelta(t, ttlms, testtime.X*1000.0, testtime.X*5.0, "ttl header within tolerance")

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

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
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

type dialerTestFn struct {
	name   string
	callCh chan struct{}
}

func newCallee(name string) *dialerTestFn {
	ch := make(chan struct{})
	return &dialerTestFn{
		name:   name,
		callCh: ch,
	}
}

func (c *dialerTestFn) Call() {
	close(c.callCh)
}

func (c *dialerTestFn) Expect(t *testing.T, timeout time.Duration, f func()) error {
	f()

	timer := time.NewTimer(timeout)

	select {
	case <-c.callCh:
		timer.Stop()
		return nil
	case <-timer.C:
		return fmt.Errorf("timed out waiting for %s name", c.name)
	}
}

func TestCallSuccessDialer(t *testing.T) {
	// This tests verifies that dialer interception code will correctly disconnect a peer if
	// Dialer library will catch an error on a connected peer.
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
		},
	))

	dialer := newCallee("dialer")
	closer := newCallee("closer")

	httpTransport := NewTransport(
		DialerCalled(dialer.Call),
		CloserCalled(closer.Call))

	out := httpTransport.NewSingleOutbound(successServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	res, err := out.Call(ctx, &transport.Request{})
	require.NoError(t, err)
	defer res.Body.Close()

	t.Run("expect closer call", func(t *testing.T) {
		assert.NoError(t,
			closer.Expect(t, testtime.Second, func() {
				successServer.Close()
			}))
	})

	t.Run("expect dialer call", func(t *testing.T) {
		ctx, cancel = context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		assert.NoError(t,
			dialer.Expect(t, time.Second, func() {
				_, err = out.Call(ctx, &transport.Request{})
				require.Error(t, err)
			}))
	})
}

func TestAddReservedHeader(t *testing.T) {
	tests := []string{
		"Rpc-Foo",
		"rpc-header-foo",
		"RPC-Bar",
	}

	for _, tt := range tests {
		assert.Panics(t, func() { AddHeader(tt, "bar") })
	}
}

func TestOutboundHeaders(t *testing.T) {
	tests := []struct {
		desc    string
		context context.Context
		headers transport.Headers
		opts    []OutboundOption

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
		{
			desc:    "extra headers",
			headers: transport.NewHeaders().With("x", "y"),
			opts: []OutboundOption{
				AddHeader("X-Foo", "bar"),
				AddHeader("X-BAR", "BAZ"),
			},
			wantHeaders: map[string]string{
				"Rpc-Header-X": "y",
				"X-Foo":        "bar",
				"X-Bar":        "BAZ",
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
			ctx, cancel = context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()
		}

		out := httpTransport.NewSingleOutbound(server.URL, tt.opts...)
		assert.Len(t, out.Transports(), 1, "transports must contain the transport")
		// we use == instead of assert.Equal because we want to do a pointer
		// comparison
		assert.True(t, httpTransport == out.Transports()[0], "transports must match")

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
		ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
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

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 200*testtime.Millisecond)
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

	assert.Equal(t, yarpcerrors.FailedPreconditionErrorf("error waiting for HTTP outbound to start for service: service: context finished while waiting for instance to start: context deadline exceeded"), err)
}

func TestGetPeerForRequestErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name string
		peer *peertest.MockPeer
		err  error
	}{
		{
			name: "error choosing peer",
		},
		{
			name: "error casting peer",
			peer: peertest.NewMockPeer(ctrl),
			err:  errors.New("err"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chooser := peertest.NewMockChooser(ctrl)

			out := NewTransport().NewSingleOutbound("http://127.0.0.1:9999")
			out.chooser = chooser

			ctx := context.Background()
			treq := &transport.Request{}

			chooser.EXPECT().Choose(ctx, treq).Return(tt.peer, nil, tt.err)

			_, _, err := out.getPeerForRequest(ctx, treq)
			require.Error(t, err)
		})
	}
}

func TestWithCoreHeaders(t *testing.T) {
	endpoint := "http://127.0.0.1:9999"
	out := NewTransport().NewSingleOutbound(endpoint)
	require.NoError(t, out.Start())

	httpReq := httptest.NewRequest("", endpoint, nil)

	shardKey := "sharding"
	routingKey := "routing"
	routingDelegate := "delegate"

	treq := &transport.Request{
		ShardKey:        shardKey,
		RoutingKey:      routingKey,
		RoutingDelegate: routingDelegate,
	}
	result := out.withCoreHeaders(httpReq, treq, time.Second)

	assert.Equal(t, shardKey, result.Header.Get(ShardKeyHeader))
	assert.Equal(t, routingKey, result.Header.Get(RoutingKeyHeader))
	assert.Equal(t, routingDelegate, result.Header.Get(RoutingDelegateHeader))
}

func TestNoRequest(t *testing.T) {
	tran := NewTransport()
	out := tran.NewSingleOutbound("localhost:0")

	_, err := out.Call(context.Background(), nil)
	assert.Equal(t, yarpcerrors.InvalidArgumentErrorf("request for http unary outbound was nil"), err)

	_, err = out.CallOneway(context.Background(), nil)
	assert.Equal(t, yarpcerrors.InvalidArgumentErrorf("request for http oneway outbound was nil"), err)
}

func TestOutboundNoDeadline(t *testing.T) {
	out := NewTransport().NewSingleOutbound("http://foo-host:8080")

	_, err := out.call(context.Background(), &transport.Request{})
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing context deadline"), err)
}

func TestServiceMatchSuccess(t *testing.T) {
	matchServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			w.Header().Set(ServiceHeader, req.Header.Get(ServiceHeader))
			_, err := w.Write([]byte("Service name header return"))
			assert.NoError(t, err)
		},
	))
	defer matchServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(matchServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := out.Call(ctx, &transport.Request{
		Service: "Service",
	})
	require.NoError(t, err)
}

func TestServiceMatchFailed(t *testing.T) {
	mismatchServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			w.Header().Set(ServiceHeader, "ThisIsAWrongSvcName")
			_, err := w.Write([]byte("Wrong service name header return"))
			assert.NoError(t, err)
		},
	))
	defer mismatchServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(mismatchServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := out.Call(ctx, &transport.Request{
		Service: "Service",
	})
	assert.Error(t, err, "expected failure for service name dismatch")
}

func TestServiceMatchNoHeader(t *testing.T) {
	noHeaderServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			// intentionally do not set a response header
			_, err := w.Write([]byte("No service name header return"))
			assert.NoError(t, err)
		},
	))
	defer noHeaderServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(noHeaderServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err := out.Call(ctx, &transport.Request{
		Service: "Service",
	})
	require.NoError(t, err)
}
