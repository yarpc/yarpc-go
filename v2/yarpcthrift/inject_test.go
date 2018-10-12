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

package yarpcthrift

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	yarpc "go.uber.org/yarpc/v2"
)

type someInterface interface{}

var _typeOfSomeInterface = reflect.TypeOf((*someInterface)(nil)).Elem()

func TestClientBuilderOptions(t *testing.T) {

	tests := []struct {
		desc string
		give reflect.StructField
		want clientConfig
	}{
		{
			desc: "no options",
			give: reflect.StructField{
				Name: "Client",
				Type: _typeOfSomeInterface,
				Tag:  `service:"keyvalue"`,
			},
		},
		{
			desc: "enveloped",
			give: reflect.StructField{
				Name: "Client",
				Type: _typeOfSomeInterface,
				Tag:  `service:"keyvalue" thrift:"enveloped"`,
			},
			want: clientConfig{Enveloping: true},
		},
		{
			desc: "multiplexed",
			give: reflect.StructField{
				Name: "Client",
				Type: _typeOfSomeInterface,
				Tag:  `service:"keyvalue" thrift:"multiplexed"`,
			},
			want: clientConfig{Multiplexed: true},
		},
		{
			desc: "enveloped and multiplexed",
			give: reflect.StructField{
				Name: "Client",
				Type: _typeOfSomeInterface,
				Tag:  `service:"keyvalue" thrift:"enveloped,multiplexed"`,
			},
			want: clientConfig{Enveloping: true, Multiplexed: true},
		},
		{
			desc: "ignore unknown",
			give: reflect.StructField{
				Name: "Client",
				Type: _typeOfSomeInterface,
				Tag:  `service:"keyvalue" thrift:"enveloped,foo=bar,Multiplexed"`,
			},
			want: clientConfig{Enveloping: true, Multiplexed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			var cfg clientConfig
			opts := ClientBuilderOptions(&yarpc.Client{}, tt.give)
			for _, o := range opts {
				o.applyClientOption(&cfg)
			}

			assert.Equal(t, tt.want, cfg)
		})
	}
}
