package thrift

import (
	"reflect"
	"testing"

	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
			opts := ClientBuilderOptions(transporttest.NewMockClientConfig(mockCtrl), tt.give)
			for _, o := range opts {
				o.applyClientOption(&cfg)
			}

			assert.Equal(t, tt.want, cfg)
		})
	}
}
