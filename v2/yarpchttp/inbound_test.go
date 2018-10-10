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

package yarpchttp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalyarpctest"
	"go.uber.org/yarpc/v2/internal/routertest"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestStartAddrInUse(t *testing.T) {
	i1 := &Inbound{
		Addr:   ":0",
		Router: newTestRouter(nil),
	}

	require.NoError(t, i1.Start(context.Background()), "inbound 1 must start without an error")
	i2 := &Inbound{
		Addr:   i1.Listener.Addr().String(),
		Router: newTestRouter(nil),
	}
	err := i2.Start(context.Background())
	require.Error(t, err)

	oe, ok := err.(*net.OpError)
	assert.True(t, ok && oe.Op == "listen", "expected a listen error")
	if ok {
		se, ok := oe.Err.(*os.SyscallError)
		assert.True(t, ok && se.Syscall == "bind" && se.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}

	assert.NoError(t, i1.Stop(context.Background()))
}

func TestInboundStartAndStop(t *testing.T) {
	i := &Inbound{
		Addr:   ":0",
		Router: newTestRouter(nil),
	}
	require.NoError(t, i.Start(context.Background()))
	assert.NotEqual(t, ":0", i.Listener.Addr().String())
	assert.NoError(t, i.Stop(context.Background()))
}

func TestInboundStartError(t *testing.T) {
	i := &Inbound{
		Addr:   "invalid",
		Router: new(yarpctest.MockRouter),
	}
	err := i.Start(context.Background())
	assert.Error(t, err, "expected failure")
}

func TestInboundStartErrorBadGrabHeader(t *testing.T) {
	i := &Inbound{
		Addr:        ":0",
		Router:      new(yarpctest.MockRouter),
		GrabHeaders: []string{"x-valid", "y-invalid"},
	}
	assert.Equal(t, yarpcerror.CodeInvalidArgument, yarpcerror.FromError(i.Start(context.Background())).Code())
}

func TestInboundMux(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, httpReq *http.Request) {
		w.Write([]byte("healthy"))
	})

	router := yarpctest.NewMockRouter(mockCtrl)
	router.EXPECT().Procedures()
	i := &Inbound{
		Addr:       ":0",
		Router:     router,
		Mux:        mux,
		MuxPattern: "/rpc/v1",
	}
	require.NoError(t, i.Start(context.Background()))
	defer i.Stop(context.Background())

	url := fmt.Sprintf("http://%v/", internalyarpctest.ZeroAddrToHostPort(i.Listener.Addr()))
	resp, err := http.Get(url + "health")
	if assert.NoError(t, err, "/health failed") {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if assert.NoError(t, err, "/health body read error") {
			assert.Equal(t, "healthy", string(body), "/health body mismatch")
		}
	}

	// this should fail
	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer dialer.Stop(context.Background())
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(url),
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, _, err = outbound.Call(ctx, &yarpc.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  yarpc.Encoding("raw"),
	}, yarpc.NewBufferString("derp"))

	if assert.Error(t, err, "RPC call to / should have failed") {
		assert.Equal(t, yarpcerror.CodeNotFound, yarpcerror.FromError(err).Code())
	}

	host := outbound.URL.Host
	outbound.URL = parseURL("http://host:port/rpc/v1")
	outbound.URL.Host = host

	h := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	spec := yarpc.NewUnaryTransportHandlerSpec(h)
	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithCaller("foo").
		WithService("bar").
		WithProcedure("hello"),
	).Return(spec, nil)

	h.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(&yarpc.Response{}, &yarpc.Buffer{}, nil)

	response, responseBuf, err := outbound.Call(ctx, &yarpc.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  yarpc.Encoding("raw"),
	}, yarpc.NewBufferString("derp"))

	require.NoError(t, err, "expected rpc request to succeed")
	assert.NotNil(t, response)

	s, err := ioutil.ReadAll(responseBuf)
	require.NoError(t, err)
	assert.Empty(t, s)
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
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "OK")
	})
	intercept := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, "intercepted")
		})
	}

	inbound := &Inbound{
		Addr:        "127.0.0.1:0",
		Router:      newTestRouter(nil),
		Mux:         mux,
		Interceptor: intercept,
	}
	require.NoError(t, inbound.Start(context.Background()), "Failed to start inbound")
	defer inbound.Stop(context.Background())

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			url := fmt.Sprintf("http://%v%v", inbound.Listener.Addr(), tt.path)
			_, body, err := httpGet(t, url)
			require.NoError(t, err, "request failed")
			assert.Equal(t, tt.want, string(body))
		})
	}
}

func TestRequestAfterStop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "OK")
	})

	inbound := Inbound{
		Addr:   "127.0.0.1:0",
		Router: newTestRouter(nil),
		Mux:    mux,
	}
	require.NoError(t, inbound.Start(context.Background()), "Failed to start inbound")

	url := fmt.Sprintf("http://%v/health", inbound.Listener.Addr())
	_, body, err := httpGet(t, url)
	require.NoError(t, err, "expect successful response")
	assert.Equal(t, "OK", body, "response mismatch")

	require.NoError(t, inbound.Stop(context.Background()), "Failed to stop inbound")

	_, _, err = httpGet(t, url)
	assert.Error(t, err, "requests should fail once inbound is stopped")
}

func httpGet(t *testing.T, url string) (*http.Response, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("GET %v failed: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to read reponse from %v: %v", url, err)
	}

	return resp, string(body), nil
}
