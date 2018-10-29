package yarpcjson

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/yarpc/v2"
)

func TestNewCodec(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{
			"foo",
			func(context.Context, *struct{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"bar",
			func(context.Context, map[string]interface{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"baz",
			func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
		{
			"qux",
			func(context.Context, interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		newCodec(tt.Name, tt.Func)
	}
}

func TestJsonCodec_Decode(t *testing.T) {
	c := jsonCodec{
		reader: mapReader{reflect.TypeOf(make(map[string]interface{}))},
	}

	jsonContent := `{"foo": 42}`
	validReqBuf := yarpc.NewBufferString(jsonContent)
	_, err := c.Decode(validReqBuf)
	assert.NoError(t, err)

	invalidReqBuf := yarpc.NewBufferString(`invalid`)
	_, err = c.Decode(invalidReqBuf)
	assert.Error(t, err)
}

func TestJsonCodec_Encode(t *testing.T) {
	c := jsonCodec{
		reader: mapReader{reflect.TypeOf(make(map[string]interface{}))},
	}

	validResponse := simpleResponse{
		Success: true,
	}
	_, err := c.Encode(validResponse)
	assert.NoError(t, err)

	invalidResponse := make(chan int)
	_, err = c.Encode(invalidResponse)
	assert.Error(t, err)
}
