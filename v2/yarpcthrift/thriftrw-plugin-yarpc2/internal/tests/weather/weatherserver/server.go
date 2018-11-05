// Code generated by thriftrw-plugin-yarpc
// @generated

package weatherserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcthrift"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/weather"
)

// Interface is the server-side interface for the Weather service.
type Interface interface {
	Check(
		ctx context.Context,
	) (string, error)
}

// New prepares an implementation of the Weather service for
// registration.
//
// 	handler := WeatherHandler{}
// 	dispatcher.Register(weatherserver.New(handler))
func New(impl Interface, opts ...yarpcthrift.RegisterOption) []yarpc.TransportProcedure {
	h := handler{impl}
	service := yarpcthrift.Service{
		Name: "Weather",
		Methods: []yarpcthrift.Method{

			yarpcthrift.Method{
				Name:         "check",
				Handler:      yarpcthrift.Handler(h.Check),
				Signature:    "Check() (string)",
				ThriftModule: weather.ThriftModule,
			},
		},
	}

	procedures := make([]yarpc.TransportProcedure, 0, 1)
	procedures = append(procedures, yarpcthrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }

func (h handler) Check(ctx context.Context, body wire.Value) (yarpcthrift.Response, error) {
	var args weather.Weather_Check_Args
	if err := args.FromWire(body); err != nil {
		return yarpcthrift.Response{}, err
	}

	success, err := h.impl.Check(ctx)

	appErr := err
	result, err := weather.Weather_Check_Helper.WrapResponse(success, err)

	var response yarpcthrift.Response
	if err == nil {
		response.ApplicationError = appErr
		response.Body = result
	}
	return response, err
}
