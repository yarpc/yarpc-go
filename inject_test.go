package yarpc_test

import (
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/clientconfig"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

func TestInjectClientSuccess(t *testing.T) {
	type unknownClient interface{}

	type knownClient interface{}
	clear := yarpc.RegisterClientBuilder(
		func(transport.ClientConfig) knownClient { return knownClient(struct{}{}) })
	defer clear()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name   string
		target interface{}

		// list of services for which ClientConfig() should return successfully
		knownServices []string

		// list of field names in target we expect to be nil or non-nil
		wantNil    []string
		wantNonNil []string
	}{
		{
			name:   "empty",
			target: &struct{}{},
		},
		{
			name: "unknown service non-nil",
			target: &struct {
				Client json.Client `service:"foo"`
			}{
				Client: json.New(clientconfig.MultiOutbound(
					"foo",
					"bar",
					transport.Outbounds{
						Unary: transporttest.NewMockUnaryOutbound(mockCtrl),
					})),
			},
			wantNonNil: []string{"Client"},
		},
		{
			name: "unknown type untagged",
			target: &struct {
				Client unknownClient `notservice:"foo"`
			}{},
			wantNil: []string{"Client"},
		},
		{
			name: "unknown type non-nil",
			target: &struct {
				Client unknownClient `service:"foo"`
			}{Client: unknownClient(struct{}{})},
			wantNonNil: []string{"Client"},
		},
		{
			name:          "known type",
			knownServices: []string{"foo"},
			target: &struct {
				Client knownClient `service:"foo"`
			}{},
			wantNonNil: []string{"Client"},
		},
		{
			name:          "default encodings",
			knownServices: []string{"jsontest", "rawtest"},
			target: &struct {
				JSON json.Client `service:"jsontest"`
				Raw  raw.Client  `service:"rawtest"`
			}{},
			wantNonNil: []string{"JSON", "Raw"},
		},
		{
			name: "unexported field",
			target: &struct {
				rawClient raw.Client `service:"rawtest"`
			}{},
			wantNil: []string{"rawClient"},
		},
	}

	for _, tt := range tests {
		cp := newMockClientConfigProvider(mockCtrl, tt.knownServices...)
		assert.NotPanics(t, func() {
			yarpc.InjectClients(cp, tt.target)
		}, tt.name)

		for _, fieldName := range tt.wantNil {
			field := reflect.ValueOf(tt.target).Elem().FieldByName(fieldName)
			assert.True(t, field.IsNil(), "expected %q to be nil", fieldName)
		}

		for _, fieldName := range tt.wantNonNil {
			field := reflect.ValueOf(tt.target).Elem().FieldByName(fieldName)
			assert.False(t, field.IsNil(), "expected %q to be non-nil", fieldName)
		}
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
