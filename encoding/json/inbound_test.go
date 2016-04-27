package json

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type simpleRequest struct {
	Name       string
	Attributes map[string]int32
}

type simpleResponse struct {
	Success bool
}

func TestHandleStructSuccess(t *testing.T) {
	h := func(r *Request, body *simpleRequest) (*simpleResponse, *Response, error) {
		assert.Equal(t, "simpleCall", r.Procedure)
		assert.Equal(t, "foo", body.Name)
		assert.Equal(t, map[string]int32{"bar": 42}, body.Attributes)

		return &simpleResponse{Success: true}, nil, nil
	}

	handler := jsonHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Body:      jsonBody(`{"name": "foo", "attributes": {"bar": 42}}`),
	}, resw)
	require.NoError(t, err)

	var response simpleResponse
	require.NoError(t, json.Unmarshal(resw.Body.Bytes(), &response))

	assert.Equal(t, simpleResponse{Success: true}, response)
}

func TestHandleMapSuccess(t *testing.T) {
	h := func(_ *Request, body map[string]interface{}) (map[string]string, *Response, error) {
		assert.Equal(t, 42.0, body["foo"])
		assert.Equal(t, []interface{}{"a", "b", "c"}, body["bar"])

		return map[string]string{"success": "true"}, nil, nil
	}

	handler := jsonHandler{
		reader:  mapReader{reflect.TypeOf(make(map[string]interface{}))},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Body:      jsonBody(`{"foo": 42, "bar": ["a", "b", "c"]}`),
	}, resw)
	require.NoError(t, err)

	var response struct{ Success string }
	require.NoError(t, json.Unmarshal(resw.Body.Bytes(), &response))
	assert.Equal(t, "true", response.Success)
}

func TestHandleInterfaceEmptySuccess(t *testing.T) {
	h := func(_ *Request, body interface{}) (interface{}, *Response, error) {
		return body, nil, nil
	}

	handler := jsonHandler{reader: ifaceEmptyReader{}, handler: reflect.ValueOf(h)}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Body:      jsonBody(`{"foo": 42, "bar": ["a", "b", "c"]}`),
	}, resw)
	require.NoError(t, err)

	assert.JSONEq(t, `{"foo": 42, "bar": ["a", "b", "c"]}`, resw.Body.String())
}

func TestHandleSuccessWithResponseHeaders(t *testing.T) {
	h := func(*Request, *simpleRequest) (*simpleResponse, *Response, error) {
		response := &Response{Headers: transport.Headers{"foo": "bar"}}
		return &simpleResponse{Success: true}, response, nil
	}

	handler := jsonHandler{
		reader:  structReader{reflect.TypeOf(simpleRequest{})},
		handler: reflect.ValueOf(h),
	}

	resw := new(transporttest.FakeResponseWriter)
	err := handler.Handle(context.Background(), &transport.Request{
		Procedure: "simpleCall",
		Body:      jsonBody(`{"name": "foo", "attributes": {"bar": 42}}`),
	}, resw)
	require.NoError(t, err)

	assert.Equal(t, transport.Headers{"foo": "bar"}, resw.Headers)
}

func jsonBody(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
