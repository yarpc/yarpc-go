// Code generated by thriftrw-plugin-yarpc
// @generated

package barclient

import (
	yarpc "go.uber.org/yarpc"
	transport "go.uber.org/yarpc/api/transport"
	thrift "go.uber.org/yarpc/encoding/thrift"
	fooclient "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/fooclient"
	reflect "reflect"
)

// Interface is a client for the Bar service.
type Interface interface {
	fooclient.Interface
}

// New builds a new client for the Bar service.
//
//	client := barclient.New(dispatcher.ClientConfig("bar"))
func New(c transport.ClientConfig, opts ...thrift.ClientOption) Interface {
	return client{
		c: thrift.New(thrift.Config{
			Service:      "Bar",
			ClientConfig: c,
		}, opts...),
		nwc: thrift.NewNoWire(thrift.Config{
			Service:      "Bar",
			ClientConfig: c,
		}, opts...),

		Interface: fooclient.New(
			c,
			append(
				opts,
				thrift.Named("Bar"),
			)...,
		),
	}
}

func init() {
	yarpc.RegisterClientBuilder(
		func(c transport.ClientConfig, f reflect.StructField) Interface {
			return New(c, thrift.ClientBuilderOptions(c, f)...)
		},
	)
}

type client struct {
	fooclient.Interface

	c   thrift.Client
	nwc thrift.NoWireClient
}
