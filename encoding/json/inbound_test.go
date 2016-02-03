package json

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

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
	h := wrapHandler(
		"foo",
		func(_ context.Context, _ yarpc.Meta, r *simpleRequest) (*simpleResponse, yarpc.Meta, error) {
			assert.Equal(t, "foo", r.Name)
			assert.Equal(t, map[string]int32{"bar": 42}, r.Attributes)

			return &simpleResponse{Success: true}, nil, nil
		},
	)

	res, err := h.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Body:      jsonBody(`{"name": "foo", "attributes": {"bar": 42}}`),
	})
	require.NoError(t, err)

	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	var response simpleResponse
	require.NoError(t, json.Unmarshal(body, &response))

	assert.Equal(t, simpleResponse{Success: true}, response)
}

func TestHandleMapSuccess(t *testing.T) {
	h := wrapHandler(
		"foo",
		func(_ context.Context, _ yarpc.Meta, r map[string]interface{}) (map[string]string, yarpc.Meta, error) {
			// 42.0 instead of 42 because json.Decode defaults to float64 for
			// numbers.
			assert.Equal(t, 42.0, r["foo"])
			assert.Equal(t, []interface{}{"a", "b", "c"}, r["bar"])

			return map[string]string{"success": "true"}, nil, nil
		},
	)

	res, err := h.Handle(context.Background(), &transport.Request{
		Procedure: "foo",
		Body:      jsonBody(`{"foo": 42, "bar": ["a", "b", "c"]}`),
	})
	require.NoError(t, err)

	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	var response struct{ Success string }
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, "true", response.Success)
}

func jsonBody(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
