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
	"fmt"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

func TestHandlerFailures(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    context.Context
		call   *fakeInboundCall
		expect func(*transporttest.MockHandler)
		msgs   []string
	}{
		{
			desc: "no timeout on context",
			ctx:  context.Background(),
			call: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			msgs: []string{"timeout required"},
		},
		{
			desc: "arg2 reader error",
			call: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    nil,
				arg3:    []byte{0x00},
			},
			msgs: []string{"failed to read headers"},
		},
		{
			desc: "arg2 parse error",
			call: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.JSON,
				arg2:    []byte("{not valid JSON}"),
				arg3:    []byte{0x00},
			},
			msgs: []string{"failed to read headers"},
		},
		{
			desc: "arg3 reader error",
			call: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    nil,
			},
			msgs: []string{"failed to read body"},
		},
		{
			desc: "internal error",
			call: &fakeInboundCall{
				service: "foo",
				caller:  "bar",
				method:  "hello",
				format:  tchannel.Raw,
				arg2:    []byte{0x00, 0x00},
				arg3:    []byte{0x00},
			},
			expect: func(h *transporttest.MockHandler) {
				h.EXPECT().Handle(
					gomock.Any(),
					transporttest.NewRequestMatcher(
						t, &transport.Request{
							Caller:    "bar",
							Service:   "foo",
							Encoding:  raw.Encoding,
							TTL:       time.Second,
							Procedure: "hello",
							Body:      bytes.NewReader([]byte{0x00}),
						},
					), gomock.Any(),
				).Return(fmt.Errorf("great sadness"))
			},
			msgs: []string{"internal error", "great sadness"},
		},
	}

	for _, tt := range tests {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		if tt.ctx != nil {
			ctx = tt.ctx
		}

		mockCtrl := gomock.NewController(t)
		thandler := transporttest.NewMockHandler(mockCtrl)
		if tt.expect != nil {
			tt.expect(thandler)
		}

		resp := newResponseRecorder()
		tt.call.resp = resp

		h := handler{thandler}
		h.handle(ctx, tt.call)
		require.Error(t, resp.systemErr, "expected error for %q", tt.desc)

		for _, msg := range tt.msgs {
			assert.Contains(
				t, resp.systemErr.Error(), msg,
				"error should contain message for %q", tt.desc)
		}

		mockCtrl.Finish()
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		format tchannel.Format
		apply  func(*responseWriter)
		arg2   []byte
		arg3   []byte
	}{
		{
			format: tchannel.Raw,
			apply: func(w *responseWriter) {
				headers := transport.NewHeaders(map[string]string{"foo": "bar"})
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
				headers := transport.NewHeaders(map[string]string{"foo": "bar"})
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
	}

	for _, tt := range tests {
		call := &fakeInboundCall{format: tt.format}
		resp := newResponseRecorder()
		call.resp = resp

		w := newResponseWriter(call)
		tt.apply(w)
		assert.NoError(t, w.Close())

		assert.Nil(t, resp.systemErr)
		assert.Equal(t, tt.arg2, resp.arg2.Bytes())
		assert.Equal(t, tt.arg3, resp.arg3.Bytes())
	}
}

func TestResponseWriterAddHeadersAfterWrite(t *testing.T) {
	call := &fakeInboundCall{format: tchannel.Raw, resp: newResponseRecorder()}
	w := newResponseWriter(call)
	w.Write([]byte("foo"))
	assert.Panics(t, func() {
		w.AddHeaders(transport.Headers{"foo": "bar"})
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

		w := newResponseWriter(&fakeInboundCall{
			format: tchannel.Raw,
			resp:   resp,
		})
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
