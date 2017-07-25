// Code generated by thriftrw-plugin-yarpc
// @generated

package baseservicefx

import (
	"go.uber.org/fx"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseserviceclient"
)

// Params defines the dependencies for BaseService client.
type Params struct {
	fx.In

	Provider transport.ClientConfigProvider
}

// Result defines the object BaseService client provides.
type Result struct {
	fx.Out

	Client baseserviceclient.Interface
}

// Client provides a BaseService client to an Fx application using the given name
// for routing.
//
// 	fx.Provide(
// 		baseservicefx.Client("..."),
// 		newHandler,
// 	)
func Client(name string, opts ...thrift.ClientOption) interface{} {
	return func(p Params) Result {
		client := baseserviceclient.New(p.Provider.ClientConfig(name), opts...)
		return Result{Client: client}
	}
}
