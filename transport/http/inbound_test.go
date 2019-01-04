// Copyright (c) 2019 Uber Technologies, Inc.
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
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/routertest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestStartAddrInUse(t *testing.T) {
	t1 := NewTransport()
	i1 := t1.NewInbound(":0")

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
	i := x.NewInbound(":0")
	i.SetRouter(newTestRouter(nil))
	require.NoError(t, i.Start())
	assert.NotEqual(t, ":0", i.Addr().String())
	assert.NotNil(t, i.Addr())
	assert.NoError(t, i.Stop())
	assert.Nil(t, i.Addr())
}

func TestInboundStartAndStop(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound(":0")
	i.SetRouter(newTestRouter(nil))
	require.NoError(t, i.Start())
	assert.NotEqual(t, ":0", i.Addr().String())
	assert.NoError(t, i.Stop())
}

func TestInboundStartError(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound("invalid")
	i.SetRouter(new(transporttest.MockRouter))
	err := i.Start()
	assert.Error(t, err, "expected failure")
}

func TestInboundStartErrorBadGrabHeader(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound(":0", GrabHeaders("x-valid", "y-invalid"))
	i.SetRouter(new(transporttest.MockRouter))
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(i.Start()).Code())
}

func TestInboundStopWithoutStarting(t *testing.T) {
	x := NewTransport()
	i := x.NewInbound(":8000")
	assert.Nil(t, i.Addr())
	assert.NoError(t, i.Stop())
}

func TestInboundMux(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpTransport := NewTransport()
	// TODO transport lifecycle

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	i := httpTransport.NewInbound(":0", Mux("/rpc/v1", mux))
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
		body, err := ioutil.ReadAll(resp.Body)
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
		Body:      bytes.NewReader([]byte("derp")),
	})

	if assert.Error(t, err, "RPC call to / should have failed") {
		assert.Equal(t, yarpcerrors.CodeNotFound, yarpcerrors.FromError(err).Code())
	}

	o.setURLTemplate("http://host:port/rpc/v1")
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
		Body:      bytes.NewReader([]byte("derp")),
	})

	if assert.NoError(t, err, "expected rpc request to succeed") {
		defer res.Body.Close()
		s, err := ioutil.ReadAll(res.Body)
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
