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

package yarpctchannel

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tchannel "github.com/uber/tchannel-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/internal/routertest"
	"go.uber.org/yarpc/v2/yarpcencoding"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpctest"
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
			wantHeaders: map[string]string{"Rpc-Header-Foo": "bar"},
		},
		{
			format: tchannel.Thrift,
			headers: []byte{
				0x00, 0x01, // 1 header
				0x00, 0x03, 'F', 'o', 'o', // Foo
				0x00, 0x03, 'B', 'a', 'r', // Bar
			},
			wantHeaders: map[string]string{"Foo": "Bar"},
		},
	}

	for _, tt := range tests {
		rpcHandler := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
		router := yarpctest.NewMockRouter(mockCtrl)

		spec := yarpc.NewUnaryTransportHandlerSpec(rpcHandler)
		tchHandler := handler{router: router}

		router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
			WithService("service").
			WithProcedure("hello"),
		).Return(spec, nil)

		rpcHandler.EXPECT().Handle(
			yarpctest.NewContextMatcher(t),
			&yarpc.Request{
				Caller:          "caller",
				Service:         "service",
				Transport:       "tchannel",
				Headers:         yarpc.HeadersFromMap(tt.wantHeaders),
				Encoding:        yarpc.Encoding(tt.format),
				Procedure:       "hello",
				ShardKey:        "shard",
				RoutingKey:      "routekey",
				RoutingDelegate: "routedelegate",
			},
			yarpc.NewBufferString("world"),
		).Return(&yarpc.Response{}, yarpc.NewBufferString(""), nil)

		respRecorder := newResponseRecorder()

		ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
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
		expectCall func(*yarpctest.MockUnaryTransportHandler)

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
			expectCall: func(h *yarpctest.MockUnaryTransportHandler) {
				h.EXPECT().Handle(
					yarpctest.NewContextMatcher(t, yarpctest.ContextTTL(internaltesttime.Second)),
					&yarpc.Request{
						Caller:    "bar",
						Service:   "foo",
						Transport: "tchannel",
						Encoding:  yarpc.Encoding("raw"),
						Procedure: "hello",
					},
					yarpc.NewBufferBytes([]byte{0x00}),
				).Return(nil, nil, fmt.Errorf("great sadness"))
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
			expectCall: func(h *yarpctest.MockUnaryTransportHandler) {
				req := &yarpc.Request{
					Caller:    "bar",
					Service:   "foo",
					Transport: "tchannel",
					Encoding:  yarpc.Encoding("json"),
					Procedure: "hello",
				}
				h.EXPECT().Handle(
					yarpctest.NewContextMatcher(t, yarpctest.ContextTTL(internaltesttime.Second)),
					req,
					// yarpctest.NewRequestMatcher(t, req),
					yarpc.NewBufferString("{}"),
				).Return(nil, nil, yarpcencoding.ResponseBodyEncodeError(req, errors.New(
					"serialization derp",
				)))
			},
			wantStatus: tchannel.ErrCodeBadRequest,
		},
		{
			desc: "handler timeout",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), internaltesttime.Millisecond)
			},
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "waituntiltimeout",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expectCall: func(h *yarpctest.MockUnaryTransportHandler) {
				req := &yarpc.Request{
					Caller:    "bar",
					Service:   "foo",
					Transport: "tchannel",
					Encoding:  yarpc.Encoding("raw"),
					Procedure: "waituntiltimeout",
				}
				h.EXPECT().Handle(
					yarpctest.NewContextMatcher(
						t, yarpctest.ContextTTL(internaltesttime.Millisecond)),
					req,
					yarpc.NewBufferBytes([]byte{0x00}),
				).Do(func(ctx context.Context, _ *yarpc.Request, _ *yarpc.Buffer) {
					<-ctx.Done()
				}).Return(nil, nil, context.DeadlineExceeded)
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
			expectCall: func(h *yarpctest.MockUnaryTransportHandler) {
				req := &yarpc.Request{
					Caller:    "bar",
					Service:   "foo",
					Transport: "tchannel",
					Encoding:  yarpc.Encoding("raw"),
					Procedure: "panic",
				}
				h.EXPECT().Handle(
					yarpctest.NewContextMatcher(
						t, yarpctest.ContextTTL(internaltesttime.Second)),
					req,
					yarpc.NewBufferBytes([]byte{0x00}),
				).Do(func(context.Context, *yarpc.Request, *yarpc.Buffer) {
					panic("oops I panicked!")
				})
			},
			wantStatus: tchannel.ErrCodeUnexpected,
		},
	}

	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
		if tt.ctx != nil {
			ctx = tt.ctx
		} else if tt.ctxFunc != nil {
			ctx, cancel = tt.ctxFunc()
		}
		defer cancel()

		mockCtrl := gomock.NewController(t)
		thandler := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
		spec := yarpc.NewUnaryTransportHandlerSpec(thandler)

		if tt.expectCall != nil {
			tt.expectCall(thandler)
		}

		resp := newResponseRecorder()
		tt.sendCall.resp = resp

		router := yarpctest.NewMockRouter(mockCtrl)
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

func TestGetSystemError(t *testing.T) {
	tests := []struct {
		giveErr  error
		wantCode tchannel.SystemErrCode
	}{
		{
			giveErr:  yarpcerror.UnavailableErrorf("test"),
			wantCode: tchannel.ErrCodeDeclined,
		},
		{
			giveErr:  errors.New("test"),
			wantCode: tchannel.ErrCodeUnexpected,
		},
		{
			giveErr:  yarpcerror.InvalidArgumentErrorf("test"),
			wantCode: tchannel.ErrCodeBadRequest,
		},
		{
			giveErr:  tchannel.NewSystemError(tchannel.ErrCodeBusy, "test"),
			wantCode: tchannel.ErrCodeBusy,
		},
		{
			giveErr:  yarpcerror.New(yarpcerror.Code(1235), "test"),
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
