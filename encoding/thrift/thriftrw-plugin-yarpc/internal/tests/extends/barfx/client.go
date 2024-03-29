// Code generated by thriftrw-plugin-yarpc
// @generated

package barfx

import (
	fx "go.uber.org/fx"
	yarpc "go.uber.org/yarpc"
	transport "go.uber.org/yarpc/api/transport"
	restriction "go.uber.org/yarpc/api/x/restriction"
	thrift "go.uber.org/yarpc/encoding/thrift"
	barclient "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/barclient"
)

// Params defines the dependencies for the Bar client.
type Params struct {
	fx.In

	Provider    yarpc.ClientConfig
	Restriction restriction.Checker `optional:"true"`
}

// Result defines the output of the Bar client module. It provides a
// Bar client to an Fx application.
type Result struct {
	fx.Out

	Client barclient.Interface

	// We are using an fx.Out struct here instead of just returning a client
	// so that we can add more values or add named versions of the client in
	// the future without breaking any existing code.
}

// Client provides a Bar client to an Fx application using the given name
// for routing.
//
//	fx.Provide(
//		barfx.Client("..."),
//		newHandler,
//	)
func Client(name string, opts ...thrift.ClientOption) interface{} {
	return func(p Params) Result {
		cc := p.Provider.ClientConfig(name)
		if namer, ok := cc.GetUnaryOutbound().(transport.Namer); ok && p.Restriction != nil {
			if err := p.Restriction.Check(thrift.Encoding, namer.TransportName()); err != nil {
				panic(err.Error())
			}
		}
		client := barclient.New(cc, opts...)
		return Result{Client: client}
	}
}
