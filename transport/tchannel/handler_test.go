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

package tchannel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/routertest"
	"go.uber.org/yarpc/internal/testtime"
	pkgerrors "go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestHandlerErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		format  tchannel.Format
		headers []byte

		wantHeaders map[string]string
	}{
		{
			format:      tchannel.JSON,
			headers:     []byte(`{"Rpc-Header-Foo": "bar"}`),
			wantHeaders: map[string]string{"rpc-header-foo": "bar"},
		},
		{
			format: tchannel.Thrift,
			headers: []byte{
				0x00, 0x01, // 1 header
				0x00, 0x03, 'F', 'o', 'o', // Foo
				0x00, 0x03, 'B', 'a', 'r', // Bar
			},
			wantHeaders: map[string]string{"foo": "Bar"},
		},
	}

	for _, tt := range tests {
		rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
		router := transporttest.NewMockRouter(mockCtrl)

		spec := transport.NewUnaryHandlerSpec(rpcHandler)
		tchHandler := handler{router: router}

		router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
			WithService("service").
			WithProcedure("hello"),
		).Return(spec, nil)

		rpcHandler.EXPECT().Handle(
			transporttest.NewContextMatcher(t),
			transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:          "caller",
					Service:         "service",
					Headers:         transport.HeadersFromMap(tt.wantHeaders),
					Encoding:        transport.Encoding(tt.format),
					Procedure:       "hello",
					ShardKey:        "shard",
					RoutingKey:      "routekey",
					RoutingDelegate: "routedelegate",
					Body:            bytes.NewReader([]byte("world")),
				}),
			gomock.Any(),
		).Return(nil)

		respRecorder := newResponseRecorder()

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		tchHandler.handle(ctx, &fakeInboundCall{
			service:         "service",
			caller:          "caller",
			format:          tt.format,
			method:          "hello",
			shardkey:        "shard",
			routingkey:      "routekey",
			routingdelegate: "routedelegate",
			arg2:            tt.headers,
			arg3:            []byte("world"),
			resp:            respRecorder,
		})

		assert.NoError(t, respRecorder.systemErr, "did not expect an error")
	}
}

func TestHandlerFailures(t *testing.T) {
	tests := []struct {
		desc string

		// context to use in the callm a default one is used otherwise.
		ctx     context.Context
		ctxFunc func() (context.Context, context.CancelFunc)

		sendCall   *fakeInboundCall
		expectCall func(*transporttest.MockUnaryHandler)

		wantStatus tchannel.SystemErrCode // expected status
	}{
		{
			desc: "no timeout on context",
			ctx:  context.Background(),
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			wantStatus: tchannel.ErrCodeBadRequest,
		},
		{
			desc: "arg2 reader error",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    nil,
				arg3:    []byte{0x00},
			},
			wantStatus: tchannel.ErrCodeBadRequest,
		},
		{
			desc: "arg2 parse error",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.JSON,
				arg2:    []byte("{not valid JSON}"),
				arg3:    []byte{0x00},
			},
			wantStatus: tchannel.ErrCodeBadRequest,
		},
		{
			desc: "arg3 reader error",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    nil,
			},
			wantStatus: tchannel.ErrCodeUnexpected,
		},
		{
			desc: "internal error",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expectCall: func(h *transporttest.MockUnaryHandler) {
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(t, transporttest.ContextTTL(testtime.Second)),
					transporttest.NewRequestMatcher(
						t, &transport.Request{
							Caller:    "bar",
							Service:   "foo",
							Encoding:  raw.Encoding,
							Procedure: "hello",
							Body:      bytes.NewReader([]byte{0x00}),
						},
					), gomock.Any(),
				).Return(fmt.Errorf("great sadness"))
			},
			wantStatus: tchannel.ErrCodeUnexpected,
		},
		{
			desc: "arg3 encode error",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.JSON,
				arg2:    []byte("{}"),
				arg3:    []byte("{}"),
			},
			expectCall: func(h *transporttest.MockUnaryHandler) {
				req := &transport.Request{
					Caller:    "bar",
					Service:   "foo",
					Encoding:  json.Encoding,
					Procedure: "hello",
					Body:      bytes.NewReader([]byte("{}")),
				}
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(t, transporttest.ContextTTL(testtime.Second)),
					transporttest.NewRequestMatcher(t, req),
					gomock.Any(),
				).Return(
					pkgerrors.ResponseBodyEncodeError(req, errors.New(
						"serialization derp",
					)))
			},
			wantStatus: tchannel.ErrCodeBadRequest,
		},
		{
			desc: "handler timeout",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), testtime.Millisecond)
			},
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "waituntiltimeout",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expectCall: func(h *transporttest.MockUnaryHandler) {
				req := &transport.Request{
					Service:   "foo",
					Caller:    "bar",
					Procedure: "waituntiltimeout",
					Encoding:  raw.Encoding,
					Body:      bytes.NewReader([]byte{0x00}),
				}
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(
						t, transporttest.ContextTTL(testtime.Millisecond)),
					transporttest.NewRequestMatcher(t, req),
					gomock.Any(),
				).Do(func(ctx context.Context, _ *transport.Request, _ transport.ResponseWriter) {
					<-ctx.Done()
				}).Return(context.DeadlineExceeded)
			},
			wantStatus: tchannel.ErrCodeTimeout,
		},
		{
			desc: "handler panic",
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "panic",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expectCall: func(h *transporttest.MockUnaryHandler) {
				req := &transport.Request{
					Service:   "foo",
					Caller:    "bar",
					Procedure: "panic",
					Encoding:  raw.Encoding,
					Body:      bytes.NewReader([]byte{0x00}),
				}
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(
						t, transporttest.ContextTTL(testtime.Second)),
					transporttest.NewRequestMatcher(t, req),
					gomock.Any(),
				).Do(func(context.Context, *transport.Request, transport.ResponseWriter) {
					panic("oops I panicked!")
				})
			},
			wantStatus: tchannel.ErrCodeUnexpected,
		},
	}

	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		if tt.ctx != nil {
			ctx = tt.ctx
		} else if tt.ctxFunc != nil {
			ctx, cancel = tt.ctxFunc()
		}
		defer cancel()

		mockCtrl := gomock.NewController(t)
		thandler := transporttest.NewMockUnaryHandler(mockCtrl)
		spec := transport.NewUnaryHandlerSpec(thandler)

		if tt.expectCall != nil {
			tt.expectCall(thandler)
		}

		resp := newResponseRecorder()
		tt.sendCall.resp = resp

		router := transporttest.NewMockRouter(mockCtrl)
		router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
			WithService(tt.sendCall.service).
			WithProcedure(tt.sendCall.method),
		).Return(spec, nil).AnyTimes()

		handler{router: router}.handle(ctx, tt.sendCall)
		err := resp.systemErr
		require.Error(t, err, "expected error for %q", tt.desc)

		systemErr, isSystemErr := err.(tchannel.SystemError)
		require.True(t, isSystemErr, "expected %v for %q to be a system error", err, tt.desc)
		assert.Equal(t, tt.wantStatus, systemErr.Code(), tt.desc)

		mockCtrl.Finish()
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		format           tchannel.Format
		apply            func(*responseWriter)
		arg2             []byte
		arg3             []byte
		applicationError bool
	}{
		{
			format: tchannel.Raw,
			apply: func(w *responseWriter) {
				headers := transport.HeadersFromMap(map[string]string{"foo": "bar"})
				w.AddHeaders(headers)
				_, err := w.Write([]byte("hello "))
				require.NoError(t, err)
				_, err = w.Write([]byte("world"))
				require.NoError(t, err)
			},
			arg2: []byte{
				0x00, 0x01,
				0x00, 0x03, 'f', 'o', 'o',
				0x00, 0x03, 'b', 'a', 'r',
			},
			arg3: []byte("hello world"),
		},
		{
			format: tchannel.Raw,
			apply: func(w *responseWriter) {
				_, err := w.Write([]byte("foo"))
				require.NoError(t, err)
				_, err = w.Write([]byte("bar"))
				require.NoError(t, err)
			},
			arg2: []byte{0x00, 0x00},
			arg3: []byte("foobar"),
		},
		{
			format: tchannel.JSON,
			apply: func(w *responseWriter) {
				headers := transport.HeadersFromMap(map[string]string{"foo": "bar"})
				w.AddHeaders(headers)

				_, err := w.Write([]byte("{"))
				require.NoError(t, err)

				_, err = w.Write([]byte("}"))
				require.NoError(t, err)
			},
			arg2: []byte(`{"foo":"bar"}` + "\n"),
			arg3: []byte("{}"),
		},
		{
			format: tchannel.JSON,
			apply: func(w *responseWriter) {
				_, err := w.Write([]byte("{"))
				require.NoError(t, err)

				_, err = w.Write([]byte("}"))
				require.NoError(t, err)
			},
			arg2: []byte("{}\n"),
			arg3: []byte("{}"),
		},
		{
			format: tchannel.Raw,
			apply: func(w *responseWriter) {
				w.SetApplicationError(errors.New("application_error"))
				_, err := w.Write([]byte("hello"))
				require.NoError(t, err)
			},
			arg2:             []byte{0x00, 0x00},
			arg3:             []byte("hello"),
			applicationError: true,
		},
	}

	for _, tt := range tests {
		call := &fakeInboundCall{format: tt.format}
		resp := newResponseRecorder()
		call.resp = resp

		w := newResponseWriter(call.Response(), call.Format())
		tt.apply(w)
		assert.NoError(t, w.Close())

		assert.Nil(t, resp.systemErr)
		assert.Equal(t, tt.arg2, resp.arg2.Bytes())
		assert.Equal(t, tt.arg3, resp.arg3.Bytes())

		if tt.applicationError {
			assert.True(t, resp.applicationError, "expected an application error")
		}
	}
}

func TestResponseWriterFailure(t *testing.T) {
	tests := []struct {
		setupResp func(*responseRecorder)
		messages  []string
	}{
		{
			setupResp: func(rr *responseRecorder) {
				rr.arg2 = nil
			},
			messages: []string{"no arg2 provided"},
		},
		{
			setupResp: func(rr *responseRecorder) {
				rr.arg3 = nil
			},
			messages: []string{"no arg3 provided"},
		},
	}

	for _, tt := range tests {
		resp := newResponseRecorder()
		tt.setupResp(resp)

		w := newResponseWriter(resp, tchannel.Raw)
		_, err := w.Write([]byte("foo"))
		assert.NoError(t, err)
		_, err = w.Write([]byte("bar"))
		assert.NoError(t, err)
		err = w.Close()
		assert.Error(t, err)
		for _, msg := range tt.messages {
			assert.Contains(t, err.Error(), msg)
		}
	}
}

func TestResponseWriterEmptyBodyHeaders(t *testing.T) {
	res := newResponseRecorder()
	w := newResponseWriter(res, tchannel.Raw)

	w.AddHeaders(transport.NewHeaders().With("foo", "bar"))
	require.NoError(t, w.Close())

	assert.NotEmpty(t, res.arg2.Bytes(), "headers must not be empty")
	assert.Empty(t, res.arg3.Bytes(), "body must be empty but was %#v", res.arg3.Bytes())
	assert.False(t, res.applicationError, "application error must be false")
}

func TestGetSystemError(t *testing.T) {
	tests := []struct {
		giveErr  error
		wantCode tchannel.SystemErrCode
	}{
		{
			giveErr:  yarpcerrors.UnavailableErrorf("test"),
			wantCode: tchannel.ErrCodeDeclined,
		},
		{
			giveErr:  errors.New("test"),
			wantCode: tchannel.ErrCodeUnexpected,
		},
		{
			giveErr:  yarpcerrors.InvalidArgumentErrorf("test"),
			wantCode: tchannel.ErrCodeBadRequest,
		},
		{
			giveErr:  tchannel.NewSystemError(tchannel.ErrCodeBusy, "test"),
			wantCode: tchannel.ErrCodeBusy,
		},
		{
			giveErr:  yarpcerrors.Newf(yarpcerrors.Code(1235), "test"),
			wantCode: tchannel.ErrCodeUnexpected,
		},
	}
	for i, tt := range tests {
		t.Run(string(i), func(t *testing.T) {
			gotErr := getSystemError(tt.giveErr)
			tchErr, ok := gotErr.(tchannel.SystemError)
			require.True(t, ok, "did not return tchannel error")
			assert.Equal(t, tt.wantCode, tchErr.Code())
		})
	}
}
