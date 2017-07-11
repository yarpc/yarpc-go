// Copyright (c) 2017 Uber Technologies, Inc.
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

package gob

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"reflect"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type simpleRequest struct {
	Name       string
	Attributes map[string]int32
}

type simpleResponse struct {
	Success bool
}

func TestHandleStructSuccess(t *testing.T) {
	h := func(ctx context.Context, body *simpleRequest) (*simpleResponse, error) {
		assert.Equal(t, "simpleCall", yarpc.CallFromContext(ctx).Procedure())
		assert.Equal(t, "foo", body.Name)
		assert.Equal(t, map[string]int32{"bar": 42}, body.Attributes)

		return &simpleResponse{Success: true}, nil
	}

	handler := gobHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Encoding:  "gob",
		Body: gobFrom(t, simpleRequest{
			Name:       "foo",
			Attributes: map[string]int32{"bar": 42},
		}),
	}, resw)
	require.NoError(t, err)

	var response simpleResponse
	require.NoError(t, gob.NewDecoder(&resw.Body).Decode(&response))

	assert.Equal(t, simpleResponse{Success: true}, response)
}

func TestHandleMapSuccess(t *testing.T) {
	h := func(ctx context.Context, body map[string]interface{}) (map[string]string, error) {
		assert.Equal(t, 42.0, body["foo"])
		assert.Equal(t, []string{"a", "b", "c"}, body["bar"])

		return map[string]string{"success": "true"}, nil
	}

	handler := gobHandler{
		reader:  mapReader{reflect.TypeOf(make(map[string]interface{}))},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Encoding:  "gob",
		Body: gobFrom(t, map[string]interface{}{
			"foo": 42.0,
			"bar": []string{"a", "b", "c"},
		}),
	}, resw)
	require.NoError(t, err)

	var response map[string]string
	require.NoError(t, gob.NewDecoder(&resw.Body).Decode(&response))
	assert.Equal(t, "true", response["success"])
}

func TestHandleSuccessWithResponseHeaders(t *testing.T) {
	h := func(ctx context.Context, _ *simpleRequest) (*simpleResponse, error) {
		require.NoError(t, yarpc.CallFromContext(ctx).WriteResponseHeader("foo", "bar"))
		return &simpleResponse{Success: true}, nil
	}

	handler := gobHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Encoding:  "gob",
		Body: gobFrom(t, simpleRequest{
			Name:       "foo",
			Attributes: map[string]int32{"bar": 42},
		}),
	}, resw)
	require.NoError(t, err)

	assert.Equal(t, transport.NewHeaders().With("foo", "bar"), resw.Headers)
}

func gobFrom(t *testing.T, v interface{}) io.Reader {
	var buff bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buff).Encode(v))
	return bytes.NewReader(buff.Bytes())
}
