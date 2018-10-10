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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalyarpctest"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	headers := make(http.Header)
	headers.Set(CallerHeader, "moe")
	headers.Set(EncodingHeader, "raw")
	headers.Set(TTLMSHeader, "1000")
	headers.Set(ProcedureHeader, "nyuck")
	headers.Set(ServiceHeader, "curly")
	headers.Set(ShardKeyHeader, "shard")
	headers.Set(RoutingKeyHeader, "routekey")
	headers.Set(RoutingDelegateHeader, "routedelegate")

	router := yarpctest.NewMockRouter(mockCtrl)
	rpcHandler := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	spec := yarpc.NewUnaryTransportHandlerSpec(rpcHandler)

	router.EXPECT().Choose(gomock.Any(), internalyarpctest.NewMatcher().
		WithService("curly").
		WithProcedure("nyuck"),
	).Return(spec, nil)

	rpcHandler.EXPECT().Handle(
		yarpctest.NewContextMatcher(t, yarpctest.ContextTTL(time.Second)),
		&yarpc.Request{
			Caller:          "moe",
			Service:         "curly",
			Transport:       "http",
			Encoding:        yarpc.Encoding("raw"),
			Procedure:       "nyuck",
			ShardKey:        "shard",
			RoutingKey:      "routekey",
			RoutingDelegate: "routedelegate",
		},
		yarpc.NewBufferString("Nyuck Nyuck"),
	).Return(&yarpc.Response{}, &yarpc.Buffer{}, nil)

	httpHandler := handler{router: router, tracer: &opentracing.NoopTracer{}}
	httpReq := &http.Request{
		Method: "POST",
		Header: headers,
		Body:   ioutil.NopCloser(bytes.NewReader([]byte("Nyuck Nyuck"))),
	}
	rw := httptest.NewRecorder()
	httpHandler.ServeHTTP(rw, httpReq)

	assert.Equal(t, rw.Code, 200, "expected 200 code")
	assert.Empty(t, rw.Body.String())
}

func TestHandlerHeaders(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		giveEncoding string
		giveHeaders  http.Header
		grabHeaders  map[string]struct{}

		wantTTL     time.Duration
		wantHeaders map[string]string
	}{
		{
			giveEncoding: "json",
			giveHeaders: http.Header{
				TTLMSHeader:      {"1000"},
				"Rpc-Header-Foo": {"bar"},
				"X-Baz":          {"bat"},
			},
			grabHeaders: map[string]struct{}{"x-baz": {}},
			wantTTL:     time.Second,
			wantHeaders: map[string]string{
				"Foo":   "bar",
				"x-baz": "bat",
			},
		},
		{
			giveEncoding: "raw",
			giveHeaders: http.Header{
				TTLMSHeader: {"100"},
				"Rpc-Foo":   {"ignored"},
			},
			wantTTL:     100 * time.Millisecond,
			wantHeaders: map[string]string{},
		},
		{
			giveEncoding: "thrift",
			giveHeaders: http.Header{
				TTLMSHeader: {"1000"},
			},
			wantTTL:     time.Second,
			wantHeaders: map[string]string{},
		},
		{
			giveEncoding: "proto",
			giveHeaders: http.Header{
				TTLMSHeader: {"1000"},
			},
			wantTTL:     time.Second,
			wantHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.giveEncoding, func(*testing.T) {
			router := yarpctest.NewMockRouter(mockCtrl)
			rpcHandler := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
			spec := yarpc.NewUnaryTransportHandlerSpec(rpcHandler)

			router.EXPECT().Choose(gomock.Any(), internalyarpctest.NewMatcher().
				WithService("service").
				WithProcedure("hello"),
			).Return(spec, nil)

			httpHandler := handler{router: router, tracer: &opentracing.NoopTracer{}, grabHeaders: tt.grabHeaders}

			rpcHandler.EXPECT().Handle(
				yarpctest.NewContextMatcher(t,
					yarpctest.ContextTTL(tt.wantTTL),
				),
				&yarpc.Request{
					Caller:    "caller",
					Service:   "service",
					Transport: "http",
					Encoding:  yarpc.Encoding(tt.giveEncoding),
					Procedure: "hello",
					Headers:   yarpc.HeadersFromMap(tt.wantHeaders),
				},
				yarpc.NewBufferString("world"),
			).Return(&yarpc.Response{}, &yarpc.Buffer{}, nil)

			headers := http.Header{}
			for k, vs := range tt.giveHeaders {
				for _, v := range vs {
					headers.Add(k, v)
				}
			}
			headers.Set(CallerHeader, "caller")
			headers.Set(ServiceHeader, "service")
			headers.Set(EncodingHeader, tt.giveEncoding)
			headers.Set(ProcedureHeader, "hello")

			httpReq := &http.Request{
				Method: "POST",
				Header: headers,
				Body:   ioutil.NopCloser(bytes.NewReader([]byte("world"))),
			}
			rw := httptest.NewRecorder()
			httpHandler.ServeHTTP(rw, httpReq)
			assert.Equal(t, 200, rw.Code, "expected 200 status code")
			assert.Equal(t, getContentType(yarpc.Encoding(tt.giveEncoding)), rw.HeaderMap.Get("Content-Type"))
		})
	}
}

func TestHandlerFailures(t *testing.T) {
	t.Skip() // TODO restore validator middleware

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	service, procedure := "fake", "hello"

	baseHeaders := make(http.Header)
	baseHeaders.Set(CallerHeader, "somecaller")
	baseHeaders.Set(EncodingHeader, "raw")
	baseHeaders.Set(TTLMSHeader, "1000")
	baseHeaders.Set(ProcedureHeader, procedure)
	baseHeaders.Set(ServiceHeader, service)

	headersWithBadTTL := headerCopyWithout(baseHeaders, TTLMSHeader)
	headersWithBadTTL.Set(TTLMSHeader, "not a number")

	tests := []struct {
		msg     string
		httpReq *http.Request

		// if we expect an error as a result of the TTL
		errTTL   bool
		wantCode yarpcerror.Code
	}{
		{
			msg:      "get root not found",
			httpReq:  &http.Request{Method: "GET"},
			wantCode: yarpcerror.CodeNotFound,
		},
		{
			msg: "post without call header",
			httpReq: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, CallerHeader),
			},
			wantCode: yarpcerror.CodeInvalidArgument,
		},
		{
			msg: "post without service header",
			httpReq: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ServiceHeader),
			},
			wantCode: yarpcerror.CodeInvalidArgument,
		},
		{
			msg: "post without procedure header",
			httpReq: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ProcedureHeader),
			},
			wantCode: yarpcerror.CodeInvalidArgument,
		},
		{
			msg: "post without timeout header",
			httpReq: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, TTLMSHeader),
			},
			wantCode: yarpcerror.CodeInvalidArgument,
			errTTL:   true,
		},
		{
			msg: "post without headers",
			httpReq: &http.Request{
				Method: "POST",
			},
			wantCode: yarpcerror.CodeInvalidArgument,
		},
		{
			msg: "post with bad timeout",
			httpReq: &http.Request{
				Method: "POST",
				Header: headersWithBadTTL,
			},
			wantCode: yarpcerror.CodeInvalidArgument,
			errTTL:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			httpReq := tt.httpReq
			if httpReq.Body == nil {
				httpReq.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
			}

			router := yarpctest.NewMockRouter(mockCtrl)

			if tt.errTTL {
				// since TTL is checked after we've determined the transport type, if we have an
				// error with TTL it will be discovered after we read from the router
				spec := yarpc.NewUnaryTransportHandlerSpec(panickedHandler{})
				router.EXPECT().Choose(gomock.Any(), internalyarpctest.NewMatcher().
					WithService(service).
					WithProcedure(procedure),
				).Return(spec, nil)
			}

			h := handler{router: router, tracer: &opentracing.NoopTracer{}}

			rw := httptest.NewRecorder()
			h.ServeHTTP(rw, tt.httpReq)

			httpStatusCode := rw.Code
			assert.True(t, httpStatusCode >= 400 && httpStatusCode < 500, "expected 400 level code, got %d", httpStatusCode)
			code := statusCodeToBestCode(httpStatusCode)
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, "text/plain; charset=utf8", rw.HeaderMap.Get("Content-Type"))
			io.Copy(os.Stdout, rw.Body)
		})
	}
}

func TestHandlerInternalFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	headers := make(http.Header)
	headers.Set(CallerHeader, "somecaller")
	headers.Set(EncodingHeader, "raw")
	headers.Set(TTLMSHeader, "1000")
	headers.Set(ProcedureHeader, "hello")
	headers.Set(ServiceHeader, "fake")

	httpReq := http.Request{
		Method: "POST",
		Header: headers,
		Body:   ioutil.NopCloser(bytes.NewReader([]byte{})),
	}

	rpcHandler := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	rpcHandler.EXPECT().Handle(
		yarpctest.NewContextMatcher(t, yarpctest.ContextTTL(time.Second)),
		&yarpc.Request{
			Caller:    "somecaller",
			Service:   "fake",
			Transport: "http",
			Encoding:  yarpc.Encoding("raw"),
			Procedure: "hello",
		},
		gomock.Any(),
	).Return(nil, nil, fmt.Errorf("great sadness"))

	router := yarpctest.NewMockRouter(mockCtrl)
	spec := yarpc.NewUnaryTransportHandlerSpec(rpcHandler)

	router.EXPECT().Choose(gomock.Any(), internalyarpctest.NewMatcher().
		WithService("fake").
		WithProcedure("hello"),
	).Return(spec, nil)

	httpHandler := handler{router: router, tracer: &opentracing.NoopTracer{}}
	responseRecorder := httptest.NewRecorder()
	httpHandler.ServeHTTP(responseRecorder, &httpReq)

	code := responseRecorder.Code
	assert.True(t, code >= 500 && code < 600, "expected 500 level response, got '%d'", code)
	assert.Equal(t,
		`error for service "fake" and procedure "hello": great sadness`, responseRecorder.Body.String())
}

type panickedHandler struct{}

func (th panickedHandler) Handle(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	panic("oops I panicked!")
}

func TestHandlerPanic(t *testing.T) {
	dialer := &Dialer{}
	dialer.Start(context.Background())
	defer dialer.Stop(context.Background())

	router := yarpcrouter.NewMapRouter("yarpc-test")
	router.Register([]yarpc.Procedure{
		{
			Name:        "panic",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(panickedHandler{}),
		},
	})
	inbound := &Inbound{
		Addr:   "localhost:0",
		Router: router,
	}
	require.NoError(t, inbound.Start(context.Background()))
	defer inbound.Stop(context.Background())

	outbound := &Outbound{
		Dialer: dialer,
		URL: &url.URL{
			Scheme: "https",
			Host:   inbound.Listener.Addr().String(),
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, _, err := outbound.Call(ctx, &yarpc.Request{
		Caller:    "yarpc-test",
		Service:   "yarpc-test",
		Procedure: "panic",
	}, &yarpc.Buffer{})
	defer cancel()

	assert.Equal(t, yarpcerror.CodeUnknown, yarpcerror.FromError(err).Code())
}

func headerCopyWithout(headers http.Header, names ...string) http.Header {
	newHeaders := make(http.Header)
	for k, vs := range headers {
		for _, v := range vs {
			newHeaders.Add(k, v)
		}
	}

	for _, k := range names {
		newHeaders.Del(k)
	}

	return newHeaders
}

func TestResponseWriter(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := newResponseWriter(recorder)

	response := &yarpc.Response{
		Headers: yarpc.HeadersFromMap(map[string]string{
			"foo":       "bar",
			"shard-key": "123",
		}),
	}

	rw.SetResponse(response, yarpc.NewBufferString("hello"))
	rw.Close(http.StatusOK)

	assert.Equal(t, "bar", recorder.Header().Get("rpc-header-foo"))
	assert.Equal(t, "123", recorder.Header().Get("rpc-header-shard-key"))
	assert.Equal(t, "hello", recorder.Body.String())
}
