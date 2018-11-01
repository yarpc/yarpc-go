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
	"github.com/stretchr/testify/require"

	"go.uber.org/yarpc/v2"
)

func TestNewCodec(t *testing.T) {
	tests := []struct {
		Name           string
		Func           interface{}
		ExpectedReader requestReader
	}{
		{
			Name: "foo",
			Func: func(context.Context, *struct{}) (*struct{}, error) {
				return nil, nil
			},
			ExpectedReader: structReader{},
		},
		{
			Name: "bar",
			Func: func(context.Context, map[string]interface{}) (*struct{}, error) {
				return nil, nil
			},
			ExpectedReader: mapReader{},
		},
		{
			Name: "baz",
			Func: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
			ExpectedReader: mapReader{},
		},
		{
			Name: "qux",
			Func: func(context.Context, interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
			ExpectedReader: ifaceEmptyReader{},
		},
	}

	for _, tt := range tests {
		c := newCodec(tt.Name, tt.Func)
		assert.IsType(t, tt.ExpectedReader, c.reader)
	}
}

func TestJsonCodecDecode(t *testing.T) {
	c := jsonCodec{
		reader: structReader{reflect.TypeOf(simpleResponse{})},
	}

	jsonContent := `{"Success":true}`
	validReqBuf := yarpc.NewBufferString(jsonContent)
	body, err := c.Decode(validReqBuf)
	require.NoError(t, err)
	assert.Equal(t, body.(*simpleResponse), &simpleResponse{Success: true})

	invalidReqBuf := yarpc.NewBufferString(`invalid`)
	body, err = c.Decode(invalidReqBuf)
	assert.Error(t, err)
	assert.Equal(t, nil, body)
}

func TestJsonCodecEncode(t *testing.T) {
	c := jsonCodec{
		reader: mapReader{reflect.TypeOf(make(map[string]interface{}))},
	}

	validResponse := simpleResponse{
		Success: true,
	}
	body, err := c.Encode(validResponse)
	require.NoError(t, err)
	assert.Equal(t, `{"Success":true}`, strings.TrimSuffix(body.String(), "\n"))

	invalidResponse := make(chan int)
	_, err = c.Encode(invalidResponse)
	assert.Error(t, err)
}
