// Copyright (c) 2022 Uber Technologies, Inc.
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

package yarpc_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/clientconfig"
)

func TestRegisterClientBuilderPanics(t *testing.T) {
	tests := []struct {
		name string
		give interface{}
	}{
		{name: "nil", give: nil},
		{name: "wrong kind", give: 42},
		{
			name: "already registered",
			give: func(transport.ClientConfig) json.Client { return nil },
		},
		{
			name: "wrong argument type",
			give: func(int) json.Client { return nil },
		},
		{
			name: "wrong return type",
			give: func(transport.ClientConfig) string { return "" },
		},
		{
			name: "no arguments",
			give: func() json.Client { return nil },
		},
		{
			name: "too many arguments",
			give: func(transport.ClientConfig, reflect.StructField, string) json.Client { return nil },
		},
		{
			name: "wrong number of arguments",
			give: func(transport.ClientConfig, ...string) json.Client { return nil },
		},
		{
			name: "wrong number of returns",
			give: func(transport.ClientConfig) (json.Client, error) { return nil, nil },
		},
	}

	for _, tt := range tests {
		assert.Panics(t, func() { yarpc.RegisterClientBuilder(tt.give) }, tt.name)
	}
}

func TestInjectClientsPanics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type unknownClient interface{}

	tests := []struct {
		name           string
		failOnServices []string
		target         interface{}
	}{
		{
			name:   "not a pointer to a struct",
			target: struct{}{},
		},
		{
			name:           "unknown service",
			failOnServices: []string{"foo"},
			target: &struct {
				Client json.Client `service:"foo"`
			}{},
		},
		{
			name: "unknown client",
			target: &struct {
				Client unknownClient `service:"bar"`
			}{},
		},
	}

	for _, tt := range tests {
		cp := newMockClientConfigProvider(mockCtrl)
		for _, s := range tt.failOnServices {
			cp.EXPECT().ClientConfig(s).Do(func(s string) {
				panic(fmt.Sprintf("unknown service %q", s))
			})
		}

		assert.Panics(t, func() {
			yarpc.InjectClients(cp, tt.target)
		}, tt.name)
	}
}

type someClient interface{}

// Helps build client builders (of type someClient) which verify the
// ClientConfig and optionally, the StructField.
type clientBuilderConfig struct {
	ClientConfig gomock.Matcher
	StructField  gomock.Matcher
}

func (c clientBuilderConfig) clientConfigBuilder(t *testing.T) func(cc transport.ClientConfig) someClient {
	return func(cc transport.ClientConfig) someClient {
		require.True(t, c.ClientConfig.Matches(cc), "client config %v did not match %v", cc, c.ClientConfig)
		return someClient(struct{}{})
	}
}

func (c clientBuilderConfig) Get(t *testing.T) interface{} {
	ccBuilder := c.clientConfigBuilder(t)
	if c.StructField == nil {
		return ccBuilder
	}

	return func(cc transport.ClientConfig, f reflect.StructField) someClient {
		require.True(t, c.StructField.Matches(f), "struct field %#v did not match %v", f, c.StructField)
		return ccBuilder(cc)
	}
}

func TestInjectClientSuccess(t *testing.T) {
	type testCase struct {
		target interface{}

		// list of client builders to register using RegisterClientBuilder.
		//
		// Test instances of these can be built with
		// clientBuilderConfig.Get(t).
		clientBuilders []interface{}

		// list of services for which ClientConfig() should return successfully
		knownServices []string

		// list of field names in target we expect to be nil or non-nil
		wantNil    []string
		wantNonNil []string
	}

	tests := []struct {
		name  string
		build func(*testing.T, *gomock.Controller) testCase
	}{
		{
			name: "empty",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.target = &struct{}{}
				return
			},
		},
		{
			name: "unknown service non-nil",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.target = &struct {
					Client json.Client `service:"foo"`
				}{
					Client: json.New(clientconfig.MultiOutbound(
						"foo",
						"bar",
						transport.Outbounds{
							Unary: transporttest.NewMockUnaryOutbound(mockCtrl),
						})),
				}
				tt.wantNonNil = []string{"Client"}
				return
			},
		},
		{
			name: "unknown type untagged",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.target = &struct {
					Client someClient `notservice:"foo"`
				}{}
				tt.wantNil = []string{"Client"}
				return
			},
		},
		{
			name: "unknown type non-nil",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.target = &struct {
					Client someClient `service:"foo"`
				}{Client: someClient(struct{}{})}
				tt.wantNonNil = []string{"Client"}
				return
			},
		},
		{
			name: "known type",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.knownServices = []string{"foo"}
				tt.clientBuilders = []interface{}{
					clientBuilderConfig{ClientConfig: gomock.Any()}.Get(t),
				}
				tt.target = &struct {
					Client someClient `service:"foo"`
				}{}
				tt.wantNonNil = []string{"Client"}
				return
			},
		},
		{
			name: "known type with struct field",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.knownServices = []string{"foo"}
				tt.clientBuilders = []interface{}{
					clientBuilderConfig{
						ClientConfig: gomock.Any(),
						StructField: gomock.Eq(reflect.StructField{
							Name:  "Client",
							Type:  reflect.TypeOf((*someClient)(nil)).Elem(),
							Index: []int{0},
							Tag:   `service:"foo" thrift:"bar"`,
						}),
					}.Get(t),
				}
				tt.target = &struct {
					Client someClient `service:"foo" thrift:"bar"`
				}{}
				tt.wantNonNil = []string{"Client"}
				return
			},
		},
		{
			name: "default encodings",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.knownServices = []string{"jsontest", "rawtest"}
				tt.target = &struct {
					JSON json.Client `service:"jsontest"`
					Raw  raw.Client  `service:"rawtest"`
				}{}
				tt.wantNonNil = []string{"JSON", "Raw"}
				return
			},
		},
		{
			name: "unexported field",
			build: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.target = &struct {
					rawClient raw.Client `service:"rawtest"`
				}{}
				tt.wantNil = []string{"rawClient"}
				return
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			tt := testCase.build(t, mockCtrl)

			for _, builder := range tt.clientBuilders {
				forget := yarpc.RegisterClientBuilder(builder)
				defer forget()
			}

			cp := newMockClientConfigProvider(mockCtrl, tt.knownServices...)
			assert.NotPanics(t, func() {
				yarpc.InjectClients(cp, tt.target)
			})

			for _, fieldName := range tt.wantNil {
				field := reflect.ValueOf(tt.target).Elem().FieldByName(fieldName)
				assert.True(t, field.IsNil(), "expected %q to be nil", fieldName)
			}

			for _, fieldName := range tt.wantNonNil {
				field := reflect.ValueOf(tt.target).Elem().FieldByName(fieldName)
				assert.False(t, field.IsNil(), "expected %q to be non-nil", fieldName)
			}
		})
	}
}

// newMockClientConfigProvider builds a MockClientConfigProvider which expects ClientConfig()
// calls for the given services and returns mock ClientConfigs for them.
func newMockClientConfigProvider(ctrl *gomock.Controller, services ...string) *transporttest.MockClientConfigProvider {
	cp := transporttest.NewMockClientConfigProvider(ctrl)
	for _, s := range services {
		cp.EXPECT().ClientConfig(s).Return(transporttest.NewMockClientConfig(ctrl))
	}
	return cp
}
