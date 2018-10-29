package yarpcjson

import (
	"context"
	"reflect"
	"strings"
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
		reader: structReader{reflect.TypeOf(simpleResponse{})},
	}

	jsonContent := `{"Success":true}`
	validReqBuf := yarpc.NewBufferString(jsonContent)
	body, err := c.Decode(validReqBuf)
	assert.Equal(t, *body.(*simpleResponse), simpleResponse{Success: true})
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
	body, err := c.Encode(validResponse)
	assert.Equal(t, `{"Success":true}`, strings.TrimSuffix(body.String(), "\n"))
	assert.NoError(t, err)

	invalidResponse := make(chan int)
	_, err = c.Encode(invalidResponse)
	assert.Error(t, err)
}
