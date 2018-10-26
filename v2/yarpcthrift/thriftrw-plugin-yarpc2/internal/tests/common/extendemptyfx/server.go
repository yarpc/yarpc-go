// Code generated by thriftrw-plugin-yarpc
// @generated

package extendemptyfx

import (
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcthrift"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/extendemptyserver"
)

// ServerParams defines the dependencies for the ExtendEmpty server.
type ServerParams struct {
	fx.In

	Handler extendemptyserver.Interface
}

// ServerResult defines the output of ExtendEmpty server module. It provides the
// procedures of a ExtendEmpty handler to an Fx application.
//
// The procedures are provided to the "yarpcfx" value group. Dig 1.2 or newer
// must be used for this feature to work.
type ServerResult struct {
	fx.Out

	Procedures []yarpc.Procedure `group:"yarpcfx"`
}

// Server provides procedures for ExtendEmpty to an Fx application. It expects a
// extendemptyfx.Interface to be present in the container.
//
// 	fx.Provide(
// 		func(h *MyExtendEmptyHandler) extendemptyserver.Interface {
// 			return h
// 		},
// 		extendemptyfx.Server(),
// 	)
func Server(opts ...yarpcthrift.RegisterOption) interface{} {
	return func(p ServerParams) ServerResult {
		procedures := extendemptyserver.New(p.Handler, opts...)
		return ServerResult{Procedures: procedures}
	}
}
