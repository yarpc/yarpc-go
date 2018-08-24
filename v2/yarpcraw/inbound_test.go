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

package yarpcraw

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go/testutils/testreader"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctransporttest"
)

func TestRawHandler(t *testing.T) {
	// handler to use for test cases where the handler should not be called
	handlerNotCalled :=
		func(ctx context.Context, body []byte) ([]byte, error) {
			t.Errorf("unexpected call handle(%v)", body)
			return nil, fmt.Errorf("unexpected call handle(%v)", body)
		}

	tests := []struct {
		procedure  string
		headers    yarpc.Headers
		bodyChunks [][]byte
		handler    UnaryHandler

		wantErr     string
		wantHeaders yarpc.Headers
		wantBody    []byte
	}{
		{
			procedure: "foo",
			bodyChunks: [][]byte{
				{1, 2, 3},
				{4, 5, 6},
			},
			handler: func(ctx context.Context, body []byte) ([]byte, error) {
				assert.Equal(t, "foo", yarpc.CallFromContext(ctx).Procedure())
				assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, body)
				return []byte("hello"), nil
			},
			wantBody: []byte("hello"),
		},
		{
			procedure: "foo",
			bodyChunks: [][]byte{
				{1, 2, 3},
				{4, 5, 6},
			},
			handler: func(ctx context.Context, body []byte) ([]byte, error) {
				assert.Equal(t, "foo", yarpc.CallFromContext(ctx).Procedure())
				assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, body)
				return []byte("hello"), errors.New("bar")
			},
			wantBody: []byte("hello"),
			wantErr:  "bar",
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
			handler: func(ctx context.Context, body []byte) ([]byte, error) {
				assert.Equal(t, []byte{}, body)
				return nil, fmt.Errorf("great sadness")
			},
			wantErr: "great sadness",
		},
		{
			procedure:  "responseHeaders",
			bodyChunks: [][]byte{},
			handler: func(ctx context.Context, body []byte) ([]byte, error) {
				require.NoError(t, yarpc.CallFromContext(ctx).WriteResponseHeader("hello", "world"))
				return []byte{}, nil
			},
			wantHeaders: yarpc.NewHeaders().With("hello", "world"),
		},
	}

	for _, tt := range tests {
		handler := rawUnaryHandler{tt.handler}
		resw := new(yarpctransporttest.FakeResponseWriter)

		writer, chunkReader := testreader.ChunkReader()
		for _, chunk := range tt.bodyChunks {
			writer <- chunk
		}
		close(writer)

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		err := handler.Handle(ctx, &yarpc.Request{
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
			assert.NoError(t, err)
		}
		assert.Equal(t, tt.wantHeaders, resw.Headers)
		if tt.wantBody != nil {
			assert.Equal(t, tt.wantBody, resw.Body.Bytes(),
				"body does not match for %s", tt.procedure)
		}
	}
}
