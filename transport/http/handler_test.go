// Copyright (c) 2016 Uber Technologies, Inc.
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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	yarpc "go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestHandlerSucces(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	headers := make(http.Header)
	headers.Set(CallerHeader, "moe")
	headers.Set(EncodingHeader, "raw")
	headers.Set(TTLMSHeader, "1000")
	headers.Set(ProcedureHeader, "nyuck")
	headers.Set(ServiceHeader, "curly")

	rpcHandler := transporttest.NewMockHandler(mockCtrl)
	registry := transporttest.NewMockRegistry(mockCtrl)

	registry.EXPECT().GetHandler("curly", "nyuck").Return(rpcHandler, nil)
	rpcHandler.EXPECT().Handle(
		transporttest.NewContextMatcher(t,
			transporttest.ContextTTL(time.Second),
		),
		transport.Options{},
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:    "moe",
				Service:   "curly",
				Encoding:  raw.Encoding,
				Procedure: "nyuck",
				Body:      bytes.NewReader([]byte("Nyuck Nyuck")),
			},
		),
		gomock.Any(),
	).Return(nil)

	httpHandler := handler{Registry: registry}
	req := &http.Request{
		Method: "POST",
		Header: headers,
		Body:   ioutil.NopCloser(bytes.NewReader([]byte("Nyuck Nyuck"))),
	}
	rw := httptest.NewRecorder()
	httpHandler.ServeHTTP(rw, req)
	code := rw.Code
	assert.Equal(t, code, 200, "expected 200 code")
	assert.Equal(t, rw.Body.String(), "")
}

func TestHandlerHeaders(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		giveHeaders http.Header

		wantTTL     time.Duration
		wantHeaders map[string]string
	}{
		{
			giveHeaders: http.Header{
				TTLMSHeader:      {"1000"},
				"Rpc-Header-Foo": {"bar"},
			},
			wantTTL: time.Second,
			wantHeaders: map[string]string{
				"foo": "bar",
			},
		},
		{
			giveHeaders: http.Header{
				TTLMSHeader: {"100"},
				"Rpc-Foo":   {"ignored"},
			},
			wantTTL:     100 * time.Millisecond,
			wantHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		rpcHandler := transporttest.NewMockHandler(mockCtrl)
		registry := transporttest.NewMockRegistry(mockCtrl)

		registry.EXPECT().GetHandler("service", "hello").Return(rpcHandler, nil)
		httpHandler := handler{Registry: registry}

		rpcHandler.EXPECT().Handle(
			transporttest.NewContextMatcher(t,
				transporttest.ContextTTL(tt.wantTTL),
			),
			transport.Options{},
			transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  raw.Encoding,
					Procedure: "hello",
					Headers:   transport.HeadersFromMap(tt.wantHeaders),
					Body:      bytes.NewReader([]byte("world")),
				}),
			gomock.Any(),
		).Return(nil)

		headers := http.Header{}
		for k, vs := range tt.giveHeaders {
			for _, v := range vs {
				headers.Add(k, v)
			}
		}
		headers.Set(CallerHeader, "caller")
		headers.Set(ServiceHeader, "service")
		headers.Set(EncodingHeader, "raw")
		headers.Set(ProcedureHeader, "hello")

		req := &http.Request{
			Method: "POST",
			Header: headers,
			Body:   ioutil.NopCloser(bytes.NewReader([]byte("world"))),
		}
		rw := httptest.NewRecorder()
		httpHandler.ServeHTTP(rw, req)
		assert.Equal(t, 200, rw.Code, "expected 200 status code")
	}
}

func TestHandlerFailures(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	baseHeaders := make(http.Header)
	baseHeaders.Set(CallerHeader, "somecaller")
	baseHeaders.Set(EncodingHeader, "raw")
	baseHeaders.Set(TTLMSHeader, "1000")
	baseHeaders.Set(ProcedureHeader, "hello")
	baseHeaders.Set(ServiceHeader, "fake")

	headersWithBadTTL := headerCopyWithout(baseHeaders, TTLMSHeader)
	headersWithBadTTL.Set(TTLMSHeader, "not a number")

	tests := []struct {
		req *http.Request
		msg string
	}{
		{&http.Request{Method: "GET"}, "404 page not found\n"},
		{
			&http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, CallerHeader),
			},
			"BadRequest: missing caller name\n",
		},
		{
			&http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ServiceHeader),
			},
			"BadRequest: missing service name\n",
		},
		{
			&http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ProcedureHeader),
			},
			"BadRequest: missing procedure\n",
		},
		{
			&http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, TTLMSHeader),
			},
			"BadRequest: missing TTL\n",
		},
		{
			&http.Request{
				Method: "POST",
			},
			"BadRequest: missing service name, procedure, caller name, TTL, and encoding\n",
		},
		{
			&http.Request{
				Method: "POST",
				Header: headersWithBadTTL,
			},
			`BadRequest: invalid TTL "not a number" for procedure "hello" of service "fake": must be positive integer` + "\n",
		},
	}

	for _, tt := range tests {
		req := tt.req
		if req.Body == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
		}

		h := handler{Registry: transporttest.NewMockRegistry(mockCtrl)}
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, tt.req)

		code := rw.Code
		assert.True(t, code >= 400 && code < 500, "expected 400 level code")
		assert.Equal(t, rw.Body.String(), tt.msg)
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

	request := http.Request{
		Method: "POST",
		Header: headers,
		Body:   ioutil.NopCloser(bytes.NewReader([]byte{})),
	}

	rpcHandler := transporttest.NewMockHandler(mockCtrl)
	rpcHandler.EXPECT().Handle(
		transporttest.NewContextMatcher(t, transporttest.ContextTTL(time.Second)),
		transport.Options{},
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:    "somecaller",
				Service:   "fake",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				Body:      bytes.NewReader([]byte{}),
			},
		),
		gomock.Any(),
	).Return(fmt.Errorf("great sadness"))

	registry := transporttest.NewMockRegistry(mockCtrl)

	registry.EXPECT().GetHandler("fake", "hello").Return(rpcHandler, nil)
	httpHandler := handler{Registry: registry}
	httpResponse := httptest.NewRecorder()
	httpHandler.ServeHTTP(httpResponse, &request)

	code := httpResponse.Code
	assert.True(t, code >= 500 && code < 600, "expected 500 level response")
	assert.Equal(t,
		`UnexpectedError: error for procedure "hello" of service "fake": great sadness`+"\n",
		httpResponse.Body.String())
}

type panickedHandler struct{}

func (th panickedHandler) Handle(context.Context, transport.Options, *transport.Request, transport.ResponseWriter) error {
	panic("oops I panicked!")
}

func TestHandlerPanic(t *testing.T) {
	inbound := NewInbound("localhost:0")
	serverDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "yarpc-test",
		Inbounds: []transport.Inbound{inbound},
	})
	serverDispatcher.Register([]transport.Registrant{
		{Procedure: "panic", Handler: panickedHandler{}},
	})

	require.NoError(t, serverDispatcher.Start())
	defer serverDispatcher.Stop()

	outbound := NewOutbound(fmt.Sprintf("http://%s", inbound.Addr().String()))
	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:      "yarpc-test-client",
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})
	require.NoError(t, clientDispatcher.Start())
	defer clientDispatcher.Stop()

	client := raw.New(clientDispatcher.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	_, _, err := client.Call(ctx, yarpc.NewReqMeta().Procedure("panic"), []byte{})

	assert.True(t, transport.IsUnexpectedError(err), "Must be an UnexpectedError")
	assert.Equal(t,
		`UnexpectedError: error for procedure "panic" of service "yarpc-test": panic: oops I panicked!`,
		err.Error())
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
	writer := newResponseWriter(recorder)

	headers := transport.HeadersFromMap(map[string]string{
		"foo":       "bar",
		"shard-key": "123",
	})
	writer.AddHeaders(headers)

	_, err := writer.Write([]byte("hello"))
	require.NoError(t, err)

	assert.Equal(t, "bar", recorder.Header().Get("rpc-header-foo"))
	assert.Equal(t, "123", recorder.Header().Get("rpc-header-shard-key"))
	assert.Equal(t, "hello", recorder.Body.String())
}
