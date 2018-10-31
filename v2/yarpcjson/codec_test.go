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
		newCodec(tt.Func)
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
