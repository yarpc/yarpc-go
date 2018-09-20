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

package yarpcjson

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
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

	reqBuf := yarpc.NewBufferString(`{"name": "foo", "attributes": {"bar": 42}}`)
	_, resBuf, err := handler.Handle(context.Background(), &yarpc.Request{
		Procedure: "simpleCall",
		Encoding:  "json",
	}, reqBuf)
	require.NoError(t, err)

	var response simpleResponse
	require.NoError(t, json.Unmarshal(resBuf.Bytes(), &response))
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

	reqBuf := yarpc.NewBufferString(`{"foo": 42, "bar": ["a", "b", "c"]}`)
	_, resBuf, err := handler.Handle(context.Background(), &yarpc.Request{
		Procedure: "foo",
		Encoding:  "json",
	}, reqBuf)

	require.NoError(t, err)

	var response struct{ Success string }
	require.NoError(t, json.Unmarshal(resBuf.Bytes(), &response))
	assert.Equal(t, "true", response.Success)
}

func TestHandleInterfaceEmptySuccess(t *testing.T) {
	h := func(ctx context.Context, body interface{}) (interface{}, error) {
		return body, nil
	}

	handler := jsonHandler{reader: ifaceEmptyReader{}, handler: reflect.ValueOf(h)}

	reqBuf := yarpc.NewBufferString(`["a", "b", "c"]`)
	_, _, err := handler.Handle(context.Background(), &yarpc.Request{
		Procedure: "foo",
		Encoding:  "json",
	}, reqBuf)

	require.NoError(t, err)
	assert.JSONEq(t, `["a", "b", "c"]`, reqBuf.String())
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

	reqBuf := yarpc.NewBufferString(`{"name": "foo", "attributes": {"bar": 42}}`)
	res, _, err := handler.Handle(context.Background(), &yarpc.Request{
		Procedure: "simpleCall",
		Encoding:  "json",
	}, reqBuf)

	require.NoError(t, err)
	assert.Equal(t, yarpc.NewHeaders().With("foo", "bar"), res.Headers)
}

func TestHandleBothResponseError(t *testing.T) {
	h := func(ctx context.Context, body *simpleRequest) (*simpleResponse, error) {
		assert.Equal(t, "simpleCall", yarpc.CallFromContext(ctx).Procedure())
		assert.Equal(t, "foo", body.Name)
		assert.Equal(t, map[string]int32{"bar": 42}, body.Attributes)

		return &simpleResponse{Success: true}, errors.New("bar")
	}

	handler := jsonHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	reqBuf := yarpc.NewBufferString(`{"name": "foo", "attributes": {"bar": 42}}`)
	_, resBuf, err := handler.Handle(context.Background(), &yarpc.Request{
		Procedure: "simpleCall",
		Encoding:  "json",
	}, reqBuf)

	require.Equal(t, errors.New("bar"), err)

	var response simpleResponse
	require.NoError(t, json.Unmarshal(resBuf.Bytes(), &response))
	assert.Equal(t, simpleResponse{Success: true}, response)
}
