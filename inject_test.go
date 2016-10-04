package yarpc_test

import (
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRegisterClientFactoryPanics(t *testing.T) {
	tests := []struct {
		name string
		give interface{}
	}{
		{name: "nil", give: nil},
		{name: "wrong kind", give: 42},
		{
			name: "already registered",
			give: func(transport.Channel) json.Client { return nil },
		},
		{
			name: "wrong argument type",
			give: func(int) json.Client { return nil },
		},
		{
			name: "wrong return type",
			give: func(transport.Channel) string { return "" },
		},
		{
			name: "wrong number of arguments",
			give: func(transport.Channel, ...string) json.Client { return nil },
		},
		{
			name: "wrong number of returns",
			give: func(transport.Channel) (json.Client, error) { return nil, nil },
		},
	}

	for _, tt := range tests {
		assert.Panics(t, func() { yarpc.RegisterClientFactory(tt.give) }, tt.name)
	}
}

func TestInjectClientsPanics(t *testing.T) {
	type unknownClient interface{}

	tests := []struct {
		name      string
		outbounds []string
		target    interface{}
	}{
		{
			name:   "not a pointer to a struct",
			target: struct{}{},
		},
		{
			name: "unknown service",
			target: &struct {
				Client json.Client `service:"foo"`
			}{},
		},
		{
			name:      "unknown client",
			outbounds: []string{"foo"},
			target: &struct {
				Client unknownClient `service:"foo"`
			}{},
		},
	}

	for _, tt := range tests {
		dispatcherWithOutbounds(t, tt.outbounds, func(d yarpc.Dispatcher) {
			assert.Panics(t, func() { yarpc.InjectClients(d, tt.target) }, tt.name)
		})
	}
}

func TestInjectClientSuccess(t *testing.T) {
	type unknownClient interface{}

	type knownClient interface{}
	clear := yarpc.RegisterClientFactory(
		func(transport.Channel) knownClient { return knownClient(struct{}{}) })
	defer clear()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name      string
		outbounds []string
		target    interface{}
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
				Client: json.New(transport.IdentityChannel(
					"foo", "bar", transporttest.NewMockOutbound(mockCtrl))),
			},
		},
		{
			name: "unknown type untagged",
			target: &struct {
				Client unknownClient `notservice:"foo"`
			}{},
		},
		{
			name:      "unknown type non-nil",
			outbounds: []string{"foo"},
			target: &struct {
				Client unknownClient `service:"foo"`
			}{Client: unknownClient(struct{}{})},
		},
		{
			name:      "known type",
			outbounds: []string{"foo"},
			target: &struct {
				Client knownClient `service:"foo"`
			}{},
		},
		{
			name:      "default encodings",
			outbounds: []string{"jsontest", "rawtest"},
			target: &struct {
				JSON json.Client `service:"jsontest"`
				Raw  raw.Client  `service:"rawtest"`
			}{},
		},
	}

	for _, tt := range tests {
		dispatcherWithOutbounds(t, tt.outbounds, func(d yarpc.Dispatcher) {
			assert.NotPanics(t, func() { yarpc.InjectClients(d, tt.target) }, tt.name)
		})
	}
}

func dispatcherWithOutbounds(t *testing.T, outnames []string, f func(yarpc.Dispatcher)) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	outbounds := make(transport.Outbounds, len(outnames))
	for _, name := range outnames {
		outbounds[name] = transporttest.NewMockOutbound(mockCtrl)
	}

	f(yarpc.NewDispatcher(yarpc.Config{Name: "foo", Outbounds: outbounds}))
}
