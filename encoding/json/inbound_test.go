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

package json

import (
	"bytes"
	"context"
	"encoding/json"
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

	handler := jsonHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Encoding:  "json",
		Body:      jsonBody(`{"name": "foo", "attributes": {"bar": 42}}`),
	}, resw)
	require.NoError(t, err)

	var response simpleResponse
	require.NoError(t, json.Unmarshal(resw.Body.Bytes(), &response))

	assert.Equal(t, simpleResponse{Success: true}, response)
}

func TestHandleMapSuccess(t *testing.T) {
	h := func(ctx context.Context, body map[string]interface{}) (map[string]string, error) {
		assert.Equal(t, 42.0, body["foo"])
		assert.Equal(t, []interface{}{"a", "b", "c"}, body["bar"])

		return map[string]string{"success": "true"}, nil
	}

	handler := jsonHandler{
		reader:  mapReader{reflect.TypeOf(make(map[string]interface{}))},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Encoding:  "json",
		Body:      jsonBody(`{"foo": 42, "bar": ["a", "b", "c"]}`),
	}, resw)
	require.NoError(t, err)

	var response struct{ Success string }
	require.NoError(t, json.Unmarshal(resw.Body.Bytes(), &response))
	assert.Equal(t, "true", response.Success)
}

func TestHandleInterfaceEmptySuccess(t *testing.T) {
	h := func(ctx context.Context, body interface{}) (interface{}, error) {
		return body, nil
	}

	handler := jsonHandler{reader: ifaceEmptyReader{}, handler: reflect.ValueOf(h)}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Encoding:  "json",
		Body:      jsonBody(`["a", "b", "c"]`),
	}, resw)
	require.NoError(t, err)

	assert.JSONEq(t, `["a", "b", "c"]`, resw.Body.String())
}

func TestHandleSuccessWithResponseHeaders(t *testing.T) {
	h := func(ctx context.Context, _ *simpleRequest) (*simpleResponse, error) {
		require.NoError(t, yarpc.CallFromContext(ctx).WriteResponseHeader("foo", "bar"))
		return &simpleResponse{Success: true}, nil
	}

	handler := jsonHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Encoding:  "json",
		Body:      jsonBody(`{"name": "foo", "attributes": {"bar": 42}}`),
	}, resw)
	require.NoError(t, err)

	assert.Equal(t, transport.NewHeaders().With("foo", "bar"), resw.Headers)
}

func jsonBody(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
