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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestCallSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, httpReq *http.Request) {
			defer httpReq.Body.Close()

			ttl := httpReq.Header.Get(TTLMSHeader)
			ttlms, err := strconv.Atoi(ttl)
			assert.NoError(t, err, "can parse TTL header")
			assert.InDelta(t, ttlms, testtime.X*1000.0, testtime.X*5.0, "ttl header within tolerance")

			assert.Equal(t, "caller", httpReq.Header.Get(CallerHeader))
			assert.Equal(t, "service", httpReq.Header.Get(ServiceHeader))
			assert.Equal(t, "raw", httpReq.Header.Get(EncodingHeader))
			assert.Equal(t, "hello", httpReq.Header.Get(ProcedureHeader))

			body, err := ioutil.ReadAll(httpReq.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("world"), body)
			}

			w.Header().Set("rpc-header-foo", "bar")
			_, err = w.Write([]byte("great success"))
			assert.NoError(t, err)
		},
	))
	defer successServer.Close()

	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(successServer.URL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	response, responseBuf, err := outbound.Call(ctx, &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "hello",
	}, yarpc.NewBufferString("world"))
	require.NoError(t, err)

	foo, ok := response.Headers.Get("foo")
	assert.True(t, ok, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(responseBuf)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("great success"), body)
	}
}

func TestAddReservedHeader(t *testing.T) {
	assert.Panics(t, func() {
		// TODO pare this down
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		dialer := &Dialer{}
		dialer.Start(ctx)
		defer dialer.Stop(ctx)

		response, responseBuf, err := (&Outbound{
			Dialer: dialer,
			URL:    &url.URL{Host: "localhost:8080"},
			Headers: http.Header{
				"RPC-WAT":             []string{},
				"Rpc-Not-On-My-Watch": []string{},
				"rpc-i-dont-think-so": []string{},
			},
		}).Call(ctx, &yarpc.Request{}, &yarpc.Buffer{})
		require.Nil(t, response)
		require.Nil(t, responseBuf)
		require.NoError(t, err)
	})
}

func TestOutboundHeaders(t *testing.T) {
	tests := []struct {
		desc     string
		context  context.Context
		headers  yarpc.Headers
		outbound *Outbound

		wantHeaders map[string]string
	}{
		{
			desc:    "application headers",
			headers: yarpc.NewHeaders().With("foo", "bar").With("baz", "Qux"),
			wantHeaders: map[string]string{
				"Rpc-Header-Foo": "bar",
				"Rpc-Header-Baz": "Qux",
			},
			outbound: &Outbound{},
		},
		{
			desc:    "extra headers",
			headers: yarpc.NewHeaders().With("x", "y"),
			outbound: &Outbound{
				Headers: http.Header{
					"X-Foo": []string{"bar"},
					"X-Bar": []string{"BAZ"},
				},
			},
			wantHeaders: map[string]string{
				"Rpc-Header-X": "y",
				"X-Foo":        "bar",
				"X-Bar":        "BAZ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			dialer := &Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			defer func() {
				require.NoError(t, dialer.Stop(context.Background()))
			}()

			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, httpReq *http.Request) {
					defer httpReq.Body.Close()
					for k, v := range tt.wantHeaders {
						assert.Equal(
							t, v, httpReq.Header.Get(k), "%v: header %v did not match", tt.desc, k)
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

			outbound := &*tt.outbound
			outbound.Dialer = dialer
			outbound.URL = parseURL(server.URL)

			response, responseBuf, err := outbound.Call(ctx, &yarpc.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  yarpc.Encoding("raw"),
				Headers:   tt.headers,
				Procedure: "hello",
			}, yarpc.NewBufferString("world"))

			assert.NoError(t, err, "%v: call failed", tt.desc)
			assert.NotNil(t, response)
			assert.NotNil(t, responseBuf)
		})
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

	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, httpReq *http.Request) {
					w.Header().Add("Rpc-Status", tt.status)
					defer httpReq.Body.Close()
				},
			))
			defer server.Close()

			outbound := &Outbound{Dialer: dialer, URL: parseURL(server.URL)}

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
			defer cancel()

			response, _, err := outbound.Call(ctx, &yarpc.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  yarpc.Encoding("raw"),
				Procedure: "hello",
			}, yarpc.NewBufferString("world"))

			assert.Equal(t, response.ApplicationError, tt.appError, "%v: application status", tt.desc)
			assert.NoError(t, err, "%v: call failed", tt.desc)
		})
	}
}

func TestCallFailures(t *testing.T) {
	notFoundServer := httptest.NewServer(http.NotFoundHandler())
	defer notFoundServer.Close()

	internalErrorServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, httpReq *http.Request) {
			http.Error(w, "great sadness", http.StatusInternalServerError)
		}))
	defer internalErrorServer.Close()

	tests := []struct {
		url      string
		messages []string
	}{
		{"not a URL", []string{"protocol scheme"}},
		{notFoundServer.URL, []string{"404", "page not found"}},
		{internalErrorServer.URL, []string{"great sadness"}},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			dialer := &Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			defer func() {
				require.NoError(t, dialer.Stop(context.Background()))
			}()

			outbound := &Outbound{Dialer: dialer, URL: parseURL(tt.url)}

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()
			_, _, err := outbound.Call(ctx, &yarpc.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  yarpc.Encoding("raw"),
				Procedure: "wat",
			}, yarpc.NewBufferString("huh"))
			assert.Error(t, err, "expected failure")
			for _, msg := range tt.messages {
				assert.Contains(t, err.Error(), msg)
			}
		})
	}
}

func TestGetPeerForRequestErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name string
		peer *yarpctest.MockPeer
		err  error
	}{
		{
			name: "error choosing peer",
		},
		{
			name: "error casting peer",
			peer: yarpctest.NewMockPeer(ctrl),
			err:  errors.New("err"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chooser := yarpctest.NewMockChooser(ctrl)

			dialer := &Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			defer func() {
				require.NoError(t, dialer.Stop(context.Background()))
			}()
			outbound := &Outbound{
				Chooser: chooser,
				URL:     &url.URL{Host: "127.0.0.1:9999"},
			}

			ctx := context.Background()
			req := &yarpc.Request{}

			chooser.EXPECT().Choose(ctx, req).Return(tt.peer, nil, tt.err)

			_, _, err := outbound.getPeerForRequest(ctx, req)
			require.Error(t, err)
		})
	}
}

func TestWithCoreHeaders(t *testing.T) {
	addr := "http://127.0.0.1:9999"
	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(addr),
	}

	httpReq := httptest.NewRequest("", addr, nil)

	shardKey := "sharding"
	routingKey := "routing"
	routingDelegate := "delegate"

	req := &yarpc.Request{
		ShardKey:        shardKey,
		RoutingKey:      routingKey,
		RoutingDelegate: routingDelegate,
	}
	result := outbound.withCoreHeaders(httpReq, req, time.Second)

	assert.Equal(t, shardKey, result.Header.Get(ShardKeyHeader))
	assert.Equal(t, routingKey, result.Header.Get(RoutingKeyHeader))
	assert.Equal(t, routingDelegate, result.Header.Get(RoutingDelegateHeader))
}

func TestNoRequest(t *testing.T) {
	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    &url.URL{Host: "localhost:0"},
	}

	_, _, err := outbound.Call(context.Background(), nil, &yarpc.Buffer{})
	assert.Equal(t, yarpcerror.InvalidArgumentErrorf("request for http unary outbound was nil"), err)
}

func TestOutboundNoDeadline(t *testing.T) {
	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    &url.URL{Host: "foo-host:8080"},
	}

	_, _, err := outbound.call(context.Background(), &yarpc.Request{}, &yarpc.Buffer{})
	assert.Equal(t, yarpcerror.Newf(yarpcerror.CodeInvalidArgument, "missing context deadline"), err)
}

func TestServiceMatchSuccess(t *testing.T) {
	matchServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, httpReq *http.Request) {
			defer httpReq.Body.Close()
			w.Header().Set(ServiceHeader, httpReq.Header.Get(ServiceHeader))
			_, err := w.Write([]byte("Service name header return"))
			assert.NoError(t, err)
		},
	))
	defer matchServer.Close()

	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(matchServer.URL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, _, err := outbound.Call(ctx, &yarpc.Request{Service: "Service"}, &yarpc.Buffer{})
	require.NoError(t, err)
}

func TestServiceMatchFailed(t *testing.T) {
	mismatchServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, httpReq *http.Request) {
			defer httpReq.Body.Close()
			w.Header().Set(ServiceHeader, "ThisIsAWrongSvcName")
			_, err := w.Write([]byte("Wrong service name header return"))
			assert.NoError(t, err)
		},
	))
	defer mismatchServer.Close()

	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(mismatchServer.URL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, _, err := outbound.Call(ctx, &yarpc.Request{Service: "Service"}, &yarpc.Buffer{})
	assert.Error(t, err, "expected failure for service name dismatch")
}

func TestServiceMatchNoHeader(t *testing.T) {
	noHeaderServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, httpReq *http.Request) {
			defer httpReq.Body.Close()
			// intentionally do not set a response header
			_, err := w.Write([]byte("No service name header return"))
			assert.NoError(t, err)
		},
	))
	defer noHeaderServer.Close()

	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer func() {
		require.NoError(t, dialer.Stop(context.Background()))
	}()
	outbound := &Outbound{
		Dialer: dialer,
		URL:    parseURL(noHeaderServer.URL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, _, err := outbound.Call(ctx, &yarpc.Request{Service: "Service"}, &yarpc.Buffer{})
	require.NoError(t, err)
}

func parseURL(in string) *url.URL {
	out, _ := url.Parse(in)
	return out
}
