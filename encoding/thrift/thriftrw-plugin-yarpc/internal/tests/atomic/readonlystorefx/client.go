// Code generated by thriftrw-plugin-yarpc
// @generated

package readonlystorefx

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoreclient"
)

// Params defines the dependencies for yarpc.
type Params struct {
	ClientConfigProvider transport.ClientConfigProvider
}

// Result defines the object yarpc provides.
type Result struct {
	Client readonlystoreclient.Interface
}

// Client provides a ReadOnlyStore client to an Fx application using the given name
// for routing.
//
// 	fx.Provide(
// 		readonlystorefx.Client("..."),
// 		newHandler,
// 	)
func Client(name string, opts ...thrift.ClientOption) interface{} {
	return func(p Params) Result {
		client := readonlystoreclient.New(p.ClientConfigProvider.ClientConfig(name), opts...)
		return Result{
			Client: client,
		}
	}
}
