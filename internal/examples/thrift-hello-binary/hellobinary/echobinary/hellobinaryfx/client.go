// Code generated by thriftrw-plugin-yarpc
// @generated

package hellobinaryfx

import (
	"go.uber.org/fx"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/internal/examples/thrift-hello-binary/hellobinary/echobinary/hellobinaryclient"
)

// Params defines the dependencies for the HelloBinary client.
type Params struct {
	fx.In

	Provider yarpc.ClientConfig
}

// Result defines the output of this Fx module. It provides a HelloBinary client
// to an Fx application.
type Result struct {
	fx.Out

	Client hellobinaryclient.Interface

	// We are using an fx.Out struct here instead of just returning a client
	// so that we can add more values or add named versions of the client in
	// the future without breaking any existing code.
}

// Client provides a HelloBinary client to an Fx application using the given name
// for routing.
//
// 	fx.Provide(
// 		hellobinaryfx.Client("..."),
// 		newHandler,
// 	)
func Client(name string, opts ...thrift.ClientOption) interface{} {
	return func(p Params) Result {
		client := hellobinaryclient.New(p.Provider.ClientConfig(name), opts...)
		return Result{Client: client}
	}
}
