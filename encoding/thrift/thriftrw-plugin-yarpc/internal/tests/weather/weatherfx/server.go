// Code generated by thriftrw-plugin-yarpc
// @generated

package weatherfx

import (
	fx "go.uber.org/fx"
	transport "go.uber.org/yarpc/api/transport"
	thrift "go.uber.org/yarpc/encoding/thrift"
	weatherserver "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/weather/weatherserver"
)

// ServerParams defines the dependencies for the Weather server.
type ServerParams struct {
	fx.In

	Handler weatherserver.Interface
}

// ServerResult defines the output of Weather server module. It provides the
// procedures of a Weather handler to an Fx application.
//
// The procedures are provided to the "yarpcfx" value group. Dig 1.2 or newer
// must be used for this feature to work.
type ServerResult struct {
	fx.Out

	Procedures []transport.Procedure `group:"yarpcfx"`
}

// Server provides procedures for Weather to an Fx application. It expects a
// weatherfx.Interface to be present in the container.
//
//	fx.Provide(
//		func(h *MyWeatherHandler) weatherserver.Interface {
//			return h
//		},
//		weatherfx.Server(),
//	)
func Server(opts ...thrift.RegisterOption) interface{} {
	return func(p ServerParams) ServerResult {
		procedures := weatherserver.New(p.Handler, opts...)
		return ServerResult{Procedures: procedures}
	}
}
