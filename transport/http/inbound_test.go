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

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/http2test"
	"go.uber.org/yarpc/internal/routertest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestStartAddrInUse(t *testing.T) {
	t1 := NewTransport()
	i1 := t1.NewInbound("127.0.0.1:0")

	assert.Len(t, i1.Transports(), 1, "transports must contain the transport")
	// we use == instead of assert.Equal because we want to do a pointer
	// comparison
	assert.True(t, t1 == i1.Transports()[0], "transports must match")

	i1.SetRouter(newTestRouter(nil))
	require.NoError(t, i1.Start(), "inbound 1 must start without an error")
	t2 := NewTransport()
	i2 := t2.NewInbound(i1.Addr().String())
	i2.SetRouter(newTestRouter(nil))
	err := i2.Start()

	require.Error(t, err)
	oe, ok := err.(*net.OpError)
	assert.True(t, ok && oe.Op == "listen", "expected a listen error")
	if ok {
		se, ok := oe.Err.(*os.SyscallError)
		assert.True(t, ok && se.Syscall == "bind" && se.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}

	assert.NoError(t, i1.Stop())
}

func TestNilAddrAfterStop(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("127.0.0.1:0")
	i.SetRouter(newTestRouter(nil))
	require.NoError(t, i.Start())
	assert.NotEqual(t, "127.0.0.1:0", i.Addr().String())
	assert.NotNil(t, i.Addr())
	assert.NoError(t, i.Stop())
	assert.Nil(t, i.Addr())
}

func TestInboundStartAndStop(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("127.0.0.1:0")
	i.SetRouter(newTestRouter(nil))
	require.NoError(t, i.Start())
	assert.NotEqual(t, "127.0.0.1:0", i.Addr().String())
	assert.NoError(t, i.Stop())
}

func TestInboundStartError(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("invalid")
	i.SetRouter(new(transporttest.MockRouter))
	assert.Error(t, i.Start(), "expected failure")
}

func TestInboundStartErrorBadGrabHeader(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("127.0.0.1:0", GrabHeaders("x-valid", "y-invalid"))
	i.SetRouter(new(transporttest.MockRouter))
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(i.Start()).Code())
}

func TestInboundHeaderPreallocation(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("127.0.0.1:0")
	assert.Equal(t, HeaderPreallocationUnfiltered, i.headerPreallocationStrategy)

	tests := []struct {
		name     string
		strategy HeaderPreallocationStrategy
	}{
		{name: "unfiltered", strategy: HeaderPreallocationUnfiltered},
		{name: "scan", strategy: HeaderPreallocationScan},
		{name: "disabled", strategy: HeaderPreallocationDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := NewTransport()
			i := x.NewInbound(
				"127.0.0.1:0",
				InboundHeaderPreallocation(tt.strategy),
			)
			assert.Equal(t, tt.strategy, i.headerPreallocationStrategy)
		})
	}
}

func TestInboundStartErrorInvalidHeaderPreallocation(t *testing.T) {
	tests := []struct {
		name     string
		strategy HeaderPreallocationStrategy
	}{
		{name: "empty", strategy: ""},
		{name: "unknown", strategy: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := NewTransport()
			i := x.NewInbound(
				"127.0.0.1:0",
				InboundHeaderPreallocation(tt.strategy),
			)
			i.SetRouter(new(transporttest.MockRouter))

			err := i.Start()
			assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(err).Code())
			assert.Contains(t, err.Error(), "unknown header preallocation strategy")
		})
	}
}

func TestInboundStopWithoutStarting(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("127.0.0.1:8000")
	assert.Nil(t, i.Addr())
	assert.NoError(t, i.Stop())
}

func TestInboundMux(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpTransport := NewTransport()
	defer httpTransport.Stop()
	// TODO transport lifecycle

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	i := httpTransport.NewInbound("127.0.0.1:0", Mux("/rpc/v1", mux))
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	reg := transporttest.NewMockRouter(mockCtrl)
	reg.EXPECT().Procedures()
	i.SetRouter(reg)
	require.NoError(t, i.Start())

	defer i.Stop()

	addr := fmt.Sprintf("http://%v/", yarpctest.ZeroAddrToHostPort(i.Addr()))
	resp, err := http.Get(addr + "health")
	if assert.NoError(t, err, "/health failed") {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if assert.NoError(t, err, "/health body read error") {
			assert.Equal(t, "healthy", string(body), "/health body mismatch")
		}
	}

	// this should fail
	o := httpTransport.NewSingleOutbound(addr)
	require.NoError(t, o.Start(), "failed to start outbound")
	defer o.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, err = o.Call(ctx, &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      strings.NewReader("derp"),
	})

	if assert.Error(t, err, "RPC call to / should have failed") {
		assert.Equal(t, yarpcerrors.CodeNotFound, yarpcerrors.FromError(err).Code())
	}

	o.setURLTemplate("http://host:12345/rpc/v1")
	require.NoError(t, o.Start(), "failed to start outbound")
	defer o.Stop()

	spec := transport.NewUnaryHandlerSpec(h)
	reg.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithCaller("foo").
		WithService("bar").
		WithProcedure("hello"),
	).Return(spec, nil)

	h.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	res, err := o.Call(ctx, &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      strings.NewReader("derp"),
	})

	if assert.NoError(t, err, "expected rpc request to succeed") {
		defer res.Body.Close()
		s, err := io.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Empty(t, s)
		}
	}
}

func TestMuxWithInterceptor(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "/health",
			want: "OK",
		},
		{
			path: "/",
			want: "intercepted",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})
	intercept := func(transportHandler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "intercepted")
		})
	}

	transport := NewTransport()
	inbound := transport.NewInbound("127.0.0.1:0", Mux("/", mux), Interceptor(intercept))
	inbound.SetRouter(newTestRouter(nil))
	require.NoError(t, inbound.Start(), "Failed to start inbound")
	defer inbound.Stop()

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{inbound},
	})
	require.NoError(t, dispatcher.Start(), "Failed to start dispatcher")
	defer dispatcher.Stop()

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			url := fmt.Sprintf("http://%v%v", inbound.Addr(), tt.path)
			_, body, err := httpGet(t, url)
			require.NoError(t, err, "request failed")
			assert.Equal(t, tt.want, string(body))
		})
	}
}

func TestMultipleInterceptors(t *testing.T) {
	const (
		yarpcResp  = "YARPC response"
		viaResp    = "Via response"
		healthResp = "health response"
		userResp   = "user response"
	)
	// This should be the underlying yarpc handler but it can't be set directly
	// For the ease of testing, use an interceptor and register it last.
	baseHandler := Interceptor(func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, yarpcResp)
		})
	})

	viaInterceptor := Interceptor(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, viaResp)
			h.ServeHTTP(w, r)
		})
	})

	healthInterceptor := Interceptor(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				io.WriteString(w, healthResp)
			} else {
				h.ServeHTTP(w, r)
			}
		})
	})

	userInterceptor := Interceptor(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/user" {
				io.WriteString(w, userResp)
			} else {
				h.ServeHTTP(w, r)
			}
		})
	})

	tests := []struct {
		msg          string
		interceptors []InboundOption
		url          string
		want         string
	}{
		{
			msg:          "no user interceptor, /yarpc",
			interceptors: []InboundOption{healthInterceptor},
			url:          "/yarpc",
			want:         yarpcResp,
		},
		{
			msg:          "no user interceptor, /health",
			interceptors: []InboundOption{healthInterceptor},
			url:          "/health",
			want:         healthResp,
		},
		{
			msg:          "no user interceptor, /user",
			interceptors: []InboundOption{healthInterceptor},
			url:          "/user",
			want:         yarpcResp,
		},
		{
			msg:          "user interceptor, /yarpc",
			interceptors: []InboundOption{healthInterceptor, userInterceptor},
			url:          "/yarpc",
			want:         yarpcResp,
		},
		{
			msg:          "user interceptor, /health",
			interceptors: []InboundOption{healthInterceptor, userInterceptor},
			url:          "/health",
			want:         healthResp,
		},
		{
			msg:          "user interceptor, /user",
			interceptors: []InboundOption{healthInterceptor, userInterceptor},
			url:          "/user",
			want:         userResp,
		},
		{
			msg:          "ordering guaranteed",
			interceptors: []InboundOption{viaInterceptor},
			url:          "/yarpc",
			want:         viaResp + yarpcResp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			transport := NewTransport()
			inbound := transport.NewInbound("127.0.0.1:0", append(tt.interceptors, baseHandler)...)
			inbound.SetRouter(newTestRouter(nil))
			require.NoError(t, inbound.Start(), "Failed to start inbound")
			defer inbound.Stop()

			dispatcher := yarpc.NewDispatcher(yarpc.Config{
				Name:     "server",
				Inbounds: yarpc.Inbounds{inbound},
			})
			require.NoError(t, dispatcher.Start(), "Failed to start dispatcher")
			defer dispatcher.Stop()

			url := fmt.Sprintf("http://%v%v", inbound.Addr(), tt.url)
			_, body, err := httpGet(t, url)
			require.NoError(t, err, "request failed")
			assert.Equal(t, tt.want, string(body))
		})
	}
}

func TestRequestAfterStop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})

	transport := NewTransport()
	inbound := transport.NewInbound("127.0.0.1:0", Mux("/", mux))
	inbound.SetRouter(newTestRouter(nil))
	require.NoError(t, inbound.Start(), "Failed to start inbound")

	url := fmt.Sprintf("http://%v/health", inbound.Addr())
	_, body, err := httpGet(t, url)
	require.NoError(t, err, "expect successful response")
	assert.Equal(t, "OK", body, "response mismatch")

	require.NoError(t, inbound.Stop(), "Failed to stop inbound")

	_, _, err = httpGet(t, url)
	assert.Error(t, err, "requests should fail once inbound is stopped")
}

func TestInboundWithHTTPVersion(t *testing.T) {
	t.Run("HTTP1", func(t *testing.T) {
		// Create a new transport that supports both HTTP/1.1 and HTTP/2
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888")

		dispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name:     "myservice",
			Inbounds: yarpc.Inbounds{inbound},
		})
		require.NoError(t, dispatcher.Start(), "failed to start dispatcher")
		t.Cleanup(func() { _ = dispatcher.Stop() })

		// Make a request to the /health endpoint.
		res, err := http.Get("http://127.0.0.1:8888/health")
		require.Nil(t, err, "got error making request: %v", err)
		t.Cleanup(func() { _ = res.Body.Close() })
	})

	t.Run("HTTP2", func(t *testing.T) {
		// Create a new transport that supports both HTTP/1.1 and HTTP/2
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888")

		dispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name:     "myservice",
			Inbounds: yarpc.Inbounds{inbound},
		})
		require.NoError(t, dispatcher.Start(), "failed to start dispatcher")
		t.Cleanup(func() { _ = dispatcher.Stop() })

		client := http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, network, addr)
				},
			},
		}
		t.Cleanup(client.CloseIdleConnections)
		// Make a request to the /health endpoint.
		res, err := client.Get("http://127.0.0.1:8888/health")
		require.Nil(t, err, "got error making request: %v", err)
		t.Cleanup(func() { _ = res.Body.Close() })
	})

	t.Run("HTTP2 should fail when disabledHTTP2 flag is set", func(t *testing.T) {
		// Create a new transport that only supports HTTP/1.1
		testTransport := NewTransport()
		// Disable HTTP/2
		inbound := testTransport.NewInbound("127.0.0.1:8888", DisableHTTP2(true))

		dispatcher := yarpc.NewDispatcher(yarpc.Config{
			Name:     "myservice",
			Inbounds: yarpc.Inbounds{inbound},
		})
		require.NoError(t, dispatcher.Start(), "failed to start dispatcher")
		t.Cleanup(func() { _ = dispatcher.Stop() })

		client := http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, network, addr)
				},
			},
		}
		t.Cleanup(client.CloseIdleConnections)
		// Make a request to the /health endpoint.
		res, err := client.Get("http://127.0.0.1:8888/health")
		require.Error(t, err, "expected error making request")
		require.Nil(t, res, "expected response to be nil")
	})
}

func TestInboundWithTimeouts(t *testing.T) {
	t.Run("ReadHeaderTimeout", func(t *testing.T) {
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888", ReadHeaderTimeout(5*time.Second))

		require.Equal(t, 5*time.Second, inbound.server.ReadHeaderTimeout)
	})

	t.Run("WriteTimeout", func(t *testing.T) {
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888", WriteTimeout(5*time.Second))

		require.Equal(t, 5*time.Second, inbound.server.WriteTimeout)
	})

	t.Run("ReadTimeout", func(t *testing.T) {
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888", ReadTimeout(5*time.Second))

		require.Equal(t, 5*time.Second, inbound.server.ReadTimeout)
	})

	t.Run("IdleTimeout", func(t *testing.T) {
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888", IdleTimeout(60*time.Second))

		require.Equal(t, 60*time.Second, inbound.server.IdleTimeout)
	})

	t.Run("EnableOverrideOriginalItemWithCanonicalizedKey", func(t *testing.T) {
		testTransport := NewTransport()
		inbound := testTransport.NewInbound("127.0.0.1:8888", EnableOverrideOriginalItemWithCanonicalizedKey())

		require.True(t, inbound.overrideOriginalItemWithCanonicalizedKey)
	})
}

func TestInboundDuplicateHeaderCounterVecWithMeter(t *testing.T) {
	// Verify that the inbound creates the duplicateHeaderCounterVec when
	// a meter is provided, and increments it on duplicate headers.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	root := metrics.New()
	meter := root.Scope()

	httpTransport := NewTransport(Meter(meter))
	i := httpTransport.NewInbound("127.0.0.1:0", GrabHeaders("x-custom"))

	rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
	router := transporttest.NewMockRouter(mockCtrl)
	router.EXPECT().Procedures().AnyTimes()
	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithService("svc").
		WithProcedure("proc"),
	).Return(transport.NewUnaryHandlerSpec(rpcHandler), nil)

	rpcHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	i.SetRouter(router)
	require.NoError(t, i.Start())
	defer i.Stop()

	addr := fmt.Sprintf("http://%v", yarpctest.ZeroAddrToHostPort(i.Addr()))
	reqBody := strings.NewReader("")
	httpReq, err := http.NewRequest("POST", addr, reqBody)
	require.NoError(t, err)

	httpReq.Header.Set(CallerHeader, "caller")
	httpReq.Header.Set(ServiceHeader, "svc")
	httpReq.Header.Set(EncodingHeader, "raw")
	httpReq.Header.Set(ProcedureHeader, "proc")
	httpReq.Header.Set(TTLMSHeader, "1000")
	// Duplicate: prefixed and non-prefixed with different values.
	httpReq.Header.Set("Rpc-Header-X-Custom", "prefixed-val")
	httpReq.Header.Set("X-Custom", "non-prefixed-val")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	snap := root.Snapshot()
	expected := metrics.Snapshot{
		Name:  "yarpc_duplicate_headers",
		Value: 1,
		Tags: metrics.Tags{
			"header_name": "x-custom",
			"source":      "caller",
			"dest":        "svc",
			"service":     "yarpc",
		},
	}
	assert.Contains(t, snap.Counters, expected, "expected duplicate_headers counter to be incremented")
}

func TestInboundDuplicateHeaderCounterVecWithoutMeter(t *testing.T) {
	// Verify that the inbound starts successfully without a meter and
	// handles duplicate headers without panic.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpTransport := NewTransport() // no meter
	i := httpTransport.NewInbound("127.0.0.1:0", GrabHeaders("x-custom"))

	rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
	router := transporttest.NewMockRouter(mockCtrl)
	router.EXPECT().Procedures().AnyTimes()
	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithService("svc").
		WithProcedure("proc"),
	).Return(transport.NewUnaryHandlerSpec(rpcHandler), nil)

	rpcHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	i.SetRouter(router)
	require.NoError(t, i.Start())
	defer i.Stop()

	addr := fmt.Sprintf("http://%v", yarpctest.ZeroAddrToHostPort(i.Addr()))
	reqBody := strings.NewReader("")
	httpReq, err := http.NewRequest("POST", addr, reqBody)
	require.NoError(t, err)

	httpReq.Header.Set(CallerHeader, "caller")
	httpReq.Header.Set(ServiceHeader, "svc")
	httpReq.Header.Set(EncodingHeader, "raw")
	httpReq.Header.Set(ProcedureHeader, "proc")
	httpReq.Header.Set(TTLMSHeader, "1000")
	// Duplicate headers — should not panic even without meter.
	httpReq.Header.Set("Rpc-Header-X-Custom", "prefixed-val")
	httpReq.Header.Set("X-Custom", "non-prefixed-val")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestInboundDuplicateHeaderCounterVecNoDuplicates(t *testing.T) {
	// With a meter, but no duplicate headers: counter should not be incremented.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	root := metrics.New()
	meter := root.Scope()

	httpTransport := NewTransport(Meter(meter))
	i := httpTransport.NewInbound("127.0.0.1:0", GrabHeaders("x-custom"))

	rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
	router := transporttest.NewMockRouter(mockCtrl)
	router.EXPECT().Procedures().AnyTimes()
	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithService("svc").
		WithProcedure("proc"),
	).Return(transport.NewUnaryHandlerSpec(rpcHandler), nil)

	rpcHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	i.SetRouter(router)
	require.NoError(t, i.Start())
	defer i.Stop()

	addr := fmt.Sprintf("http://%v", yarpctest.ZeroAddrToHostPort(i.Addr()))
	reqBody := strings.NewReader("")
	httpReq, err := http.NewRequest("POST", addr, reqBody)
	require.NoError(t, err)

	httpReq.Header.Set(CallerHeader, "caller")
	httpReq.Header.Set(ServiceHeader, "svc")
	httpReq.Header.Set(EncodingHeader, "raw")
	httpReq.Header.Set(ProcedureHeader, "proc")
	httpReq.Header.Set(TTLMSHeader, "1000")
	// Only non-prefixed header, no duplicate.
	httpReq.Header.Set("X-Custom", "only-val")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	snap := root.Snapshot()
	for _, c := range snap.Counters {
		if c.Name == "yarpc_duplicate_headers" {
			t.Fatal("duplicate counter should not exist when there are no duplicate headers")
		}
	}
}

func httpGet(t *testing.T, url string) (*http.Response, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("GET %v failed: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to read reponse from %v: %v", url, err)
	}

	return resp, string(body), nil
}

// TestInboundH2CGracefulShutdown stops an inbound with a call in flight on an
// HTTP/2 cleartext (h2c) connection, then drops every server-side connection
// the moment Stop returns, as the process exiting would.
//
// Stop must hold the contract grpc.Server.GracefulStop provides gRPC
// inbounds: deliver a GOAWAY and let in-flight calls complete before it
// returns, since nothing still owed to the client survives the exit.
//
// The GOAWAY must follow the two-phase graceful shutdown of RFC 9113 §6.8: a
// first GOAWAY reserving the whole stream ID space, then, after at least one
// round trip, a second one carrying the last stream actually processed.
func TestInboundH2CGracefulShutdown(t *testing.T) {
	conns := &serverConns{}
	inbound, h := startBlockingInbound(t, conns.record())
	addr := inbound.Addr().String()

	c := http2test.DialRawH2C(t, addr)
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	var seq, stopSeq atomic.Int64
	stopErr := make(chan error, 1)
	go func() {
		err := inbound.Stop()
		stopSeq.Store(seq.Inc())
		stopErr <- err
	}()

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	first := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, first.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(math.MaxInt32), first.LastStreamID,
		"first GOAWAY must reserve the whole stream ID space")

	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)...)
	goAwaySeq := seq.Inc()
	second := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, second.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), second.LastStreamID, "second GOAWAY must carry the last processed stream")

	releaseSeq := seq.Inc()
	h.releaseAll()
	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.StreamDone(1))...)
	res := http2test.ResponseOn(frames, 1)
	require.False(t, res.Reset, "in-flight stream 1 was reset with %v instead of completing", res.ResetCode)
	assert.Equal(t, "200", res.Headers[":status"], "unexpected status for in-flight stream 1")
	assert.Equal(t, "payload", string(res.Body), "unexpected body for in-flight stream 1")

	// The client is done: closing its side lets the server tear down the
	// connection without waiting out its post-GOAWAY grace period.
	require.NoError(t, c.Close())

	select {
	case err := <-stopErr:
		require.NoError(t, err, "Stop must succeed once connections have drained")
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after the connection drained")
	}
	conns.closeAll()
	assert.Less(t, goAwaySeq, stopSeq.Load(),
		"Stop returned before the client observed a GOAWAY, which exiting the process would lose")
	assert.Less(t, releaseSeq, stopSeq.Load(),
		"Stop returned with stream 1 still in flight, which exiting the process would lose")

	_, err := net.Dial("tcp", addr)
	assert.Error(t, err, "new connections must be refused once Stop returns")
}

// TestInboundH2CStopWithWedgedHandler stops an inbound whose h2c connection
// has a call that never completes: Stop must still complete both phases of
// the RFC 9113 §6.8 graceful shutdown, neither give up on the call before
// ShutdownTimeout nor wait past it, and close the connection rather than leave
// it behind.
func TestInboundH2CStopWithWedgedHandler(t *testing.T) {
	timeout := 200 * testtime.Millisecond
	inbound, h := startBlockingInbound(t, ShutdownTimeout(timeout))

	c := http2test.DialRawH2C(t, inbound.Addr().String())
	c.OpenStream(t, 1, "/", h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	start := time.Now()
	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()

	frames := c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)
	first := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, first.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(math.MaxInt32), first.LastStreamID,
		"first GOAWAY must reserve the whole stream ID space")

	// The second GOAWAY does not wait for in-flight calls: it only waits for
	// the client to acknowledge the first, so a wedged call cannot hold it.
	frames = append(frames, c.ReadUntil(t, 5*testtime.Second, http2test.IsGoAway)...)
	second := frames[len(frames)-1]
	assert.Equal(t, http2.ErrCodeNo, second.ErrCode, "graceful shutdown must use NO_ERROR")
	assert.Equal(t, uint32(1), second.LastStreamID, "second GOAWAY must carry the last processed stream")

	select {
	case err := <-stopErr:
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Stop must report that the drain timed out")
	case <-time.After(timeout + 5*testtime.Second):
		require.FailNow(t, "Stop hung past ShutdownTimeout")
	}
	assert.GreaterOrEqual(t, time.Since(start), timeout,
		"Stop gave up on the in-flight call before ShutdownTimeout")

	frames = append(frames, c.ReadUntilClosed(t, 5*testtime.Second)...)
	assert.False(t, http2test.ResponseOn(frames, 1).Complete,
		"stream 1 completed although its handler never returned")
}

// TestInboundH2CShutdownWithPingAckHeldBack stops an inbound whose h2c client
// never acknowledges the PING sent with the first GOAWAY: the second GOAWAY
// must still arrive, but only once the PING ACK timeout has passed.
func TestInboundH2CShutdownWithPingAckHeldBack(t *testing.T) {
	// How long the HTTP/2 shutdown helper waits for the PING ACK.
	const pingAckTimeout = time.Second

	inbound, h := startBlockingInbound(t)
	c := dialManualPingH2C(t, inbound.Addr().String())
	c.openStream(t, 1, h2cRequestHeaders("block"), []byte("payload"))
	h.awaitEntered(t)

	start := time.Now()
	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()

	assert.Equal(t, uint32(math.MaxInt32), c.nextGoAway(t),
		"first GOAWAY must reserve the whole stream ID space")
	c.nextPing(t) // held back: never acknowledged

	assert.Equal(t, uint32(1), c.nextGoAway(t), "second GOAWAY must carry the last processed stream")
	assert.GreaterOrEqual(t, time.Since(start), pingAckTimeout,
		"second GOAWAY must wait for the PING ACK until it times out")

	h.releaseAll()
	assert.Equal(t, map[uint32]string{1: "payload"}, c.responses(t, 1), "in-flight stream 1 must complete")
	require.NoError(t, c.conn.Close())

	select {
	case err := <-stopErr:
		require.NoError(t, err, "Stop must succeed once connections have drained")
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after the connection drained")
	}
}

// TestInboundH2CShutdownServesStreamOpenedBeforePingAck stops an inbound
// whose h2c client opens a stream after the first GOAWAY, but before it
// acknowledges the PING sent with it: the client may have sent the stream
// before it read the GOAWAY, so the stream must be served, and the second
// GOAWAY must cover it.
func TestInboundH2CShutdownServesStreamOpenedBeforePingAck(t *testing.T) {
	inbound, h := startBlockingInbound(t)
	c := dialManualPingH2C(t, inbound.Addr().String())
	c.openStream(t, 1, h2cRequestHeaders("block"), []byte("payload-1"))
	h.awaitEntered(t)

	stopErr := make(chan error, 1)
	go func() { stopErr <- inbound.Stop() }()

	assert.Equal(t, uint32(math.MaxInt32), c.nextGoAway(t),
		"first GOAWAY must reserve the whole stream ID space")
	ping := c.nextPing(t)

	c.openStream(t, 3, h2cRequestHeaders("block"), []byte("payload-3"))
	h.awaitEntered(t)
	c.ackPing(t, ping)

	assert.Equal(t, uint32(3), c.nextGoAway(t),
		"second GOAWAY must carry the stream opened before the PING ACK")

	h.releaseAll()
	assert.Equal(t, map[uint32]string{1: "payload-1", 3: "payload-3"}, c.responses(t, 1, 3),
		"both streams must complete")
	require.NoError(t, c.conn.Close())

	select {
	case err := <-stopErr:
		require.NoError(t, err, "Stop must succeed once connections have drained")
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "Stop did not return after the connection drained")
	}
}

// manualPingH2CConn is a frame-level h2c client that, unlike
// http2test.RawH2CConn, leaves the server's PING frames for the test to
// acknowledge, or not. It reads and writes on the test goroutine.
type manualPingH2CConn struct {
	addr   string
	conn   net.Conn
	framer *http2.Framer
	encBuf bytes.Buffer
	enc    *hpack.Encoder
}

func dialManualPingH2C(t *testing.T, addr string) *manualPingH2CConn {
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err, "failed to dial %v", addr)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*testtime.Second)))

	c := &manualPingH2CConn{addr: addr, conn: conn, framer: http2.NewFramer(conn, conn)}
	c.enc = hpack.NewEncoder(&c.encBuf)
	c.framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	_, err = io.WriteString(conn, http2.ClientPreface)
	require.NoError(t, err, "failed to write client preface")
	require.NoError(t, c.framer.WriteSettings(), "failed to write client SETTINGS")
	return c
}

// openStream writes a POST request on a new stream, ending it with body.
func (c *manualPingH2CConn) openStream(t *testing.T, streamID uint32, headers map[string]string, body []byte) {
	c.encBuf.Reset()
	fields := []hpack.HeaderField{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: "http"},
		{Name: ":authority", Value: c.addr},
		{Name: ":path", Value: "/"},
	}
	for k, v := range headers {
		fields = append(fields, hpack.HeaderField{Name: k, Value: v})
	}
	for _, f := range fields {
		require.NoError(t, c.enc.WriteField(f))
	}
	require.NoError(t, c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: c.encBuf.Bytes(),
		EndHeaders:    true,
	}), "failed to open stream %d", streamID)
	require.NoError(t, c.framer.WriteData(streamID, true, body), "failed to write stream %d", streamID)
}

// next returns the next frame other than SETTINGS, which it acknowledges, and
// WINDOW_UPDATE.
func (c *manualPingH2CConn) next(t *testing.T) http2.Frame {
	for {
		f, err := c.framer.ReadFrame()
		require.NoError(t, err, "failed to read a frame")
		switch f := f.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				require.NoError(t, c.framer.WriteSettingsAck())
			}
			continue
		case *http2.WindowUpdateFrame:
			continue
		}
		return f
	}
}

func (c *manualPingH2CConn) nextGoAway(t *testing.T) uint32 {
	f := c.next(t)
	goAway, ok := f.(*http2.GoAwayFrame)
	require.True(t, ok, "want a GOAWAY frame, got %v", f)
	assert.Equal(t, http2.ErrCodeNo, goAway.ErrCode, "graceful shutdown must use NO_ERROR")
	return goAway.LastStreamID
}

func (c *manualPingH2CConn) nextPing(t *testing.T) [8]byte {
	f := c.next(t)
	ping, ok := f.(*http2.PingFrame)
	require.True(t, ok, "want a PING frame, got %v", f)
	require.False(t, ping.IsAck(), "want a PING frame, got a PING ACK")
	return ping.Data
}

func (c *manualPingH2CConn) ackPing(t *testing.T, data [8]byte) {
	require.NoError(t, c.framer.WritePing(true, data), "failed to acknowledge the PING")
}

// responses reads the responses on streamIDs, which may interleave, until
// each ends, and returns their bodies. Each must have status 200.
func (c *manualPingH2CConn) responses(t *testing.T, streamIDs ...uint32) map[uint32]string {
	open := make(map[uint32]bool, len(streamIDs))
	for _, id := range streamIDs {
		open[id] = true
	}
	bodies := make(map[uint32]string, len(streamIDs))
	for len(open) > 0 {
		f := c.next(t)
		id := f.Header().StreamID
		require.True(t, open[id], "unexpected frame %v", f)
		switch f := f.(type) {
		case *http2.MetaHeadersFrame:
			if status := f.PseudoValue("status"); status != "" {
				assert.Equal(t, "200", status, "unexpected status for stream %d", id)
			}
			if f.StreamEnded() {
				delete(open, id)
			}
		case *http2.DataFrame:
			bodies[id] += string(f.Data())
			if f.StreamEnded() {
				delete(open, id)
			}
		case *http2.RSTStreamFrame:
			require.FailNow(t, "stream reset", "stream %d was reset with %v", id, f.ErrCode)
		}
	}
	return bodies
}

// TestInboundStopWithoutConnections guards against Stop blocking on HTTP/2
// connection bookkeeping when no connection was ever opened.
func TestInboundStopWithoutConnections(t *testing.T) {
	for _, disableHTTP2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("DisableHTTP2=%v", disableHTTP2), func(t *testing.T) {
			inbound, _ := startBlockingInbound(t, DisableHTTP2(disableHTTP2))

			stopErr := make(chan error, 1)
			go func() { stopErr <- inbound.Stop() }()
			select {
			case err := <-stopErr:
				assert.NoError(t, err)
			case <-time.After(testtime.Second):
				require.FailNow(t, "Stop must return promptly with no open connections")
			}
			assert.NoError(t, inbound.Stop(), "Stop must be idempotent")
		})
	}
}

// TestInboundHTTP1GracefulShutdown pins the HTTP/1.1 shutdown behavior, which
// net/http already gets right, so that HTTP/2 changes cannot regress it.
func TestInboundHTTP1GracefulShutdown(t *testing.T) {
	for _, disableHTTP2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("DisableHTTP2=%v", disableHTTP2), func(t *testing.T) {
			inbound, h := startBlockingInbound(t, DisableHTTP2(disableHTTP2))
			addr := inbound.Addr().String()

			client := &http.Client{Transport: &http.Transport{}}
			t.Cleanup(client.CloseIdleConnections)
			type result struct {
				status int
				body   string
				err    error
			}
			resCh := make(chan result, 1)
			go func() {
				req, err := http.NewRequest(http.MethodPost, "http://"+addr, strings.NewReader("payload"))
				if err != nil {
					resCh <- result{err: err}
					return
				}
				for k, v := range h2cRequestHeaders("block") {
					req.Header.Set(k, v)
				}
				res, err := client.Do(req)
				if err != nil {
					resCh <- result{err: err}
					return
				}
				defer res.Body.Close()
				body, err := io.ReadAll(res.Body)
				resCh <- result{status: res.StatusCode, body: string(body), err: err}
			}()
			h.awaitEntered(t)

			var seq, stopSeq atomic.Int64
			stopErr := make(chan error, 1)
			go func() {
				err := inbound.Stop()
				stopSeq.Store(seq.Inc())
				stopErr <- err
			}()

			// Shutdown closes the listener before draining, so a refused
			// dial is the signal that Stop is now waiting on the request.
			require.Eventually(t, func() bool {
				conn, err := net.Dial("tcp", addr)
				if err == nil {
					conn.Close()
				}
				return err != nil
			}, testtime.Second, testtime.Millisecond, "Stop must close the listener")

			releaseSeq := seq.Inc()
			h.releaseAll()
			res := <-resCh
			require.NoError(t, res.err, "in-flight request must complete")
			assert.Equal(t, http.StatusOK, res.status)
			assert.Equal(t, "payload", res.body)

			select {
			case err := <-stopErr:
				assert.NoError(t, err)
			case <-time.After(5 * testtime.Second):
				require.FailNow(t, "Stop did not return after the request drained")
			}
			assert.Less(t, releaseSeq, stopSeq.Load(), "Stop returned while the request was still in flight")
		})
	}
}

// blockingHandler is a unary handler that blocks every call until released,
// then echoes the request body back.
type blockingHandler struct {
	entered     chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

func newBlockingHandler() *blockingHandler {
	return &blockingHandler{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
}

func (h *blockingHandler) Handle(_ context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	h.entered <- struct{}{}
	<-h.release
	_, err := io.Copy(resw, req.Body)
	return err
}

func (h *blockingHandler) releaseAll() { h.releaseOnce.Do(func() { close(h.release) }) }

func (h *blockingHandler) awaitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-h.entered:
	case <-time.After(5 * testtime.Second):
		require.FailNow(t, "handler was not called")
	}
}

// startBlockingInbound starts an inbound on a random local port, serving a
// "block" procedure backed by the returned handler.
func startBlockingInbound(t *testing.T, opts ...InboundOption) (*Inbound, *blockingHandler) {
	h := newBlockingHandler()
	inbound := NewTransport().NewInbound("127.0.0.1:0", opts...)
	inbound.SetRouter(newTestRouter([]transport.Procedure{{
		Name:        "block",
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
	}}))
	require.NoError(t, inbound.Start())
	t.Cleanup(func() { _ = inbound.Stop() })
	// Registered after Stop so it runs first: a pending call must not hold
	// Stop for the whole ShutdownTimeout when a test fails midway.
	t.Cleanup(h.releaseAll)
	return inbound, h
}

// serverConns records the connections an inbound accepts, so that a test can
// drop them all at once, the way exiting the process would.
type serverConns struct {
	mu    sync.Mutex
	conns []net.Conn
}

func (s *serverConns) record() InboundOption {
	return func(i *Inbound) {
		i.server.ConnState = func(c net.Conn, state http.ConnState) {
			if state == http.StateNew {
				s.mu.Lock()
				s.conns = append(s.conns, c)
				s.mu.Unlock()
			}
		}
	}
}

func (s *serverConns) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		_ = c.Close()
	}
}

func h2cRequestHeaders(procedure string) map[string]string {
	return map[string]string{
		"rpc-caller":     "caller",
		"rpc-service":    "service",
		"rpc-procedure":  procedure,
		"rpc-encoding":   string(raw.Encoding),
		"context-ttl-ms": "10000",
	}
}
