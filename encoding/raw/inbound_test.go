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

package raw

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"context"

	"github.com/stretchr/testify/assert"
	"github.com/uber/tchannel-go/testutils/testreader"
)

func TestRawHandler(t *testing.T) {
	// handler to use for test cases where the handler should not be called
	handlerNotCalled :=
		func(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
			t.Errorf("unexpected call handle(%v, %v)", reqMeta, body)
			return nil, nil, fmt.Errorf("unexpected call handle(%v, %v)", reqMeta, body)
		}

	tests := []struct {
		procedure  string
		headers    transport.Headers
		bodyChunks [][]byte
		handler    Handler

		wantErr     string
		wantHeaders transport.Headers
		wantBody    []byte
	}{
		{
			procedure: "foo",
			bodyChunks: [][]byte{
				{1, 2, 3},
				{4, 5, 6},
			},
			handler: func(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
				assert.Equal(t, "foo", reqMeta.Procedure())
				assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, body)
				return []byte("hello"), nil, nil
			},
			wantBody: []byte("hello"),
		},
		{
			procedure: "bar",
			bodyChunks: [][]byte{
				{1, 2, 3},
				nil, // triggers a read error
				{4, 5, 6},
			},
			handler: handlerNotCalled,
			wantErr: "error set by user",
			// TODO consistent error messages between languages
		},
		{
			procedure:  "baz",
			bodyChunks: [][]byte{},
			handler: func(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
				assert.Equal(t, []byte{}, body)
				return nil, nil, fmt.Errorf("great sadness")
			},
			wantErr: "great sadness",
		},
		{
			procedure:  "responseHeaders",
			bodyChunks: [][]byte{},
			handler: func(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
				resMeta := yarpc.NewResMeta().Headers(yarpc.NewHeaders().With("hello", "world"))
				return []byte{}, resMeta, nil
			},
			wantHeaders: transport.NewHeaders().With("hello", "world"),
		},
	}

	for _, tt := range tests {
		handler := rawHandler{tt.handler}
		resw := new(transporttest.FakeResponseWriter)

		writer, chunkReader := testreader.ChunkReader()
		for _, chunk := range tt.bodyChunks {
			writer <- chunk
		}
		close(writer)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := handler.Handle(ctx, &transport.Request{
			Procedure: tt.procedure,
			Headers:   tt.headers,
			Encoding:  "raw",
			Body:      chunkReader,
		}, resw)

		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Equal(t, err.Error(), tt.wantErr)
			}
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, tt.wantHeaders, resw.Headers)
				assert.Equal(t, tt.wantBody, resw.Body.Bytes(),
					"body does not match for %s", tt.procedure)
			}
		}
	}
}
