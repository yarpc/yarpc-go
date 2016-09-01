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

package tchannel

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

func TestHandlerErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		format  tchannel.Format
		headers []byte

		wantHeaders map[string]string
		wantBaggage map[string]string
	}{
		{
			format:      tchannel.JSON,
			headers:     []byte(`{"Rpc-Header-Foo": "bar", "context-foo": "Baz"}`),
			wantHeaders: map[string]string{"rpc-header-foo": "bar"},
			wantBaggage: map[string]string{"foo": "Baz"},
		},
		{
			format: tchannel.Thrift,
			headers: []byte{
				0x00, 0x02, // 2 headers
				0x00, 0x03, 'F', 'o', 'o', // Foo
				0x00, 0x03, 'B', 'a', 'r', // Bar
				0x00, 0x0B, 'C', 'o', 'n', 't', 'e', 'x', 't', '-', 'F', 'o', 'o', // Context-Foo
				0x00, 0x03, 'B', 'a', 'z', // Baz
			},
			wantHeaders: map[string]string{"foo": "Bar"},
			wantBaggage: map[string]string{"foo": "Baz"},
		},
	}

	for _, tt := range tests {
		rpcHandler := transporttest.NewMockHandler(mockCtrl)
		tchHandler := handler{nil, rpcHandler, transport.NoDeps}

		rpcHandler.EXPECT().Handle(
			transporttest.NewContextMatcher(t, transporttest.ContextBaggage(tt.wantBaggage)),
			transportOptions,
			transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:    "caller",
					Service:   "service",
					Headers:   transport.HeadersFromMap(tt.wantHeaders),
					Encoding:  transport.Encoding(tt.format),
					Procedure: "hello",
					Body:      bytes.NewReader([]byte("world")),
				}),
			gomock.Any(),
		).Return(nil)

		respRecorder := newResponseRecorder()

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		tchHandler.handle(ctx, &fakeInboundCall{
			service: "service",
			caller:  "caller",
			format:  tt.format,
			method:  "hello",
			arg2:    tt.headers,
			arg3:    []byte("world"),
			resp:    respRecorder,
		})

		assert.NoError(t, respRecorder.systemErr, "did not expect an error")
	}
}

func TestHandlerFailures(t *testing.T) {
	tests := []struct {
		desc string

		// context to use in the callm a default one is used otherwise.
		ctx context.Context

		sendCall   *fakeInboundCall
		expectCall func(*transporttest.MockHandler)

		wantErrors []string               // error message contents
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
			wantErrors: []string{"timeout required"},
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
			wantErrors: []string{
				`BadRequest: failed to decode "raw" request headers for`,
				`procedure "hello" of service "foo" from caller "bar"`,
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
			wantErrors: []string{
				`BadRequest: failed to decode "json" request headers for`,
				`procedure "hello" of service "foo" from caller "bar"`,
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
			wantErrors: []string{
				`UnexpectedError: error for procedure "hello" of service "foo"`,
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
			expectCall: func(h *transporttest.MockHandler) {
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(t, transporttest.ContextTTL(time.Second)),
					transportOptions,
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
			wantErrors: []string{
				`UnexpectedError: error for procedure "hello" of service "foo":`,
				"great sadness",
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
			expectCall: func(h *transporttest.MockHandler) {
				req := &transport.Request{
					Caller:    "bar",
					Service:   "foo",
					Encoding:  json.Encoding,
					Procedure: "hello",
					Body:      bytes.NewReader([]byte("{}")),
				}
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(t, transporttest.ContextTTL(time.Second)),
					transportOptions,
					transporttest.NewRequestMatcher(t, req),
					gomock.Any(),
				).Return(
					encoding.ResponseBodyEncodeError(req, errors.New(
						"serialization derp",
					)))
			},
			wantErrors: []string{
				`UnexpectedError: failed to encode "json" response body for`,
				`procedure "hello" of service "foo" from caller "bar":`,
				`serialization derp`,
			},
			wantStatus: tchannel.ErrCodeUnexpected,
		},
		{
			desc: "handler timeout",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
				return ctx
			}(),
			sendCall: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "waituntiltimeout",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expectCall: func(h *transporttest.MockHandler) {
				req := &transport.Request{
					Service:   "foo",
					Caller:    "bar",
					Procedure: "waituntiltimeout",
					Encoding:  raw.Encoding,
					Body:      bytes.NewReader([]byte{0x00}),
				}
				h.EXPECT().Handle(
					transporttest.NewContextMatcher(
						t, transporttest.ContextTTL(time.Millisecond)),
					transportOptions,
					transporttest.NewRequestMatcher(t, req),
					gomock.Any(),
				).Do(func(
					ctx context.Context,
					_ transport.Options,
					_ *transport.Request,
					_ transport.ResponseWriter,
				) {
					<-ctx.Done()
				}).Return(context.DeadlineExceeded)
			},
			wantErrors: []string{
				`tchannel error ErrCodeTimeout: Timeout: call to procedure "waituntiltimeout" of service "foo" from caller "bar" timed out after `},
			wantStatus: tchannel.ErrCodeTimeout,
		},
	}

	for _, tt := range tests {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		if tt.ctx != nil {
			ctx = tt.ctx
		}

		mockCtrl := gomock.NewController(t)
		thandler := transporttest.NewMockHandler(mockCtrl)
		if tt.expectCall != nil {
			tt.expectCall(thandler)
		}

		resp := newResponseRecorder()
		tt.sendCall.resp = resp

		handler{nil, thandler, transport.NoDeps}.handle(ctx, tt.sendCall)
		err := resp.systemErr
		require.Error(t, err, "expected error for %q", tt.desc)

		systemErr, isSystemErr := err.(tchannel.SystemError)
		require.True(t, isSystemErr, "expected %v for %q to be a system error", err, tt.desc)
		assert.Equal(t, tt.wantStatus, systemErr.Code(), tt.desc)

		for _, msg := range tt.wantErrors {
			assert.Contains(
				t, err.Error(), msg,
				"error should contain message for %q", tt.desc)
		}

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
				w.SetApplicationError()
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

		w := newResponseWriter(new(transport.Request), call)
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

func TestResponseWriterAddHeadersAfterWrite(t *testing.T) {
	call := &fakeInboundCall{format: tchannel.Raw, resp: newResponseRecorder()}
	w := newResponseWriter(new(transport.Request), call)
	w.Write([]byte("foo"))
	assert.Panics(t, func() {
		w.AddHeaders(transport.NewHeaders().With("foo", "bar"))
	})
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

		w := newResponseWriter(
			new(transport.Request),
			&fakeInboundCall{
				format: tchannel.Raw,
				resp:   resp,
			},
		)
		_, err := w.Write([]byte("foo"))
		assert.Error(t, err)
		for _, msg := range tt.messages {
			assert.Contains(t, err.Error(), msg)
		}

		// writing again should also fail
		_, err = w.Write([]byte("bar"))
		assert.Error(t, err)
		assert.Error(t, w.Close())
	}
}
